package webhook

import (
	"encoding/json"
	"fmt"
)

// telnyxWebhookPayload represents a Telnyx webhook payload.
type telnyxWebhookPayload struct {
	ID            string
	EventType     string
	CallControlID string
	From          string
	To            string
	Text          string
	Raw           map[string]string
}

// parseTelnyxWebhook parses a Telnyx webhook JSON payload.
func parseTelnyxWebhook(body []byte) (*telnyxWebhookPayload, error) {
	var raw struct {
		Data struct {
			ID        string `json:"id"`
			EventType string `json:"event_type"`
			Payload   struct {
				// Common fields
				CallControlID string `json:"call_control_id"`
				Text          string `json:"text"`

				// From/To can be either objects (SMS) or strings (voice)
				// Use json.RawMessage to handle both cases
				From json.RawMessage `json:"from"`
				To   json.RawMessage `json:"to"`
			} `json:"payload"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}

	payload := &telnyxWebhookPayload{
		ID:            raw.Data.ID,
		EventType:     raw.Data.EventType,
		CallControlID: raw.Data.Payload.CallControlID,
		Text:          raw.Data.Payload.Text,
		Raw:           make(map[string]string),
	}

	// Extract From - try SMS structure (object) first, then voice (string)
	payload.From = extractPhoneNumber(raw.Data.Payload.From)

	// Extract To - try SMS structure (array) first, then voice (string)
	payload.To = extractToPhoneNumber(raw.Data.Payload.To)

	// Store raw data for debugging
	payload.Raw["id"] = payload.ID
	payload.Raw["event_type"] = payload.EventType
	payload.Raw["from"] = payload.From
	payload.Raw["to"] = payload.To

	return payload, nil
}

// extractPhoneNumber extracts a phone number from a json.RawMessage
// that can be either a string or an object with phone_number field.
func extractPhoneNumber(data json.RawMessage) string {
	if len(data) == 0 {
		return ""
	}

	// Try as string first (voice events)
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		return str
	}

	// Try as object with phone_number (SMS events)
	var obj struct {
		PhoneNumber string `json:"phone_number"`
	}
	if err := json.Unmarshal(data, &obj); err == nil {
		return obj.PhoneNumber
	}

	return ""
}

// extractToPhoneNumber extracts a phone number from a json.RawMessage
// that can be either a string or an array of objects with phone_number field.
func extractToPhoneNumber(data json.RawMessage) string {
	if len(data) == 0 {
		return ""
	}

	// Try as string first (voice events)
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		return str
	}

	// Try as array of objects (SMS events)
	var arr []struct {
		PhoneNumber string `json:"phone_number"`
	}
	if err := json.Unmarshal(data, &arr); err == nil && len(arr) > 0 {
		return arr[0].PhoneNumber
	}

	return ""
}
