package daemon

import (
	"encoding/json"
	"testing"
)

func TestNewRequest(t *testing.T) {
	// Test without params
	req, err := NewRequest("1", MethodPing, nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	if req.ID != "1" {
		t.Errorf("expected ID '1', got %q", req.ID)
	}
	if req.Method != MethodPing {
		t.Errorf("expected method %q, got %q", MethodPing, req.Method)
	}
	if req.Params != nil {
		t.Errorf("expected nil params, got %v", req.Params)
	}

	// Test with params
	params := SendParams{AgentID: "test", Message: "hello"}
	req, err = NewRequest("2", MethodSend, params)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	var parsed SendParams
	if err := req.ParseParams(&parsed); err != nil {
		t.Errorf("ParseParams() error = %v", err)
	}
	if parsed.AgentID != "test" {
		t.Errorf("expected AgentID 'test', got %q", parsed.AgentID)
	}
	if parsed.Message != "hello" {
		t.Errorf("expected Message 'hello', got %q", parsed.Message)
	}
}

func TestNewResponse(t *testing.T) {
	// Test success response
	result := PingResult{Pong: true}
	resp, err := NewResponse("1", result)
	if err != nil {
		t.Fatalf("NewResponse() error = %v", err)
	}

	if resp.ID != "1" {
		t.Errorf("expected ID '1', got %q", resp.ID)
	}
	if resp.IsError() {
		t.Error("expected no error")
	}
	if resp.Err() != nil {
		t.Errorf("expected nil error, got %v", resp.Err())
	}

	var parsed PingResult
	if err := resp.ParseResult(&parsed); err != nil {
		t.Errorf("ParseResult() error = %v", err)
	}
	if !parsed.Pong {
		t.Error("expected Pong=true")
	}
}

func TestNewErrorResponse(t *testing.T) {
	resp := NewErrorResponse("1", ErrCodeNotFound, "agent not found")

	if resp.ID != "1" {
		t.Errorf("expected ID '1', got %q", resp.ID)
	}
	if !resp.IsError() {
		t.Error("expected error")
	}
	if resp.Error.Code != ErrCodeNotFound {
		t.Errorf("expected code %d, got %d", ErrCodeNotFound, resp.Error.Code)
	}
	if resp.Error.Message != "agent not found" {
		t.Errorf("expected message 'agent not found', got %q", resp.Error.Message)
	}

	err := resp.Err()
	if err == nil {
		t.Error("expected non-nil error")
	}
}

func TestRequestResponseJSON(t *testing.T) {
	// Test that Request and Response can be serialized/deserialized
	req, _ := NewRequest("123", MethodSend, SendParams{AgentID: "a1", Message: "hi"})

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal request error = %v", err)
	}

	var parsed Request
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal request error = %v", err)
	}

	if parsed.ID != req.ID {
		t.Errorf("expected ID %q, got %q", req.ID, parsed.ID)
	}
	if parsed.Method != req.Method {
		t.Errorf("expected method %q, got %q", req.Method, parsed.Method)
	}

	// Test Response
	resp, _ := NewResponse("123", SendResult{EventID: "evt_1", Delivered: true})

	data, err = json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal response error = %v", err)
	}

	var parsedResp Response
	if err := json.Unmarshal(data, &parsedResp); err != nil {
		t.Fatalf("Unmarshal response error = %v", err)
	}

	if parsedResp.ID != resp.ID {
		t.Errorf("expected ID %q, got %q", resp.ID, parsedResp.ID)
	}
}

func TestParseParamsNil(t *testing.T) {
	req := &Request{ID: "1", Method: MethodPing}

	// Should not error on nil params
	var params SendParams
	if err := req.ParseParams(&params); err != nil {
		t.Errorf("ParseParams() on nil should not error: %v", err)
	}
}

func TestParseResultNil(t *testing.T) {
	resp := &Response{ID: "1"}

	// Should not error on nil result
	var result PingResult
	if err := resp.ParseResult(&result); err != nil {
		t.Errorf("ParseResult() on nil should not error: %v", err)
	}
}
