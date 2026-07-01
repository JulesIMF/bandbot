package bot

import (
	"sync"
	"time"

	tele "gopkg.in/telebot.v3"
)

const membershipTTL = 24 * time.Hour

type memberEntry struct {
	isMember  bool
	expiresAt time.Time
}

type MembershipCache struct {
	mu    sync.RWMutex
	cache map[int64]map[int64]memberEntry // userID -> chatID -> entry
	bot   *tele.Bot
}

func NewMembershipCache(bot *tele.Bot) *MembershipCache {
	return &MembershipCache{
		cache: make(map[int64]map[int64]memberEntry),
		bot:   bot,
	}
}

func (mc *MembershipCache) IsMember(userID, chatID int64) bool {
	mc.mu.RLock()
	if userChats, ok := mc.cache[userID]; ok {
		if entry, ok := userChats[chatID]; ok && time.Now().Before(entry.expiresAt) {
			mc.mu.RUnlock()
			return entry.isMember
		}
	}
	mc.mu.RUnlock()

	member, err := mc.bot.ChatMemberOf(&tele.Chat{ID: chatID}, &tele.User{ID: userID})
	isMember := err == nil && member != nil &&
		member.Role != tele.Left && member.Role != tele.Kicked

	mc.mu.Lock()
	if _, ok := mc.cache[userID]; !ok {
		mc.cache[userID] = make(map[int64]memberEntry)
	}
	mc.cache[userID][chatID] = memberEntry{
		isMember:  isMember,
		expiresAt: time.Now().Add(membershipTTL),
	}
	mc.mu.Unlock()

	return isMember
}

func (mc *MembershipCache) GetUserChats(userID int64, chatIDs []int64) []int64 {
	var result []int64
	for _, chatID := range chatIDs {
		if mc.IsMember(userID, chatID) {
			result = append(result, chatID)
		}
	}
	return result
}
