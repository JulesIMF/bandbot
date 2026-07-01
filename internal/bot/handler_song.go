package bot

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/julesimf/bandbot/internal/model"
	"github.com/julesimf/bandbot/internal/music"
	"github.com/julesimf/bandbot/internal/normalize"
	tele "gopkg.in/telebot.v3"
)

var tempoRe = regexp.MustCompile(`(?i)(\d{2,3})\s*(?:bpm)?`)
var keyRe = regexp.MustCompile(`(?i)([A-Ga-g][#♭b]?m?)`)

func (b *Bot) handleSong(c tele.Context) error {
	if isPrivateChat(c) {
		return b.handleSongPrivate(c)
	}

	if err := b.ensureChat(c); err != nil {
		return c.Send("Ошибка: " + err.Error())
	}

	payload := strings.TrimSpace(extractCommandPayload(c.Text()))
	if payload == "" {
		return b.showRecentSongs(c)
	}

	var name, optsStr string
	if idx := strings.Index(payload, ":"); idx != -1 {
		name = strings.TrimSpace(payload[:idx])
		optsStr = strings.TrimSpace(payload[idx+1:])
	} else {
		name = payload
	}

	name = normalize.SongName(name)
	if name == "" {
		return c.Send("Укажите название песни.")
	}

	ctx := context.Background()
	song, err := b.store.GetSong(ctx, c.Chat().ID, name)
	if err != nil {
		return c.Send("Ошибка: " + err.Error())
	}

	var parsedTempo *int
	var parsedKey *music.Key

	if optsStr != "" {
		remaining := optsStr

		if m := tempoRe.FindStringSubmatch(remaining); m != nil {
			t, _ := strconv.Atoi(m[1])
			parsedTempo = &t
			remaining = strings.Replace(remaining, m[0], "", 1)
		}

		remaining = strings.TrimSpace(remaining)
		if remaining != "" {
			if m := keyRe.FindStringSubmatch(remaining); m != nil {
				k, err := music.Parse(m[1])
				if err != nil {
					return c.Send("Ошибка тональности: " + err.Error())
				}
				parsedKey = &k
			}
		}
	}

	isNew := song == nil
	if isNew {
		username := senderUsername(c)
		if username == "" {
			return c.Send("Для создания песни у вас должен быть никнейм в Telegram.")
		}
		song = &model.Song{
			ChatID:      c.Chat().ID,
			Name:        name,
			Responsible: username,
		}
		if parsedTempo != nil {
			song.Tempo = parsedTempo
		}
		if parsedKey != nil {
			ks := parsedKey.String()
			song.Key = &ks
		}
		if err := b.store.CreateSong(ctx, song); err != nil {
			return c.Send("Ошибка создания: " + err.Error())
		}

		changes := buildInitialChanges(song)
		return b.sendSongCard(c, song, changes)
	}

	// Existing song
	_ = b.store.TouchSong(ctx, song.ID)
	changes := &ChangeHeader{}
	hasChanges := false
	user := senderDisplayName(c)

	if parsedTempo != nil && (song.Tempo == nil || *song.Tempo != *parsedTempo) {
		changes.TempoOld = song.Tempo
		changes.TempoNew = parsedTempo

		oldVal := ""
		if song.Tempo != nil {
			oldVal = fmt.Sprintf("%d bpm", *song.Tempo)
		}
		_ = b.store.AddHistory(ctx, &model.SongHistory{
			SongID: song.ID, Field: "tempo",
			OldValue: strPtr(oldVal), NewValue: fmt.Sprintf("%d bpm", *parsedTempo),
			ChangedBy: user,
		})

		song.Tempo = parsedTempo
		hasChanges = true
	}

	if parsedKey != nil {
		ks := parsedKey.String()
		if song.Key == nil || *song.Key != ks {
			changes.KeyOld = song.Key
			changes.KeyNew = &ks

			oldVal := ""
			if song.Key != nil {
				oldVal = *song.Key
			}
			_ = b.store.AddHistory(ctx, &model.SongHistory{
				SongID: song.ID, Field: "key",
				OldValue: strPtr(oldVal), NewValue: ks,
				ChangedBy: user,
			})

			song.Key = &ks
			hasChanges = true
		}
	}

	if hasChanges {
		if err := b.store.UpdateSong(ctx, song); err != nil {
			return c.Send("Ошибка обновления: " + err.Error())
		}
		song, _ = b.store.GetSongByID(ctx, song.ID)
		return b.sendSongCard(c, song, changes)
	}

	return b.sendSongCard(c, song, nil)
}

