package bot

import tele "gopkg.in/telebot.v3"

func (b *Bot) handleStart(c tele.Context) error {
	return c.Send(
		"🎵 <b>BandBot</b> — бот для управления репертуаром группы.\n\n" +
			"Добавьте меня в чат группы и используйте /help для списка команд.")
}

func (b *Bot) handleHelp(c tele.Context) error {
	return c.Send(
		"<b>Команды:</b>\n\n"+
			"/song Название — показать или создать песню\n"+
			"/song Название: 120 Cm — создать/обновить с темпом и тональностью\n"+
			"/setlist Название — показать или создать сетлист\n"+
			"/setlist — показать активный сетлист\n"+
			"/all_songs — все песни\n"+
			"/all_setlists — все сетлисты\n"+
			"/toggle_subscribe_all — подписаться/отписаться от всех изменений\n\n"+
			"<b>Хештеги:</b>\n\n"+
			"<code>#примечание Название песни\nТекст примечания</code>\n\n"+
			"<code>#закреп Название песни\nНазвание закрепа</code>\n"+
			"(ответом на закрепляемое сообщение)\n\n"+
			"<b>Inline-поиск:</b>\n"+
			"Введите <code>@имя_бота запрос</code> для поиска песен.",
		tele.ModeHTML)
}
