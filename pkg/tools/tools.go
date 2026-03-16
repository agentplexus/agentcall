// Package tools defines the MCP tools for agentcomms.
package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpkit "github.com/plexusone/mcpkit/runtime"

	"github.com/plexusone/agentcomms/pkg/chat"
	"github.com/plexusone/agentcomms/pkg/voice"
)

// InitiateCallInput is the input for the initiate_call tool.
type InitiateCallInput struct {
	Message string `json:"message"`
}

// InitiateCallOutput is the output of the initiate_call tool.
type InitiateCallOutput struct {
	CallID   string `json:"call_id"`
	Response string `json:"response"`
}

// ContinueCallInput is the input for the continue_call tool.
type ContinueCallInput struct {
	CallID  string `json:"call_id"`
	Message string `json:"message"`
}

// ContinueCallOutput is the output of the continue_call tool.
type ContinueCallOutput struct {
	Response string `json:"response"`
}

// SpeakToUserInput is the input for the speak_to_user tool.
type SpeakToUserInput struct {
	CallID  string `json:"call_id"`
	Message string `json:"message"`
}

// SpeakToUserOutput is the output of the speak_to_user tool.
type SpeakToUserOutput struct {
	Success bool `json:"success"`
}

// EndCallInput is the input for the end_call tool.
type EndCallInput struct {
	CallID  string `json:"call_id"`
	Message string `json:"message,omitempty"`
}

// EndCallOutput is the output of the end_call tool.
type EndCallOutput struct {
	DurationSeconds float64 `json:"duration_seconds"`
}

// SendMessageInput is the input for the send_message tool.
type SendMessageInput struct {
	Provider string `json:"provider"`
	ChatID   string `json:"chat_id"`
	Message  string `json:"message"`
	ReplyTo  string `json:"reply_to,omitempty"`
}

// SendMessageOutput is the output of the send_message tool.
type SendMessageOutput struct {
	Success bool `json:"success"`
}

// ListChannelsInput is the input for the list_channels tool.
type ListChannelsInput struct{}

// ListChannelsOutput is the output of the list_channels tool.
type ListChannelsOutput struct {
	Channels []chat.ChannelInfo `json:"channels"`
}

// GetMessagesInput is the input for the get_messages tool.
type GetMessagesInput struct {
	Provider string `json:"provider"`
	ChatID   string `json:"chat_id"`
	Limit    int    `json:"limit,omitempty"`
}

// GetMessagesOutput is the output of the get_messages tool.
type GetMessagesOutput struct {
	Messages []chat.MessageInfo `json:"messages"`
}

// RegisterVoiceTools registers voice-related MCP tools with the runtime.
func RegisterVoiceTools(rt *mcpkit.Runtime, manager *voice.Manager) {
	// initiate_call - Start a new call to the user
	mcpkit.AddTool(rt, &mcp.Tool{
		Name:        "initiate_call",
		Description: "Call the user on the phone to discuss something. Use this when you need to report task completion, request input, discuss decisions, or escalate blockers. The call will ring the user's phone, and when they answer, your message will be spoken. Then you'll receive their spoken response.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{
					"type":        "string",
					"description": "The message to speak to the user when they answer. Should be conversational and clear.",
				},
			},
			"required": []string{"message"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in InitiateCallInput) (*mcp.CallToolResult, InitiateCallOutput, error) {
		state, response, err := manager.InitiateCall(ctx, in.Message)
		if err != nil {
			return nil, InitiateCallOutput{}, fmt.Errorf("failed to initiate call: %w", err)
		}

		return nil, InitiateCallOutput{
			CallID:   state.ID,
			Response: response,
		}, nil
	})

	// continue_call - Continue an existing call with another message
	mcpkit.AddTool(rt, &mcp.Tool{
		Name:        "continue_call",
		Description: "Continue an active phone call by speaking another message and listening for the user's response. Use this for multi-turn conversations within the same call.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"call_id": map[string]any{
					"type":        "string",
					"description": "The ID of the active call (returned from initiate_call).",
				},
				"message": map[string]any{
					"type":        "string",
					"description": "The message to speak to the user.",
				},
			},
			"required": []string{"call_id", "message"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in ContinueCallInput) (*mcp.CallToolResult, ContinueCallOutput, error) {
		response, err := manager.ContinueCall(ctx, in.CallID, in.Message)
		if err != nil {
			return nil, ContinueCallOutput{}, fmt.Errorf("failed to continue call: %w", err)
		}

		return nil, ContinueCallOutput{
			Response: response,
		}, nil
	})

	// speak_to_user - Speak without waiting for response
	mcpkit.AddTool(rt, &mcp.Tool{
		Name:        "speak_to_user",
		Description: "Speak a message to the user without waiting for a response. Use this for acknowledgments before performing time-consuming operations, or for status updates during a call.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"call_id": map[string]any{
					"type":        "string",
					"description": "The ID of the active call.",
				},
				"message": map[string]any{
					"type":        "string",
					"description": "The message to speak to the user.",
				},
			},
			"required": []string{"call_id", "message"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in SpeakToUserInput) (*mcp.CallToolResult, SpeakToUserOutput, error) {
		err := manager.SpeakToUser(ctx, in.CallID, in.Message)
		if err != nil {
			return nil, SpeakToUserOutput{Success: false}, fmt.Errorf("failed to speak: %w", err)
		}

		return nil, SpeakToUserOutput{Success: true}, nil
	})

	// end_call - End the call with an optional final message
	mcpkit.AddTool(rt, &mcp.Tool{
		Name:        "end_call",
		Description: "End an active phone call. Optionally speak a final message before hanging up. The message will be spoken and then the call will be terminated.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"call_id": map[string]any{
					"type":        "string",
					"description": "The ID of the active call.",
				},
				"message": map[string]any{
					"type":        "string",
					"description": "Optional final message to speak before ending the call.",
				},
			},
			"required": []string{"call_id"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in EndCallInput) (*mcp.CallToolResult, EndCallOutput, error) {
		duration, err := manager.EndCall(ctx, in.CallID, in.Message)
		if err != nil {
			return nil, EndCallOutput{}, fmt.Errorf("failed to end call: %w", err)
		}

		return nil, EndCallOutput{
			DurationSeconds: duration.Seconds(),
		}, nil
	})
}