func (b *Bot) handleSongPrivate(c tele.Context) error {
	payload := strings.TrimSpace(extractCommandPayload(c.Text()))
	if payload == "" {
		return b.showRecentSongsPrivate(c)
	}

	var name, optsStr string
	if idx := strings.Index(payload, ":"); idx != -1 {
		name = strings.TrimSpace(payload[:idx])
		optsStr = strings.TrimSpace(payload[idx+1:])
	} else {
		name = payload
	}

	if strings.TrimSpace(optsStr) != "" {
		return c.Send("Изменение песен доступно только в групповом чате.")
	}

	name = normalize.SongName(name)
	if name == "" {
		return c.Send("Укажите название песни.")
	}

	chatIDs := b.getUserChatIDs(c.Sender().ID)
	if len(chatIDs) == 0 {
		return c.Send("Вы не состоите ни в одной группе с этим ботом.")
	}

	ctx := context.Background()
	songs, err := b.store.GetSongByNameInChats(ctx, chatIDs, name)
	if err != nil {
		return c.Send("Ошибка: " + err.Error())
	}

	if len(songs) == 0 {
		return c.Send(fmt.Sprintf("Песня «%s» не найдена.", name))
	}

	if len(songs) == 1 {
		song, err := b.store.GetSongByID(ctx, songs[0].ID)
		if err != nil || song == nil {
			return c.Send("Ошибка загрузки песни.")
		}
		return b.sendSongCardReadonly(c, song)
	}

	rm := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, s := range songs {
		chatName := ""
		if n, _ := b.store.GetChatName(ctx, s.ChatID); n != nil {
			chatName = *n
		}
		label := s.Name
		if chatName != "" {
			label = fmt.Sprintf("%s (%s)", s.Name, chatName)
		}
		rows = append(rows, tele.Row{
			rm.Data(label, "show_song", fmt.Sprintf("%d", s.ID)),
		})
	}
	rm.Inline(rows...)
	return c.Send(fmt.Sprintf("Песня «%s» найдена в нескольких группах. Выберите:", name), rm)
}

func (b *Bot) showRecentSongsPrivate(c tele.Context) error {
	chatIDs := b.getUserChatIDs(c.Sender().ID)
	if len(chatIDs) == 0 {
		return c.Send("Вы не состоите ни в одной группе с этим ботом.")
	}

	ctx := context.Background()
	songs, err := b.store.SearchSongsInChats(ctx, chatIDs, "", 20)
	if err != nil {
		return c.Send("Ошибка: " + err.Error())
	}
	if len(songs) == 0 {
		return c.Send("Песен пока нет.")
	}

	text := "🎵 Недавние песни:\n\n" + RenderSongList(songs)

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

func (b *Bot) sendSongCardReadonly(c tele.Context, song *model.Song) error {
	ctx := context.Background()
	chatName := ""
	if n, _ := b.store.GetChatName(ctx, song.ChatID); n != nil {
		chatName = *n
	}
	text := RenderSongCardWithChat(song, nil, nil, chatName)
	isSubbed, _ := b.store.IsSubscribed(ctx, song.ID, c.Sender().ID)
	kb := SongCardKeyboardReadonly(song, isSubbed)
	return c.Send(text, kb, tele.ModeHTML)
}

func (b *Bot) showRecentSongs(c tele.Context) error {
	ctx := context.Background()
	songs, err := b.store.SearchSongs(ctx, c.Chat().ID, "", 20)
	if err != nil {
		return c.Send("Ошибка: " + err.Error())
	}
	if len(songs) == 0 {
		return c.Send("Песен пока нет. Создайте первую: /song Название")
	}

	text := "🎵 Недавние песни:\n\n" + RenderSongList(songs)

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

func (b *Bot) sendSongCard(c tele.Context, song *model.Song, changes *ChangeHeader) error {
	ctx := context.Background()
	var ccList []string

	if changes != nil {
		ccList, _ = b.store.GetNotifyList(ctx, song)
	}

	text := RenderSongCard(song, changes, ccList)
	isSubbed, _ := b.store.IsSubscribed(ctx, song.ID, c.Sender().ID)
	kb := SongCardKeyboard(song, isSubbed)

	return c.Send(text, kb, tele.ModeHTML)
}

func (b *Bot) handleShowSongCallback(c tele.Context) error {
	songID, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Некорректный ID"})
	}

	ctx := context.Background()
	song, err := b.store.GetSongByID(ctx, songID)
	if err != nil || song == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Песня не найдена"})
	}

	_ = b.store.TouchSong(ctx, song.ID)
	_ = c.Respond()

	if isPrivateChat(c) {
		return b.sendSongCardReadonly(c, song)
	}

	if err := b.ensureChat(c); err != nil {
		return c.Send("Ошибка: " + err.Error())
	}

	text := RenderSongCard(song, nil, nil)
	isSubbed, _ := b.store.IsSubscribed(ctx, song.ID, c.Sender().ID)
	kb := SongCardKeyboard(song, isSubbed)

	return c.Send(text, kb, tele.ModeHTML)
}

func buildInitialChanges(song *model.Song) *ChangeHeader {
	ch := &ChangeHeader{}
	hasAny := false
	if song.Tempo != nil {
		ch.TempoNew = song.Tempo
		hasAny = true
	}
	if song.Key != nil {
		ch.KeyNew = song.Key
		hasAny = true
	}
	if !hasAny {
		return nil
	}
	return ch
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
