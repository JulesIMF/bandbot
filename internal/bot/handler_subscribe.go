package bot

import (
	"context"

	tele "gopkg.in/telebot.v3"
)

func (b *Bot) handleToggleSubscribeAll(c tele.Context) error {
	if isPrivateChat(c) {
		return c.Send("Эта команда работает только в групповом чате.")
	}
	if err := b.ensureChat(c); err != nil {
		return c.Send("Ошибка: " + err.Error())
	}

	username := senderUsername(c)
	if username == "" {
		return c.Send("Для подписки необходим никнейм в Telegram.")
	}

	ctx := context.Background()
	newVal, err := b.store.ToggleSubscribeAll(ctx, c.Sender().ID, c.Chat().ID, username)
	if err != nil {
		return c.Send("Ошибка: " + err.Error())
	}

	if newVal {
		return c.Send("✅ Вы подписались на все изменения в этом чате.")
	}
	return c.Send("❌ Вы отписались от всех изменений в этом чате.")
}
