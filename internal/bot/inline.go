package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/julesimf/bandbot/internal/model"
	tele "gopkg.in/telebot.v3"
)

func (b *Bot) handleInlineQuery(c tele.Context) error {
	query := strings.TrimSpace(c.Query().Text)

	ctx := context.Background()

	// Inline queries don't have a chat context, so we search across all chats
	// that the sender has interacted with. For simplicity, we'll use the query
	// to search songs globally for this bot instance.
	// A better approach would be to cache user->chat mappings, but for a band bot
	// used in a few chats this is acceptable.
	var songs []model.Song
	var err error

	if query == "" {
		songs, err = b.store.SearchSongs(ctx, 0, "", 20)
	} else {
		songs, err = b.store.SearchSongs(ctx, 0, query, 20)
	}

	if err != nil {
		return c.Answer(&tele.QueryResponse{})
	}

	var results tele.Results
	for i, s := range songs {
		var details []string
		if s.Tempo != nil {
			details = append(details, fmt.Sprintf("%d bpm", *s.Tempo))
		}
		if s.Key != nil {
			details = append(details, *s.Key)
		}

		desc := "Нет данных"
		if len(details) > 0 {
			desc = strings.Join(details, " · ")
		}

		result := &tele.ArticleResult{
			Title:       s.Name,
			Description: desc,
			Text:        fmt.Sprintf("/song %s", s.Name),
		}
		result.SetResultID(fmt.Sprintf("%d", i))
		results = append(results, result)
	}

	return c.Answer(&tele.QueryResponse{
		Results:   results,
		CacheTime: 5,
	})
}
