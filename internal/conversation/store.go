package conversation

import (
	"fmt"
	"sync"
	"time"

	"wails-app/internal/acpadapter"

	"github.com/lsongdev/miya-agents/acp"
)

type Store struct {
	mu            sync.Mutex
	conversations map[string]*Conversation
	nextID        uint64
	now           func() time.Time
}

func NewStore() *Store {
	return &Store{
		conversations: make(map[string]*Conversation),
		now:           time.Now,
	}
}

func (s *Store) RegisterSession(sessionID, cwd string) Snapshot {
	return s.RegisterSessionWithACP(sessionID, sessionID, cwd)
}

func (s *Store) RegisterSessionWithACP(conversationID, acpSessionID, cwd string) Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv := s.ensureConversationLocked(conversationID)
	if acpSessionID != "" {
		conv.ACPSessionID = acpSessionID
	}
	if cwd != "" {
		conv.Cwd = cwd
	}
	conv.UpdatedAt = s.nowString()
	return Snapshot{Conversation: cloneConversation(*conv), EventType: "conversation_registered"}
}

func (s *Store) ResetSessionWithACP(conversationID, acpSessionID, cwd string) Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.nowString()
	conv := &Conversation{
		ID:           conversationID,
		ACPSessionID: acpSessionID,
		Cwd:          cwd,
		Source:       Source{Type: "desktop"},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.conversations[conversationID] = conv
	return Snapshot{Conversation: cloneConversation(*conv), EventType: "conversation_reset"}
}

func (s *Store) Snapshot(sessionID string) (Snapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, ok := s.conversations[sessionID]
	if !ok {
		return Snapshot{}, false
	}
	return Snapshot{Conversation: cloneConversation(*conv), EventType: "snapshot"}, true
}

func (s *Store) AddLocalUserMessage(sessionID, text string) Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv := s.ensureConversationLocked(sessionID)
	now := s.nowString()
	s.closeStreamingLocked(conv)
	msg := Message{
		ID:             s.nextIDStringLocked("msg"),
		ConversationID: conv.ID,
		Role:           RoleUser,
		Status:         MessageComplete,
		CreatedAt:      now,
		UpdatedAt:      now,
		Blocks: []Block{
			{
				ID:      s.nextIDStringLocked("block"),
				Type:    BlockText,
				Content: text,
			},
		},
	}
	conv.Messages = append(conv.Messages, msg)
	conv.UpdatedAt = now
	return Snapshot{Conversation: cloneConversation(*conv), EventType: "message_added"}
}

func (s *Store) CompleteStreaming(sessionID string) (Snapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, ok := s.conversations[sessionID]
	if !ok {
		return Snapshot{}, false
	}
	changed := s.closeStreamingLocked(conv)
	conv.UpdatedAt = s.nowString()
	if !changed {
		return Snapshot{Conversation: cloneConversation(*conv), EventType: "snapshot"}, true
	}
	return Snapshot{Conversation: cloneConversation(*conv), EventType: "message_completed"}, true
}

func (s *Store) ApplyACPEvent(sessionID string, event *acpadapter.Event) Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv := s.ensureConversationLocked(sessionID)
	now := s.nowString()

	switch event.Type {
	case "user_message_chunk", "agent_message_chunk":
		role := RoleAssistant
		blockType := BlockMarkdown
		if event.Type == "user_message_chunk" {
			role = RoleUser
			blockType = BlockText
		}
		s.applyContentChunkLocked(conv, role, blockType, event, now)
	case "agent_thought_chunk":
		msg := s.currentAssistantMessageLocked(conv, now)
		s.appendOrMergeTextBlockLocked(msg, BlockThought, event.Content.Thought, event.Raw)
	case "tool_call":
		if event.Tool != nil {
			msg := s.currentAssistantMessageLocked(conv, now)
			s.upsertToolBlockLocked(msg, event.Tool, event.Raw)
		}
	case "tool_call_update":
		if event.Tool != nil {
			s.updateToolBlockLocked(conv, event.Tool, event.Raw, now)
		}
	case "plan":
		if event.Plan != nil {
			msg := s.currentAssistantMessageLocked(conv, now)
			s.upsertPlanBlockLocked(msg, event.Plan, event.Raw)
		}
	case "usage_update":
		conv.Usage = event.Usage
	case "current_mode_update":
		conv.Mode = event.Mode
	case "session_info_update":
		if event.Info != nil {
			if event.Info.Title != nil {
				conv.Title = *event.Info.Title
			}
		}
	}

	conv.UpdatedAt = now
	return Snapshot{Conversation: cloneConversation(*conv), EventType: event.Type}
}

