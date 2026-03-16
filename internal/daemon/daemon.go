// Package daemon provides the AgentComms daemon that serves as the communication hub.
package daemon

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite" // SQLite driver (registers as "sqlite")

	"github.com/plexusone/omnichat/provider"
	"github.com/plexusone/omnichat/providers/discord"
	"github.com/plexusone/omnichat/providers/telegram"
	"github.com/plexusone/omnichat/providers/whatsapp"

	"github.com/plexusone/agentcomms/ent"
	"github.com/plexusone/agentcomms/internal/bridge"
	"github.com/plexusone/agentcomms/internal/router"
	"github.com/plexusone/agentcomms/internal/transport"
)

// Config holds daemon configuration.
type Config struct {
	// DataDir is the directory for storing data (default: ~/.agentcomms).
	DataDir string

	// SocketPath is the Unix socket path for IPC.
	SocketPath string

	// Logger is the structured logger.
	Logger *slog.Logger
}

// DefaultConfig returns the default daemon configuration.
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".agentcomms")

	return &Config{
		DataDir:    dataDir,
		SocketPath: filepath.Join(dataDir, "daemon.sock"),
		Logger:     slog.Default(),
	}
}

// Daemon is the AgentComms communication hub.
type Daemon struct {
	config       *Config
	daemonConfig *DaemonConfig
	client       *ent.Client
	router       *router.Router
	chatRouter   *provider.Router
	chat         *transport.ChatTransport
	server       *Server
	logger       *slog.Logger

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
}

// New creates a new daemon with the given configuration.
func New(cfg *Config) *Daemon {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &Daemon{
		config: cfg,
		logger: cfg.Logger,
	}
}

// Start starts the daemon.
func (d *Daemon) Start(ctx context.Context) error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return fmt.Errorf("daemon already running")
	}
	d.running = true
	d.mu.Unlock()

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(d.config.DataDir, 0700); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	d.logger.Info("starting daemon",
		"data_dir", d.config.DataDir,
		"socket", d.config.SocketPath,
	)

	// Load daemon config
	daemonCfg, err := LoadDaemonConfig(d.config.DataDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	d.daemonConfig = daemonCfg

	// Validate config
	if err := daemonCfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	d.logger.Info("config loaded",
		"agents", len(daemonCfg.Agents),
		"chat_enabled", daemonCfg.HasChatProviders(),
	)

	// Initialize database
	dbPath := filepath.Join(d.config.DataDir, "data.db")
	client, err := d.initDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	d.client = client

	d.logger.Info("database initialized", "path", dbPath)

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel

	// Run schema migration
	if err := d.client.Schema.Create(ctx); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Create router
	d.router = router.New(d.client, d.logger)
	d.logger.Info("router initialized")

	// Register agents from config
	if err := d.registerAgents(ctx); err != nil {
		return fmt.Errorf("failed to register agents: %w", err)
	}

	// Start chat transport if configured
	if daemonCfg.HasChatProviders() {
		if err := d.startChat(ctx); err != nil {
			return fmt.Errorf("failed to start chat transport: %w", err)
		}
	}

	// Start IPC server
	d.startServer(ctx)

	d.logger.Info("daemon started")

	// Block until context is cancelled
	<-ctx.Done()

	return d.shutdown()
}

// registerAgents creates adapters and registers agents from config.
func (d *Daemon) registerAgents(ctx context.Context) error {
	for _, agentCfg := range d.daemonConfig.Agents {
		var adapter bridge.Adapter
		var err error

		switch agentCfg.Type {
		case "tmux":
			pane := agentCfg.TmuxPane
			if pane == "" {
				pane = "0"
			}
			adapter, err = bridge.NewTmuxAdapter(bridge.TmuxConfig{
				Session: agentCfg.TmuxSession,
				Pane:    pane,
			})
			if err != nil {
				return fmt.Errorf("failed to create tmux adapter for %s: %w", agentCfg.ID, err)
			}

		default:
			return fmt.Errorf("unsupported agent type: %s", agentCfg.Type)
		}

		if err := d.router.RegisterAgent(ctx, agentCfg.ID, adapter); err != nil {
			return fmt.Errorf("failed to register agent %s: %w", agentCfg.ID, err)
		}

		d.logger.Info("registered agent",
			"agent_id", agentCfg.ID,
			"type", agentCfg.Type,
		)
	}

	return nil
}

