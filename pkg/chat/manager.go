// Package chat provides chat channel integration using omnichat.
package chat

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/plexusone/omnichat/provider"
	"github.com/plexusone/omnichat/providers/discord"
	"github.com/plexusone/omnichat/providers/telegram"

	"github.com/plexusone/agentcomms/pkg/config"
)

// ChatSession represents an active chat session.
type ChatSession struct {
	ID           string
	ProviderName string
	ChatID       string
	StartTime    time.Time
	Messages     []provider.IncomingMessage
	mu           sync.RWMutex
}

// AddMessage adds a message to the session.
func (cs *ChatSession) AddMessage(msg provider.IncomingMessage) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.Messages = append(cs.Messages, msg)
}

// RecentMessages returns the last n messages.
func (cs *ChatSession) RecentMessages(n int) []provider.IncomingMessage {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	if len(cs.Messages) <= n {
		return cs.Messages
	}
	return cs.Messages[len(cs.Messages)-n:]
}

// Manager orchestrates chat channels using the omnichat stack.
type Manager struct {
	config   *config.Config
	router   *provider.Router
	logger   *slog.Logger
	sessions map[string]*ChatSession
	mu       sync.RWMutex

	// Session counter for generating IDs
	sessionCounter int
	counterMu      sync.Mutex
}

// New creates a new chat manager.
func New(cfg *config.Config, logger *slog.Logger) (*Manager, error) {
	if logger == nil {
		logger = slog.Default()
	}

	m := &Manager{
		config:   cfg,
		router:   provider.NewRouter(logger),
		logger:   logger,
		sessions: make(map[string]*ChatSession),
	}

	return m, nil
}

// Initialize sets up the chat providers based on configuration.
func (m *Manager) Initialize(ctx context.Context) error {
	// Register Discord if enabled
	if m.config.DiscordEnabled {
		discordProvider, err := discord.New(discord.Config{
			Token:   m.config.DiscordToken,
			GuildID: m.config.DiscordGuildID,
			Logger:  m.logger,
		})
		if err != nil {
			return fmt.Errorf("failed to create Discord provider: %w", err)
		}
		m.router.Register(discordProvider)
		m.logger.Info("Discord provider registered")
	}

	// Register Telegram if enabled
	if m.config.TelegramEnabled {
		telegramProvider, err := telegram.New(telegram.Config{
			Token:  m.config.TelegramToken,
			Logger: m.logger,
		})
		if err != nil {
			return fmt.Errorf("failed to create Telegram provider: %w", err)
		}
		m.router.Register(telegramProvider)
		m.logger.Info("Telegram provider registered")
	}

	// WhatsApp requires more complex setup (device pairing)
	// For now, we log if it's enabled but note it requires additional setup
	if m.config.WhatsAppEnabled {
		m.logger.Info("WhatsApp enabled - requires device pairing via QR code")
		// WhatsApp integration would go here using whatsmeow
		// This requires interactive QR code scanning for first-time setup
	}

	// Set up message handler
	m.router.OnMessage(provider.All(), m.handleIncomingMessage)

	// Connect all registered providers
	if err := m.router.ConnectAll(ctx); err != nil {
		return fmt.Errorf("failed to connect providers: %w", err)
	}

	return nil
}

// handleIncomingMessage handles incoming chat messages.
func (m *Manager) handleIncomingMessage(ctx context.Context, msg provider.IncomingMessage) error {
	// Get or create session
	sessionKey := fmt.Sprintf("%s:%s", msg.ProviderName, msg.ChatID)

	m.mu.Lock()
	session, ok := m.sessions[sessionKey]
	if !ok {
		session = &ChatSession{
			ID:           m.generateSessionID(),
			ProviderName: msg.ProviderName,
			ChatID:       msg.ChatID,
			StartTime:    time.Now(),
		}
		m.sessions[sessionKey] = session
	}
	m.mu.Unlock()

	session.AddMessage(msg)

	m.logger.Info("chat message received",
		"provider", msg.ProviderName,
		"chat_id", msg.ChatID,
		"from", msg.SenderName,
		"content_length", len(msg.Content),
	)

	return nil
}

// generateSessionID generates a unique session ID.
func (m *Manager) generateSessionID() string {
	m.counterMu.Lock()
	defer m.counterMu.Unlock()
	m.sessionCounter++
	return fmt.Sprintf("chat-%d-%d", m.sessionCounter, time.Now().Unix())
}

// SendMessage sends a message to a chat channel.
func (m *Manager) SendMessage(ctx context.Context, providerName, chatID, content string) error {
	return m.router.Send(ctx, providerName, chatID, provider.OutgoingMessage{
		Content: content,
	})
}

// SendMessageWithReply sends a message as a reply.
func (m *Manager) SendMessageWithReply(ctx context.Context, providerName, chatID, content, replyTo string) error {
	return m.router.Send(ctx, providerName, chatID, provider.OutgoingMessage{
		Content: content,
		ReplyTo: replyTo,
	})
}

// ListChannels returns all available chat channels.
func (m *Manager) ListChannels() []ChannelInfo {
	providers := m.router.ListProviders()
	channels := make([]ChannelInfo, 0, len(providers))

	for _, name := range providers {
		channels = append(channels, ChannelInfo{
			ProviderName: name,
			Status:       "connected",
		})
	}

	return channels
}

// ChannelInfo describes a chat channel.
type ChannelInfo struct {
	ProviderName string `json:"provider_name"`
	Status       string `json:"status"`
}

// GetMessages returns recent messages from a session.
func (m *Manager) GetMessages(providerName, chatID string, limit int) ([]MessageInfo, error) {
	sessionKey := fmt.Sprintf("%s:%s", providerName, chatID)

	m.mu.RLock()
	session, ok := m.sessions[sessionKey]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no session found for %s", sessionKey)
	}

	recent := session.RecentMessages(limit)
	messages := make([]MessageInfo, len(recent))
	for i, msg := range recent {
		messages[i] = MessageInfo{
			ID:         msg.ID,
			SenderID:   msg.SenderID,
			SenderName: msg.SenderName,
			Content:    msg.Content,
			Timestamp:  msg.Timestamp,
		}
	}

	return messages, nil
}

// MessageInfo describes a chat message.
type MessageInfo struct {
	ID         string    `json:"id"`
	SenderID   string    `json:"sender_id"`
	SenderName string    `json:"sender_name"`
	Content    string    `json:"content"`
	Timestamp  time.Time `json:"timestamp"`
}

// GetSession returns a chat session by provider and chat ID.
func (m *Manager) GetSession(providerName, chatID string) *ChatSession {
	sessionKey := fmt.Sprintf("%s:%s", providerName, chatID)
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[sessionKey]
}

// Close shuts down the chat manager.
func (m *Manager) Close() error {
	ctx := context.Background()
	return m.router.DisconnectAll(ctx)
}
