package bot

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v3"
)

func (b *Bot) handleAllSongs(c tele.Context) error {
	if err := b.ensureChat(c); err != nil {
		return c.Send("Ошибка: " + err.Error())
	}

	ctx := context.Background()
	songs, err := b.store.ListSongs(ctx, c.Chat().ID)
	if err != nil {
		return c.Send("Ошибка: " + err.Error())
	}
	if len(songs) == 0 {
		return c.Send("Песен пока нет.")
	}

	text := "🎵 Все песни:\n\n" + RenderSongList(songs)

	rm := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, s := range songs {
		rows = append(rows, tele.Row{
			rm.Data(s.Name, "show_song", fmt.Sprintf("%d", s.ID)),
		})
	}
	rm.Inline(rows...)
	return c.Send(text, rm)
}

func (b *Bot) handleAllSetlists(c tele.Context) error {
	if err := b.ensureChat(c); err != nil {
		return c.Send("Ошибка: " + err.Error())
	}

	ctx := context.Background()
	setlists, err := b.store.ListSetlists(ctx, c.Chat().ID)
	if err != nil {
		return c.Send("Ошибка: " + err.Error())
	}
	if len(setlists) == 0 {
		return c.Send("Сетлистов пока нет.")
	}

	rm := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, sl := range setlists {
		rows = append(rows, tele.Row{
			rm.Data(sl.Name, "show_sl", fmt.Sprintf("%d", sl.ID)),
		})
	}
	rm.Inline(rows...)
	return c.Send("📋 Все сетлисты:", rm)
}
