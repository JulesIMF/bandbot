package bot

import (
	"sync"
	"time"
)

type PromptType int

const (
	PromptTempo PromptType = iota
	PromptRenameSong
	PromptResponsible
	PromptRenameSetlist
	PromptSetlistSongs
	PromptCreateSetlist
)

type PromptContext struct {
	Type       PromptType
	TargetID   int
	TargetName string
	ChatID     int64
	CreatedAt  time.Time
}

const promptTTL = 24 * time.Hour

type PromptStore struct {
	mu      sync.RWMutex
	prompts map[promptKey]PromptContext
}

type promptKey struct {
	ChatID    int64
	MessageID int
}

func NewPromptStore() *PromptStore {
	ps := &PromptStore{
		prompts: make(map[promptKey]PromptContext),
	}
	go ps.cleanupLoop()
	return ps
}

func (ps *PromptStore) Set(chatID int64, messageID int, ctx PromptContext) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ctx.CreatedAt = time.Now()
	ctx.ChatID = chatID
	ps.prompts[promptKey{chatID, messageID}] = ctx
}

func (ps *PromptStore) Get(chatID int64, messageID int) (PromptContext, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	ctx, ok := ps.prompts[promptKey{chatID, messageID}]
	if !ok || time.Since(ctx.CreatedAt) > promptTTL {
		return PromptContext{}, false
	}
	return ctx, true
}

func (ps *PromptStore) Delete(chatID int64, messageID int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	delete(ps.prompts, promptKey{chatID, messageID})
}

func (ps *PromptStore) cleanupLoop() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		ps.mu.Lock()
		now := time.Now()
		for k, v := range ps.prompts {
			if now.Sub(v.CreatedAt) > promptTTL {
				delete(ps.prompts, k)
			}
		}
		ps.mu.Unlock()
	}
}
