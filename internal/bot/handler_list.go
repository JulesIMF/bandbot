package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/julesimf/bandbot/internal/model"
	tele "gopkg.in/telebot.v3"
)

func (b *Bot) handleAllSongs(c tele.Context) error {
	if isPrivateChat(c) {
		return b.handleAllSongsPrivate(c)
	}

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
	origin := fmt.Sprintf("song_list|%d", c.Chat().ID)
	rm := songListKeyboard(songs, origin)
	return c.Send(text, rm)
}

func (b *Bot) handleAllSongsPrivate(c tele.Context) error {
	chatIDs := b.getUserChatIDs(c.Sender().ID)
	if len(chatIDs) == 0 {
		return c.Send("Вы не состоите ни в одной группе с этим ботом.")
	}

	ctx := context.Background()
	songs, err := b.store.ListSongsInChats(ctx, chatIDs)
	if err != nil {
		return c.Send("Ошибка: " + err.Error())
	}
	if len(songs) == 0 {
		return c.Send("Песен пока нет.")
	}

	chatNames := b.buildChatNameMap(ctx, chatIDs)
	grouped := groupSongsByChat(songs)

	var text strings.Builder
	text.WriteString("🎵 Все песни:\n")

	rm := &tele.ReplyMarkup{}
	var rows []tele.Row

	for chatID, chatSongs := range grouped {
		name := chatNames[chatID]
		if name == "" {
			name = fmt.Sprintf("Чат %d", chatID)
		}
		text.WriteString(fmt.Sprintf("\n<b>%s</b>\n", escapeHTML(name)))
		for _, s := range chatSongs {
			text.WriteString(FormatSongLine(&s) + "\n")
			rows = append(rows, tele.Row{
				rm.Data(s.Name, "show_song", fmt.Sprintf("%d|song_list|0", s.ID)),
			})
		}
	}

	rm.Inline(rows...)
	return c.Send(text.String(), rm, tele.ModeHTML)
}

func (b *Bot) handleAllSetlists(c tele.Context) error {
	if isPrivateChat(c) {
		return b.handleAllSetlistsPrivate(c)
	}

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

	origin := fmt.Sprintf("setlist_list|%d", c.Chat().ID)
	rm := setlistListKeyboard(setlists, origin)
	return c.Send("📋 Все сетлисты:", rm)
}

func (b *Bot) handleAllSetlistsPrivate(c tele.Context) error {
	chatIDs := b.getUserChatIDs(c.Sender().ID)
	if len(chatIDs) == 0 {
		return c.Send("Вы не состоите ни в одной группе с этим ботом.")
	}

	ctx := context.Background()
	setlists, err := b.store.ListSetlistsInChats(ctx, chatIDs)
	if err != nil {
		return c.Send("Ошибка: " + err.Error())
	}
	if len(setlists) == 0 {
		return c.Send("Сетлистов пока нет.")
	}

	chatNames := b.buildChatNameMap(ctx, chatIDs)

	var text strings.Builder
	text.WriteString("📋 Все сетлисты:\n")

	rm := &tele.ReplyMarkup{}
	var rows []tele.Row

	grouped := groupSetlistsByChat(setlists)
	for chatID, chatSetlists := range grouped {
		name := chatNames[chatID]
		if name == "" {
			name = fmt.Sprintf("Чат %d", chatID)
		}
		text.WriteString(fmt.Sprintf("\n<b>%s</b>\n", escapeHTML(name)))
		for _, sl := range chatSetlists {
			text.WriteString(sl.Name + "\n")
			rows = append(rows, tele.Row{
				rm.Data(sl.Name, "show_sl", fmt.Sprintf("%d|setlist_list|0", sl.ID)),
			})
		}
	}

	rm.Inline(rows...)
	return c.Send(text.String(), rm, tele.ModeHTML)
}

func (b *Bot) buildChatNameMap(ctx context.Context, chatIDs []int64) map[int64]string {
	m := make(map[int64]string, len(chatIDs))
	for _, id := range chatIDs {
		if n, _ := b.store.GetChatName(ctx, id); n != nil {
			m[id] = *n
		}
	}
	return m
}

func groupSongsByChat(songs []model.Song) map[int64][]model.Song {
	m := make(map[int64][]model.Song)
	for _, s := range songs {
		m[s.ChatID] = append(m[s.ChatID], s)
	}
	return m
}

func groupSetlistsByChat(setlists []model.Setlist) map[int64][]model.Setlist {
	m := make(map[int64][]model.Setlist)
	for _, sl := range setlists {
		m[sl.ChatID] = append(m[sl.ChatID], sl)
	}
	return m
}