func (s *Store) applyContentChunkLocked(conv *Conversation, role, blockType string, event *acpadapter.Event, now string) {
	if event.Content == nil {
		return
	}

	msg := s.messageForChunkLocked(conv, role, event.Content.MessageID, now)
	switch event.Content.Type {
	case "text":
		s.appendOrMergeTextBlockLocked(msg, blockType, event.Content.Content, event.Raw)
	case "image":
		msg.Blocks = append(msg.Blocks, Block{
			ID:   s.nextIDStringLocked("block"),
			Type: BlockImage,
			Data: event.Content.Data,
			Mime: event.Content.Mime,
			Raw:  cloneRaw(event.Raw),
		})
	case "audio":
		msg.Blocks = append(msg.Blocks, Block{
			ID:   s.nextIDStringLocked("block"),
			Type: BlockAudio,
			Data: event.Content.Data,
			Mime: event.Content.Mime,
			Raw:  cloneRaw(event.Raw),
		})
	}
	msg.UpdatedAt = now
}

func (s *Store) messageForChunkLocked(conv *Conversation, role, protocolMessageID string, now string) *Message {
	if len(conv.Messages) > 0 {
		last := &conv.Messages[len(conv.Messages)-1]
		if last.Role == role && last.Status == MessageStreaming {
			if protocolMessageID == "" && last.ProtocolMessageID == "" {
				return last
			}
			if protocolMessageID != "" && last.ProtocolMessageID == protocolMessageID {
				return last
			}
		}
	}

	s.closeStreamingLocked(conv)
	msg := Message{
		ID:                s.nextIDStringLocked("msg"),
		ConversationID:    conv.ID,
		Role:              role,
		ProtocolMessageID: protocolMessageID,
		Status:            MessageStreaming,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	conv.Messages = append(conv.Messages, msg)
	return &conv.Messages[len(conv.Messages)-1]
}

func (s *Store) currentAssistantMessageLocked(conv *Conversation, now string) *Message {
	if len(conv.Messages) > 0 {
		last := &conv.Messages[len(conv.Messages)-1]
		if last.Role == RoleAssistant && last.Status == MessageStreaming {
			return last
		}
	}
	s.closeStreamingLocked(conv)
	msg := Message{
		ID:             s.nextIDStringLocked("msg"),
		ConversationID: conv.ID,
		Role:           RoleAssistant,
		Status:         MessageStreaming,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	conv.Messages = append(conv.Messages, msg)
	return &conv.Messages[len(conv.Messages)-1]
}

func (s *Store) appendOrMergeTextBlockLocked(msg *Message, blockType, text string, raw []byte) {
	if text == "" {
		return
	}
	if len(msg.Blocks) > 0 {
		last := &msg.Blocks[len(msg.Blocks)-1]
		if last.Type == blockType {
			last.Content += text
			last.Raw = cloneRaw(raw)
			return
		}
	}
	msg.Blocks = append(msg.Blocks, Block{
		ID:      s.nextIDStringLocked("block"),
		Type:    blockType,
		Content: text,
		Raw:     cloneRaw(raw),
	})
}

func (s *Store) upsertToolBlockLocked(msg *Message, tool *acp.ToolCall, raw []byte) {
	for i := range msg.Blocks {
		block := &msg.Blocks[i]
		if block.Type == BlockToolCall && block.Tool != nil && block.Tool.ToolCallID == tool.ToolCallID {
			block.Tool = mergeTool(block.Tool, tool)
			block.Raw = cloneRaw(raw)
			return
		}
	}
	msg.Blocks = append(msg.Blocks, Block{
		ID:   s.nextIDStringLocked("block"),
		Type: BlockToolCall,
		Tool: cloneTool(tool),
		Raw:  cloneRaw(raw),
	})
}

func (s *Store) updateToolBlockLocked(conv *Conversation, tool *acp.ToolCall, raw []byte, now string) {
	for mi := range conv.Messages {
		msg := &conv.Messages[mi]
		for bi := range msg.Blocks {
			block := &msg.Blocks[bi]
			if block.Type == BlockToolCall && block.Tool != nil && block.Tool.ToolCallID == tool.ToolCallID {
				block.Tool = mergeTool(block.Tool, tool)
				block.Raw = cloneRaw(raw)
				msg.UpdatedAt = now
				return
			}
		}
	}
	msg := s.currentAssistantMessageLocked(conv, now)
	s.upsertToolBlockLocked(msg, tool, raw)
}

func (s *Store) upsertPlanBlockLocked(msg *Message, plan *acp.Plan, raw []byte) {
	for i := range msg.Blocks {
		block := &msg.Blocks[i]
		if block.Type == BlockPlan {
			block.Plan = plan
			block.Raw = cloneRaw(raw)
			return
		}
	}
	msg.Blocks = append(msg.Blocks, Block{
		ID:   s.nextIDStringLocked("block"),
		Type: BlockPlan,
		Plan: plan,
		Raw:  cloneRaw(raw),
	})
}

func (s *Store) closeStreamingLocked(conv *Conversation) bool {
	changed := false
	for i := range conv.Messages {
		if conv.Messages[i].Status == MessageStreaming {
			conv.Messages[i].Status = MessageComplete
			conv.Messages[i].UpdatedAt = s.nowString()
			changed = true
		}
	}
	return changed
}

func (s *Store) ensureConversationLocked(sessionID string) *Conversation {
	if conv, ok := s.conversations[sessionID]; ok {
		return conv
	}
	now := s.nowString()
	conv := &Conversation{
		ID:           sessionID,
		ACPSessionID: sessionID,
		Source:       Source{Type: "desktop"},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.conversations[sessionID] = conv
	return conv
}

func (s *Store) nowString() string {
	return s.now().Format(time.RFC3339Nano)
}

func (s *Store) nextIDStringLocked(prefix string) string {
	s.nextID++
	return fmt.Sprintf("%s-%d", prefix, s.nextID)
}

func cloneConversation(conv Conversation) Conversation {
	conv.Messages = append([]Message(nil), conv.Messages...)
	for i := range conv.Messages {
		conv.Messages[i].Blocks = append([]Block(nil), conv.Messages[i].Blocks...)
	}
	return conv
}

func cloneRaw(raw []byte) []byte {
	if raw == nil {
		return nil
	}
	out := make([]byte, len(raw))
	copy(out, raw)
	return out
}

func cloneTool(tool *acp.ToolCall) *acp.ToolCall {
	if tool == nil {
		return nil
	}
	cp := *tool
	cp.Content = append([]acp.ToolCallContent(nil), tool.Content...)
	cp.Locations = append([]acp.ToolCallLocation(nil), tool.Locations...)
	cp.RawInput = cloneRaw(tool.RawInput)
	cp.RawOutput = cloneRaw(tool.RawOutput)
	return &cp
}

func mergeTool(existing, update *acp.ToolCall) *acp.ToolCall {
	merged := cloneTool(existing)
	if merged == nil {
		return cloneTool(update)
	}
	if update.Title != "" {
		merged.Title = update.Title
	}
	if update.Kind != "" {
		merged.Kind = update.Kind
	}
	if update.Status != "" {
		merged.Status = update.Status
	}
	if len(update.Content) > 0 {
		merged.Content = append([]acp.ToolCallContent(nil), update.Content...)
	}
	if len(update.Locations) > 0 {
		merged.Locations = append([]acp.ToolCallLocation(nil), update.Locations...)
	}
	if len(update.RawInput) > 0 {
		merged.RawInput = cloneRaw(update.RawInput)
	}
	if len(update.RawOutput) > 0 {
		merged.RawOutput = cloneRaw(update.RawOutput)
	}
	return merged
}
