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

	chatIDs := b.getUserChatIDs(c.Query().Sender.ID)
	var songs []model.Song
	var err error

	if len(chatIDs) > 0 {
		songs, err = b.store.SearchSongsInChats(ctx, chatIDs, query, 20)
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