// startChat initializes and starts the chat transport with omnichat.
func (d *Daemon) startChat(ctx context.Context) error {
	chatCfg := d.daemonConfig.Chat

	// Create omnichat router
	d.chatRouter = provider.NewRouter(d.logger)

	// Register Discord if configured
	if chatCfg.Discord != nil {
		p, err := discord.New(discord.Config{
			Token:   chatCfg.Discord.Token,
			GuildID: chatCfg.Discord.GuildID,
			Logger:  d.logger,
		})
		if err != nil {
			return fmt.Errorf("failed to create discord provider: %w", err)
		}
		d.chatRouter.Register(p)
		d.logger.Info("registered discord provider")
	}

	// Register Telegram if configured
	if chatCfg.Telegram != nil {
		p, err := telegram.New(telegram.Config{
			Token:  chatCfg.Telegram.Token,
			Logger: d.logger,
		})
		if err != nil {
			return fmt.Errorf("failed to create telegram provider: %w", err)
		}
		d.chatRouter.Register(p)
		d.logger.Info("registered telegram provider")
	}

	// Register WhatsApp if configured
	if chatCfg.WhatsApp != nil {
		p, err := whatsapp.New(whatsapp.Config{
			DBPath: chatCfg.WhatsApp.DBPath,
			Logger: d.logger,
			QRCallback: func(qr string) {
				// Log QR code for user to scan
				d.logger.Info("WhatsApp QR code ready - scan with WhatsApp mobile app",
					"qr", qr,
				)
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create whatsapp provider: %w", err)
		}
		d.chatRouter.Register(p)
		d.logger.Info("registered whatsapp provider")
	}

	// Create chat transport
	d.chat = transport.NewChatTransport(
		d.chatRouter,
		d.client,
		d.router,
		d.daemonConfig, // DaemonConfig implements ChannelResolver
		d.logger,
	)

	// Start in background goroutine
	go func() {
		if err := d.chat.Start(ctx); err != nil {
			d.logger.Error("chat transport error", "error", err)
		}
	}()

	return nil
}

// startServer initializes and starts the IPC server.
func (d *Daemon) startServer(ctx context.Context) {
	// Collect provider names
	var providers []string
	if d.chatRouter != nil {
		providers = d.chatRouter.ListProviders()
	}

	// ChatTransport implements ChatSender interface
	var chatSender ChatSender
	if d.chat != nil {
		chatSender = d.chat
	}

	d.server = NewServer(ServerConfig{
		SocketPath: d.config.SocketPath,
		Client:     d.client,
		Router:     d.router,
		DaemonCfg:  d.daemonConfig,
		ChatSender: chatSender,
		Providers:  providers,
		Logger:     d.logger,
	})

	// Start in background goroutine
	go func() {
		if err := d.server.Start(ctx); err != nil {
			d.logger.Error("server error", "error", err)
		}
	}()
}

// Stop stops the daemon.
func (d *Daemon) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running {
		return nil
	}

	if d.cancel != nil {
		d.cancel()
	}

	return nil
}

// shutdown performs cleanup when stopping.
func (d *Daemon) shutdown() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.logger.Info("shutting down daemon")

	// Stop server first
	if d.server != nil {
		if err := d.server.Stop(); err != nil {
			d.logger.Error("failed to stop server", "error", err)
		}
	}

	// Stop chat transport
	if d.chat != nil {
		if err := d.chat.Stop(); err != nil {
			d.logger.Error("failed to stop chat transport", "error", err)
		}
	}

	// Stop router
	if d.router != nil {
		d.router.Stop()
	}

	if d.client != nil {
		if err := d.client.Close(); err != nil {
			d.logger.Error("failed to close database", "error", err)
		}
	}

	d.running = false
	d.logger.Info("daemon stopped")

	return nil
}

// initDB initializes the SQLite database.
func (d *Daemon) initDB(dbPath string) (*ent.Client, error) {
	// SQLite connection string with WAL mode for better concurrency
	// modernc/sqlite uses "sqlite" as driver name
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create Ent driver from sql.DB
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))

	return client, nil
}

// Client returns the Ent client for database operations.
func (d *Daemon) Client() *ent.Client {
	return d.client
}

// Router returns the event router.
func (d *Daemon) Router() *router.Router {
	return d.router
}

// Chat returns the chat transport for outbound messages.
func (d *Daemon) Chat() *transport.ChatTransport {
	return d.chat
}