// RegisterChatTools registers chat-related MCP tools with the runtime.
func RegisterChatTools(rt *mcpkit.Runtime, manager *chat.Manager) {
	// send_message - Send a message to a chat channel
	mcpkit.AddTool(rt, &mcp.Tool{
		Name:        "send_message",
		Description: "Send a message to the user via a chat channel (Discord, Telegram, or WhatsApp). Use this for asynchronous communication when the user is not on a phone call.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"provider": map[string]any{
					"type":        "string",
					"description": "The chat provider to use: 'discord', 'telegram', or 'whatsapp'.",
					"enum":        []string{"discord", "telegram", "whatsapp"},
				},
				"chat_id": map[string]any{
					"type":        "string",
					"description": "The chat/channel ID to send the message to.",
				},
				"message": map[string]any{
					"type":        "string",
					"description": "The message content to send.",
				},
				"reply_to": map[string]any{
					"type":        "string",
					"description": "Optional message ID to reply to.",
				},
			},
			"required": []string{"provider", "chat_id", "message"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in SendMessageInput) (*mcp.CallToolResult, SendMessageOutput, error) {
		var err error
		if in.ReplyTo != "" {
			err = manager.SendMessageWithReply(ctx, in.Provider, in.ChatID, in.Message, in.ReplyTo)
		} else {
			err = manager.SendMessage(ctx, in.Provider, in.ChatID, in.Message)
		}
		if err != nil {
			return nil, SendMessageOutput{Success: false}, fmt.Errorf("failed to send message: %w", err)
		}

		return nil, SendMessageOutput{Success: true}, nil
	})

	// list_channels - List available chat channels
	mcpkit.AddTool(rt, &mcp.Tool{
		Name:        "list_channels",
		Description: "List all available chat channels and their connection status. Returns which messaging platforms are connected and ready to use.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in ListChannelsInput) (*mcp.CallToolResult, ListChannelsOutput, error) {
		channels := manager.ListChannels()
		return nil, ListChannelsOutput{Channels: channels}, nil
	})

	// get_messages - Get recent messages from a chat
	mcpkit.AddTool(rt, &mcp.Tool{
		Name:        "get_messages",
		Description: "Get recent messages from a chat conversation. Use this to see what the user has said in a chat channel.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"provider": map[string]any{
					"type":        "string",
					"description": "The chat provider: 'discord', 'telegram', or 'whatsapp'.",
					"enum":        []string{"discord", "telegram", "whatsapp"},
				},
				"chat_id": map[string]any{
					"type":        "string",
					"description": "The chat/channel ID to get messages from.",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Maximum number of messages to return (default: 10).",
					"default":     10,
				},
			},
			"required": []string{"provider", "chat_id"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in GetMessagesInput) (*mcp.CallToolResult, GetMessagesOutput, error) {
		limit := in.Limit
		if limit <= 0 {
			limit = 10
		}

		messages, err := manager.GetMessages(in.Provider, in.ChatID, limit)
		if err != nil {
			return nil, GetMessagesOutput{}, fmt.Errorf("failed to get messages: %w", err)
		}

		return nil, GetMessagesOutput{Messages: messages}, nil
	})
}

// RegisterTools registers all MCP tools (voice + chat + inbound) with the runtime.
// This is a convenience function that calls RegisterVoiceTools, RegisterChatTools, and RegisterInboundTools.
func RegisterTools(rt *mcpkit.Runtime, voiceManager *voice.Manager, chatManager *chat.Manager) {
	if voiceManager != nil {
		RegisterVoiceTools(rt, voiceManager)
	}
	if chatManager != nil {
		RegisterChatTools(rt, chatManager)
	}

	// Always register inbound tools - they check daemon status dynamically
	inboundManager := NewInboundManager(InboundConfig{})
	RegisterInboundTools(rt, inboundManager)
}
