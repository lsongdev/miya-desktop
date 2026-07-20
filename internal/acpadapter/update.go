package acpadapter

import (
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
}

// ParseSessionUpdate maps the ACP library's already-decoded update into the
// smaller event shape consumed by the conversation store and frontend.
func ParseSessionUpdate(update acp.SessionUpdate) (*Event, error) {
	if update.SessionUpdate == "" {
		return nil, fmt.Errorf("acp update missing sessionUpdate")
	}

	event := &Event{Type: update.SessionUpdate}
	switch update.SessionUpdate {
	case "user_message_chunk", "agent_message_chunk":
		event.Content = contentChunk(update.Content, update.MessageID)
	case "agent_thought_chunk":
		thought := update.Thought
		if thought == "" {
			thought = update.Content.Text
		}
		event.Content = &ContentChunk{Type: "text", Thought: thought}
	case "tool_call":
		event.Tool = update.ToolCall
	case "tool_call_update":
		event.Tool = toolCallFromUpdate(update.ToolCallUpdate)
	case "plan":
		event.Plan = update.Plan
	case "usage_update":
		event.Usage = update.Usage
	case "current_mode_update":
		event.Mode = update.CurrentMode
	case "session_info_update":
		event.Info = update.SessionInfo
	}
	return event, nil
}

func contentChunk(content acp.ContentBlock, messageID *acp.MessageID) *ContentChunk {
	chunk := &ContentChunk{
		Type:    content.Type,
		Content: content.Text,
		Data:    content.Data,
		Mime:    content.MimeType,
		Name:    content.Name,
	}
	if messageID != nil {
		chunk.MessageID = string(*messageID)
	}
	if content.URI != nil {
		chunk.URI = *content.URI
	}
	if content.Size != nil {
		chunk.Size = *content.Size
	}
	return chunk
}

func toolCallFromUpdate(update *acp.ToolCallUpdate) *acp.ToolCall {
	if update == nil {
		return nil
	}
	tool := &acp.ToolCall{
		ToolCallID: update.ToolCallID,
		Content:    update.Content,
		Locations:  update.Locations,
		RawInput:   update.RawInput,
		RawOutput:  update.RawOutput,
	}
	if update.Title != nil {
		tool.Title = *update.Title
	}
	if update.Kind != nil {
		tool.Kind = *update.Kind
	}
	if update.Status != nil {
		tool.Status = *update.Status
	}
	return tool
}
