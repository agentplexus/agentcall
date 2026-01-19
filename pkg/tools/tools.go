// Package tools defines the MCP tools for agentcall.
package tools

import (
	"context"
	"fmt"

	mcpkit "github.com/agentplexus/mcpkit/runtime"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/agentplexus/agentcall/pkg/callmanager"
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

// RegisterTools registers all MCP tools with the runtime.
func RegisterTools(rt *mcpkit.Runtime, manager *callmanager.Manager) {
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
