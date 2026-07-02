package bot

import (
	"context"
	"strings"

	"github.com/julesimf/bandbot/internal/model"
	"github.com/julesimf/bandbot/internal/normalize"
	tele "gopkg.in/telebot.v3"
)

func (b *Bot) handlePin(c tele.Context) error {
	if err := b.ensureChat(c); err != nil {
		return c.Send("Ошибка: " + err.Error())
	}

	if c.Message().ReplyTo == nil {
		return c.Send("Чтобы закрепить сообщение, ответьте на него (reply).")
	}

	m := pinPattern.FindStringSubmatch(c.Text())
	if m == nil || len(m) < 3 {
		return c.Send("Формат: #закреп Название песни\nНазвание закрепа\n\n(ответом на закрепляемое сообщение)")
	}

	songName := normalize.SongName(strings.TrimSpace(m[1]))
	label := strings.TrimSpace(m[2])

	if songName == "" {
		return c.Send("Укажите название песни.")
	}
	if label == "" {
		return c.Send("Укажите осмысленное название для закрепа.")
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
	pinnedMsgID := c.Message().ReplyTo.ID
	err = b.store.AddPin(ctx, &model.SongPin{
		SongID:    song.ID,
		Label:     label,
		MessageID: int64(pinnedMsgID),
		ChatID:    c.Chat().ID,
		PinnedBy:  user,
	})
	if err != nil {
		return c.Send("Ошибка добавления закрепа: " + err.Error())
	}

	_ = b.store.AddHistory(ctx, &model.SongHistory{
		SongID: song.ID, Field: "pin",
		NewValue: label, ChangedBy: user,
	})
	_ = b.store.TouchSong(ctx, song.ID)

	song, _ = b.store.GetSongByID(ctx, song.ID)
	return b.sendSongCard(c, song, &ChangeHeader{})
}
