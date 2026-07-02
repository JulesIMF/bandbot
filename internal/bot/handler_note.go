package bot

import (
	"context"
	"strings"

	"github.com/julesimf/bandbot/internal/model"
	"github.com/julesimf/bandbot/internal/normalize"
	tele "gopkg.in/telebot.v3"
)

func (b *Bot) handleNote(c tele.Context) error {
	if err := b.ensureChat(c); err != nil {
		return c.Send("Ошибка: " + err.Error())
	}

	m := notePattern.FindStringSubmatch(c.Text())
	if m == nil || len(m) < 3 {
		return c.Send("Формат: #примечание Название песни\nТекст примечания")
	}

	songName := normalize.SongName(strings.TrimSpace(m[1]))
	noteText := strings.TrimSpace(m[2])

	if songName == "" {
		return c.Send("Укажите название песни.")
	}
	if noteText == "" {
		return c.Send("Укажите текст примечания.")
	}

	ctx := context.Background()
	song, err := b.store.GetSong(ctx, c.Chat().ID, songName)
	if err != nil {
		return c.Send("Ошибка: " + err.Error())
	}
	if song == nil {
		return c.Send("Песня «" + songName + "» не найдена.")
	}

	if !b.auth.CanEditSong(ctx, c.Sender().ID, song) {
		return c.Send("Недостаточно прав.")
	}

	user := senderDisplayName(c)
	err = b.store.AddNote(ctx, &model.SongNote{
		SongID:    song.ID,
		Content:   noteText,
		CreatedBy: user,
	})
	if err != nil {
		return c.Send("Ошибка добавления примечания: " + err.Error())
	}

	_ = b.store.AddHistory(ctx, &model.SongHistory{
		SongID: song.ID, Field: "note",
		NewValue: noteText, ChangedBy: user,
	})
	_ = b.store.TouchSong(ctx, song.ID)

	song, _ = b.store.GetSongByID(ctx, song.ID)
	return b.sendSongCard(c, song, &ChangeHeader{})
}
