package acpadapter

import (
	"encoding/json"
	"fmt"

	"github.com/lsongdev/miya-agents/acp"
)

type ContentChunk struct {
	Type      string `json:"type"`
	Content   string `json:"content,omitempty"`
	Thought   string `json:"thought,omitempty"`
	Data      string `json:"data,omitempty"`
	Mime      string `json:"mime,omitempty"`
	URI       string `json:"uri,omitempty"`
	Name      string `json:"name,omitempty"`
	Size      int    `json:"size,omitempty"`
	MessageID string `json:"messageId,omitempty"`
}

type Event struct {
	Type    string                 `json:"type"`
	Content *ContentChunk          `json:"content,omitempty"`
	Tool    *acp.ToolCall          `json:"tool,omitempty"`
	Plan    *acp.Plan              `json:"plan,omitempty"`
	Usage   *acp.UsageUpdate       `json:"usage,omitempty"`
	Mode    *acp.CurrentModeUpdate `json:"mode,omitempty"`
	Info    *acp.SessionInfoUpdate `json:"info,omitempty"`
	Raw     json.RawMessage        `json:"raw,omitempty"`
}

func ParseUpdate(raw json.RawMessage) (*Event, error) {
	var disc struct {
		SessionUpdate string `json:"sessionUpdate"`
	}
	if err := json.Unmarshal(raw, &disc); err != nil {
		return nil, fmt.Errorf("acp update discriminator: %w", err)
	}
	if disc.SessionUpdate == "" {
		return nil, fmt.Errorf("acp update missing sessionUpdate")
	}

	e := &Event{Type: disc.SessionUpdate, Raw: cloneRaw(raw)}

	switch disc.SessionUpdate {
	case "user_message_chunk", "agent_message_chunk":
		var u struct {
			Content   acp.ContentBlock `json:"content"`
			MessageID *string          `json:"messageId,omitempty"`
		}
		if err := json.Unmarshal(raw, &u); err != nil {
			return nil, fmt.Errorf("%s: %w", disc.SessionUpdate, err)
		}
		e.Content = &ContentChunk{Type: u.Content.Type}
		if u.MessageID != nil {
			e.Content.MessageID = *u.MessageID
		}
		switch u.Content.Type {
		case "text":
			e.Content.Content = u.Content.Text
		case "image", "audio", "resource", "resource_link":
			e.Content.Data = u.Content.Data
			e.Content.Mime = u.Content.MimeType
			if u.Content.URI != nil {
				e.Content.URI = *u.Content.URI
			}
			e.Content.Name = u.Content.Name
			if u.Content.Size != nil {
				e.Content.Size = *u.Content.Size
			}
			if u.Content.Text != "" {
				e.Content.Content = u.Content.Text
			}
		}
	case "agent_thought_chunk":
		var u struct {
			Thought string           `json:"thought,omitempty"`
			Content acp.ContentBlock `json:"content,omitempty"`
		}
		if err := json.Unmarshal(raw, &u); err != nil {
			return nil, fmt.Errorf("agent_thought_chunk: %w", err)
		}
		text := u.Thought
		if text == "" {
			text = u.Content.Text
		}
		e.Content = &ContentChunk{Type: "text", Thought: text}
	case "tool_call":
		var u acp.ToolCall
		if err := json.Unmarshal(raw, &u); err != nil {
			return nil, fmt.Errorf("tool_call: %w", err)
		}
		if u.ToolCallID == "" {
			var wrap struct {
				ToolCall *acp.ToolCall `json:"toolCall"`
			}
			if json.Unmarshal(raw, &wrap) == nil && wrap.ToolCall != nil {
				u = *wrap.ToolCall
			}
		}
		tc := u
		e.Tool = &tc
	case "tool_call_update":
		var u acp.ToolCallUpdate
		if err := json.Unmarshal(raw, &u); err != nil {
			return nil, fmt.Errorf("tool_call_update: %w", err)
		}
		if u.ToolCallID == "" {
			var wrap struct {
				Update *acp.ToolCallUpdate `json:"toolCallUpdate"`
			}
			if json.Unmarshal(raw, &wrap) == nil && wrap.Update != nil {
				u = *wrap.Update
			}
		}
		tc := &acp.ToolCall{
			ToolCallID: u.ToolCallID,
			Content:    u.Content,
			Locations:  u.Locations,
			RawInput:   u.RawInput,
			RawOutput:  u.RawOutput,
		}
		if u.Title != nil {
			tc.Title = *u.Title
		}
		if u.Kind != nil {
			tc.Kind = *u.Kind
		}
		if u.Status != nil {
			tc.Status = *u.Status
		}
		e.Tool = tc
	case "plan":
		var u struct {
			Entries []acp.PlanEntry `json:"entries,omitempty"`
			Plan    *acp.Plan       `json:"plan,omitempty"`
		}
		if err := json.Unmarshal(raw, &u); err != nil {
			return nil, fmt.Errorf("plan: %w", err)
		}
		if u.Plan != nil {
			e.Plan = u.Plan
		} else if len(u.Entries) > 0 {
			e.Plan = &acp.Plan{Entries: u.Entries}
		}
	case "usage_update":
		var u struct {
			Usage *acp.UsageUpdate `json:"usage,omitempty"`
			Size  *uint64          `json:"size,omitempty"`
			Used  *uint64          `json:"used,omitempty"`
		}
		if err := json.Unmarshal(raw, &u); err != nil {
			return nil, fmt.Errorf("usage_update: %w", err)
		}
		if u.Usage != nil {
			e.Usage = u.Usage
		} else if u.Size != nil || u.Used != nil {
			e.Usage = &acp.UsageUpdate{}
			if u.Size != nil {
				e.Usage.Size = *u.Size
			}
			if u.Used != nil {
				e.Usage.Used = *u.Used
			}
		}
	case "current_mode_update":
		var u struct {
			CurrentMode   *acp.CurrentModeUpdate `json:"currentMode,omitempty"`
			CurrentModeID *acp.SessionModeID     `json:"currentModeId,omitempty"`
		}
		if err := json.Unmarshal(raw, &u); err != nil {
			return nil, fmt.Errorf("current_mode_update: %w", err)
		}
		if u.CurrentMode != nil {
			e.Mode = u.CurrentMode
		} else if u.CurrentModeID != nil {
			e.Mode = &acp.CurrentModeUpdate{CurrentModeID: *u.CurrentModeID}
		}
	case "session_info_update":
		var u struct {
			SessionInfo *acp.SessionInfoUpdate `json:"sessionInfo,omitempty"`
			Title       *string                `json:"title,omitempty"`
			UpdatedAt   *string                `json:"updatedAt,omitempty"`
		}
		if err := json.Unmarshal(raw, &u); err != nil {
			return nil, fmt.Errorf("session_info_update: %w", err)
		}
		if u.SessionInfo != nil {
			e.Info = u.SessionInfo
		} else if u.Title != nil || u.UpdatedAt != nil {
			e.Info = &acp.SessionInfoUpdate{Title: u.Title, UpdatedAt: u.UpdatedAt}
		}
	}

	return e, nil
}

func cloneRaw(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return nil
	}
	out := make([]byte, len(raw))
	copy(out, raw)
	return out
}
