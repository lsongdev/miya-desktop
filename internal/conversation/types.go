package conversation

import (
	"encoding/json"

	"github.com/lsongdev/miya-agents/acp"
)

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"

	MessageStreaming = "streaming"
	MessageComplete  = "complete"
	MessageFailed    = "failed"

	BlockText     = "text"
	BlockMarkdown = "markdown"
	BlockThought  = "thought"
	BlockToolCall = "tool_call"
	BlockPlan     = "plan"
	BlockImage    = "image"
	BlockAudio    = "audio"
	BlockResource = "resource"
	BlockError    = "error"
)

type Conversation struct {
	ID           string                 `json:"id"`
	ACPSessionID string                 `json:"acpSessionId"`
	RuntimeID    string                 `json:"runtimeId,omitempty"`
	Title        string                 `json:"title,omitempty"`
	Cwd          string                 `json:"cwd,omitempty"`
	Model        string                 `json:"model,omitempty"`
	Source       Source                 `json:"source"`
	Messages     []Message              `json:"messages"`
	Usage        *acp.UsageUpdate       `json:"usage,omitempty"`
	Mode         *acp.CurrentModeUpdate `json:"mode,omitempty"`
	UpdatedAt    string                 `json:"updatedAt"`
	CreatedAt    string                 `json:"createdAt"`
}

type Source struct {
	Type      string `json:"type"`
	Channel   string `json:"channel,omitempty"`
	AccountID string `json:"accountId,omitempty"`
	ThreadID  string `json:"threadId,omitempty"`
}

type Message struct {
	ID                string  `json:"id"`
	ConversationID    string  `json:"conversationId"`
	Role              string  `json:"role"`
	ProtocolMessageID string  `json:"protocolMessageId,omitempty"`
	Blocks            []Block `json:"blocks"`
	Status            string  `json:"status"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
}

type Block struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Content string          `json:"content,omitempty"`
	Data    string          `json:"data,omitempty"`
	Mime    string          `json:"mime,omitempty"`
	URI     string          `json:"uri,omitempty"`
	Name    string          `json:"name,omitempty"`
	Size    int             `json:"size,omitempty"`
	Tool    *acp.ToolCall   `json:"tool,omitempty"`
	Plan    *acp.Plan       `json:"plan,omitempty"`
	Raw     json.RawMessage `json:"raw,omitempty"`
}

type Snapshot struct {
	Conversation Conversation `json:"conversation"`
	EventType    string       `json:"eventType"`
}
