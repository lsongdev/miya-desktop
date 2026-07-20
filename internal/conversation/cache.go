package conversation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const cacheSchemaVersion = 1

type Cache struct {
	dir string
}

type cacheEnvelope struct {
	Version      int          `json:"version"`
	CachedAt     string       `json:"cachedAt"`
	Conversation Conversation `json:"conversation"`
}

func NewCache(dir string) *Cache {
	return &Cache{dir: dir}
}

func (c *Cache) Load(conversationID string) (*Conversation, error) {
	data, err := os.ReadFile(c.path(conversationID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read conversation cache: %w", err)
	}
	var envelope cacheEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("decode conversation cache: %w", err)
	}
	if envelope.Version != cacheSchemaVersion || envelope.Conversation.ID != conversationID {
		return nil, nil
	}
	conversation := cacheConversation(envelope.Conversation)
	return &conversation, nil
}

func (c *Cache) Save(conversation Conversation) error {
	if conversation.ID == "" {
		return nil
	}
	if err := os.MkdirAll(c.dir, 0700); err != nil {
		return fmt.Errorf("create conversation cache: %w", err)
	}
	conversation = cacheConversation(conversation)
	data, err := json.Marshal(cacheEnvelope{
		Version:      cacheSchemaVersion,
		CachedAt:     time.Now().Format(time.RFC3339Nano),
		Conversation: conversation,
	})
	if err != nil {
		return fmt.Errorf("encode conversation cache: %w", err)
	}
	tmp, err := os.CreateTemp(c.dir, ".conversation-*.tmp")
	if err != nil {
		return fmt.Errorf("create conversation cache temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return fmt.Errorf("secure conversation cache: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write conversation cache: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close conversation cache: %w", err)
	}
	if err := os.Rename(tmpName, c.path(conversation.ID)); err != nil {
		return fmt.Errorf("replace conversation cache: %w", err)
	}
	return nil
}

func (c *Cache) Delete(conversationID string) error {
	if err := os.Remove(c.path(conversationID)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete conversation cache: %w", err)
	}
	return nil
}

func (c *Cache) path(conversationID string) string {
	sum := sha256.Sum256([]byte(conversationID))
	return filepath.Join(c.dir, hex.EncodeToString(sum[:])+".json")
}

func cacheConversation(conversation Conversation) Conversation {
	conversation = cloneConversation(conversation)
	for messageIndex := range conversation.Messages {
		for blockIndex := range conversation.Messages[messageIndex].Blocks {
			block := &conversation.Messages[messageIndex].Blocks[blockIndex]
			block.Raw = nil
		}
	}
	return conversation
}
