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

func (b *Bot) handleKeySelect(c tele.Context) error {
	if isPrivateChat(c) {
		return c.Respond(&tele.CallbackResponse{Text: privateChatEditError})
	}
	songID, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	rm := &tele.ReplyMarkup{}
	rows := music.KeyboardRows()
	var teleRows []tele.Row
	for _, row := range rows {
		teleRows = append(teleRows, tele.Row{
			rm.Data(row[0], "set_key", fmt.Sprintf("%d|%s", songID, row[0])),
			rm.Data(row[1], "set_key", fmt.Sprintf("%d|%s", songID, row[1])),
		})
	}
	teleRows = append(teleRows, tele.Row{
		rm.Data("← Назад", "nav_back", fmt.Sprintf("song|%d", songID)),
	})
	rm.Inline(teleRows...)

	_ = c.Respond()
	return c.Edit(c.Message().Text, rm, tele.ModeHTML)
}

func (b *Bot) handleSetKey(c tele.Context) error {
	if isPrivateChat(c) {
		return c.Respond(&tele.CallbackResponse{Text: privateChatEditError})
	}
	parts := strings.SplitN(c.Callback().Data, "|", 2)
	if len(parts) != 2 {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	songID, err := strconv.Atoi(parts[0])
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}
	newKey := parts[1]

	ctx := context.Background()
	song, err := b.store.GetSongByID(ctx, songID)
	if err != nil || song == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Песня не найдена"})
	}

	if !b.auth.CanEditSong(ctx, c.Sender().ID, song) {
		return c.Respond(&tele.CallbackResponse{Text: "Недостаточно прав"})
	}

	changes := &ChangeHeader{KeyOld: song.Key, KeyNew: &newKey}
	user := senderDisplayName(c)

	oldVal := ""
	if song.Key != nil {
		oldVal = *song.Key
	}
	_ = b.store.AddHistory(ctx, &model.SongHistory{
		SongID: song.ID, Field: "key",
		OldValue: strPtr(oldVal), NewValue: newKey,
		ChangedBy: user,
	})

	song.Key = &newKey
	_ = b.store.UpdateSong(ctx, song)

	song, _ = b.store.GetSongByID(ctx, song.ID)
	ccList, _ := b.store.GetNotifyList(ctx, song)
	text := RenderSongCard(song, changes, ccList)
	isSubbed, _ := b.store.IsSubscribed(ctx, song.ID, c.Sender().ID)
	kb := SongCardKeyboard(song, isSubbed)

	_ = c.Respond()
	return c.Edit(text, kb, tele.ModeHTML)
}

func (b *Bot) handleKeyBack(c tele.Context) error {
	songID, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	ctx := context.Background()
	song, err := b.store.GetSongByID(ctx, songID)
	if err != nil || song == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Песня не найдена"})
	}

	text := RenderSongCard(song, nil, nil)
	isSubbed, _ := b.store.IsSubscribed(ctx, song.ID, c.Sender().ID)
	kb := SongCardKeyboard(song, isSubbed)

	_ = c.Respond()
	return c.Edit(text, kb, tele.ModeHTML)
}

func (b *Bot) handleTempoPrompt(c tele.Context) error {
	if isPrivateChat(c) {
		return c.Respond(&tele.CallbackResponse{Text: privateChatEditError})
	}
	songID, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	ctx := context.Background()
	song, _ := b.store.GetSongByID(ctx, songID)
	if song == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Песня не найдена"})
	}

	current := "не задан"
	if song.Tempo != nil {
		current = fmt.Sprintf("%d bpm", *song.Tempo)
	}

	_ = c.Respond()
	msg, err := b.tele.Send(c.Chat(),
		fmt.Sprintf("Смена темпа · %s\nТекущий: %s\n\nОтветьте на это сообщение, указав новый темп (например: 120 или 120 bpm)", song.Name, current))
	if err != nil {
		return err
	}
	b.prompts.Set(c.Chat().ID, msg.ID, PromptContext{Type: PromptTempo, TargetID: songID})
	return nil
}

func (b *Bot) handleRenameSongPrompt(c tele.Context) error {
	if isPrivateChat(c) {
		return c.Respond(&tele.CallbackResponse{Text: privateChatEditError})
	}
	songID, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	ctx := context.Background()
	song, _ := b.store.GetSongByID(ctx, songID)
	if song == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Песня не найдена"})
	}

	_ = c.Respond()
	msg, err := b.tele.Send(c.Chat(),
		fmt.Sprintf("Переименование песни\nТекущее название: %s\n\nОтветьте на это сообщение новым названием.", song.Name))
	if err != nil {
		return err
	}
	b.prompts.Set(c.Chat().ID, msg.ID, PromptContext{Type: PromptRenameSong, TargetID: songID})
	return nil
}

func (b *Bot) handleResponsiblePrompt(c tele.Context) error {
	if isPrivateChat(c) {
		return c.Respond(&tele.CallbackResponse{Text: privateChatEditError})
	}
	songID, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	ctx := context.Background()
	song, _ := b.store.GetSongByID(ctx, songID)
	if song == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Песня не найдена"})
	}

	current := "не назначен"
	if song.Responsible != "" {
		current = song.Responsible
	}

	_ = c.Respond()
	msg, err := b.tele.Send(c.Chat(),
		fmt.Sprintf("Смена ответственного · %s\nТекущий: %s\n\nОтветьте на это сообщение в формате @username", song.Name, current))
	if err != nil {
		return err
	}
	b.prompts.Set(c.Chat().ID, msg.ID, PromptContext{Type: PromptResponsible, TargetID: songID})
	return nil
}

var tempoReplyRe = regexp.MustCompile(`(?i)^\s*(\d{2,3})\s*(?:bpm)?\s*$`)
var responsibleReplyRe = regexp.MustCompile(`^\s*@(\w+)\s*$`)

func (b *Bot) handlePromptReply(c tele.Context) error {
	replyTo := c.Message().ReplyTo
	if replyTo == nil {
		return nil
	}

	pctx, ok := b.prompts.Get(c.Chat().ID, replyTo.ID)
	if !ok {
		return nil
	}

	b.prompts.Delete(c.Chat().ID, replyTo.ID)

	switch pctx.Type {
	case PromptTempo:
		return b.processTempoReply(c, pctx.TargetID)
	case PromptRenameSong:
		return b.processRenameSongReply(c, pctx.TargetID)
	case PromptResponsible:
		return b.processResponsibleReply(c, pctx.TargetID)
	case PromptRenameSetlist:
		return b.processRenameSetlistReply(c, pctx.TargetID)
	case PromptSetlistSongs:
		return b.processSetlistSongsReply(c, pctx.TargetID)
	case PromptCreateSetlist:
		return b.processCreateSetlistReply(c, pctx.TargetName)
	}

	return nil
}

func (b *Bot) processTempoReply(c tele.Context, songID int) error {
	m := tempoReplyRe.FindStringSubmatch(c.Text())
	if m == nil {
		return c.Send("Неверный формат темпа. Укажите число, например: 120 или 120 bpm")
	}

	tempo, _ := strconv.Atoi(m[1])
	ctx := context.Background()
	song, err := b.store.GetSongByID(ctx, songID)
	if err != nil || song == nil {
		return c.Send("Песня не найдена.")
	}

	if !b.auth.CanEditSong(ctx, c.Sender().ID, song) {
		return c.Send("Недостаточно прав.")
	}

	changes := &ChangeHeader{TempoOld: song.Tempo, TempoNew: &tempo}
	user := senderDisplayName(c)

	oldVal := ""
	if song.Tempo != nil {
		oldVal = fmt.Sprintf("%d bpm", *song.Tempo)
	}
	_ = b.store.AddHistory(ctx, &model.SongHistory{
		SongID: song.ID, Field: "tempo",
		OldValue: strPtr(oldVal), NewValue: fmt.Sprintf("%d bpm", tempo),
		ChangedBy: user,
	})

	song.Tempo = &tempo
	_ = b.store.UpdateSong(ctx, song)

	song, _ = b.store.GetSongByID(ctx, song.ID)
	return b.sendSongCard(c, song, changes)
}

func (b *Bot) processRenameSongReply(c tele.Context, songID int) error {
	newName := normalize.SongName(strings.TrimSpace(c.Text()))
	if newName == "" {
		return c.Send("Название не может быть пустым.")
	}

	ctx := context.Background()
	song, err := b.store.GetSongByID(ctx, songID)
	if err != nil || song == nil {
		return c.Send("Песня не найдена.")
	}

	if !b.auth.CanEditSong(ctx, c.Sender().ID, song) {
		return c.Send("Недостаточно прав.")
	}

	existing, _ := b.store.GetSong(ctx, song.ChatID, newName)
	if existing != nil {
		return c.Send(fmt.Sprintf("Песня «%s» уже существует.", newName))
	}

	oldName := song.Name
	changes := &ChangeHeader{NameOld: oldName, NameNew: newName}
	user := senderDisplayName(c)

	_ = b.store.AddHistory(ctx, &model.SongHistory{
		SongID: song.ID, Field: "name",
		OldValue: &oldName, NewValue: newName,
		ChangedBy: user,
	})

	song.Name = newName
	_ = b.store.UpdateSong(ctx, song)

	song, _ = b.store.GetSongByID(ctx, song.ID)
	return b.sendSongCard(c, song, changes)
}

func (b *Bot) processResponsibleReply(c tele.Context, songID int) error {
	m := responsibleReplyRe.FindStringSubmatch(c.Text())
	if m == nil {
		return c.Send("Неверный формат. Укажите ответственного в формате @username")
	}

	newResp := m[1]
	ctx := context.Background()
	song, err := b.store.GetSongByID(ctx, songID)
	if err != nil || song == nil {
		return c.Send("Песня не найдена.")
	}

	if !b.auth.CanEditSong(ctx, c.Sender().ID, song) {
		return c.Send("Недостаточно прав.")
	}

	oldResp := song.Responsible
	user := senderDisplayName(c)
	_ = b.store.AddHistory(ctx, &model.SongHistory{
		SongID: song.ID, Field: "responsible",
		OldValue: strPtr(oldResp), NewValue: newResp,
		ChangedBy: user,
	})

	// Collect notify list BEFORE changing responsible to include old one
	ccList, _ := b.store.GetNotifyList(ctx, song)
	if oldResp != "" {
		found := false
		for _, u := range ccList {
			if u == oldResp {
				found = true
				break
			}
		}
		if !found {
			ccList = append(ccList, oldResp)
		}
	}

	song.Responsible = newResp
	_ = b.store.UpdateSong(ctx, song)
	song, _ = b.store.GetSongByID(ctx, song.ID)

	changes := &ChangeHeader{ResponsibleOld: oldResp, ResponsibleNew: newResp}
	text := RenderSongCard(song, changes, ccList)
	isSubbed, _ := b.store.IsSubscribed(ctx, song.ID, c.Sender().ID)
	kb := SongCardKeyboard(song, isSubbed)

	return c.Send(text, kb, tele.ModeHTML)
}

func (b *Bot) handleSubscribe(c tele.Context) error {
	songID, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	username := senderUsername(c)
	if username == "" {
		return c.Respond(&tele.CallbackResponse{Text: "Для подписки нужен никнейм"})
	}

	ctx := context.Background()
	_ = b.store.Subscribe(ctx, &model.SongSubscriber{
		SongID: songID, UserID: c.Sender().ID, Username: username,
	})

	song, _ := b.store.GetSongByID(ctx, songID)
	if song == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Песня не найдена"})
	}

	text := RenderSongCard(song, nil, nil)
	kb := SongCardKeyboard(song, true)
	_ = c.Respond(&tele.CallbackResponse{Text: "Вы подписались на изменения"})
	return c.Edit(text, kb, tele.ModeHTML)
}

func (b *Bot) handleUnsubscribe(c tele.Context) error {
	songID, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	ctx := context.Background()
	_ = b.store.Unsubscribe(ctx, songID, c.Sender().ID)

	song, _ := b.store.GetSongByID(ctx, songID)
	if song == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Песня не найдена"})
	}

	text := RenderSongCard(song, nil, nil)
	kb := SongCardKeyboard(song, false)
	_ = c.Respond(&tele.CallbackResponse{Text: "Вы отписались от изменений"})
	return c.Edit(text, kb, tele.ModeHTML)
}

func (b *Bot) handleHistory(c tele.Context) error {
	songID, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	ctx := context.Background()
	song, _ := b.store.GetSongByID(ctx, songID)
	if song == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Песня не найдена"})
	}

	history, _ := b.store.GetHistory(ctx, songID)
	text := RenderHistory(song, history)

	rm := &tele.ReplyMarkup{}
	rm.Inline(tele.Row{
		rm.Data("← Назад", "nav_back", fmt.Sprintf("song|%d", songID)),
	})

	_ = c.Respond()
	return c.Edit(text, rm, tele.ModeHTML)
}

func (b *Bot) handleDeleteSong(c tele.Context) error {
	if isPrivateChat(c) {
		return c.Respond(&tele.CallbackResponse{Text: privateChatEditError})
	}
	songID, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	ctx := context.Background()
	song, _ := b.store.GetSongByID(ctx, songID)
	if song == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Песня не найдена"})
	}

	if !b.auth.CanDeleteSong(ctx, c.Sender().ID, song) {
		return c.Respond(&tele.CallbackResponse{Text: "Недостаточно прав"})
	}

	remaining := pressDelete(c.Chat().ID, c.Message().ID)
	if remaining > 0 {
		text := RenderSongCard(song, nil, nil)
		isSubbed, _ := b.store.IsSubscribed(ctx, song.ID, c.Sender().ID)
		kb := SongCardKeyboard(song, isSubbed, KeyboardOpts{DeleteRemaining: &remaining})
		_ = c.Respond()
		return c.Edit(text, kb, tele.ModeHTML)
	}

	deleter := senderDisplayName(c)
	text := RenderSongDeleted(song, deleter)
	ccList, _ := b.store.GetNotifyList(ctx, song)
	if len(ccList) > 0 {
		text += "\n\ncc"
		for _, u := range ccList {
			text += " @" + u
		}
	}

	_ = b.store.DeleteSong(ctx, songID)
	_ = c.Respond()
	return c.Edit(text, tele.ModeHTML)
}

func (b *Bot) handleDeleteNoteSelect(c tele.Context) error {
	if isPrivateChat(c) {
		return c.Respond(&tele.CallbackResponse{Text: privateChatEditError})
	}
	songID, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	ctx := context.Background()
	song, _ := b.store.GetSongByID(ctx, songID)
	if song == nil || len(song.Notes) == 0 {
		return c.Respond(&tele.CallbackResponse{Text: "Примечаний нет"})
	}

	rm := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, n := range song.Notes {
		label := n.Content
		if len([]rune(label)) > 40 {
			label = string([]rune(label)[:40]) + "…"
		}
		rows = append(rows, tele.Row{
			rm.Data(label, "rm_note", fmt.Sprintf("%d|%d", songID, n.ID)),
		})
	}
	rows = append(rows, tele.Row{
		rm.Data("← Назад", "nav_back", fmt.Sprintf("song|%d", songID)),
	})
	rm.Inline(rows...)

	_ = c.Respond()
	return c.Edit(c.Message().Text, rm, tele.ModeHTML)
}

func (b *Bot) handleRemoveNote(c tele.Context) error {
	if isPrivateChat(c) {
		return c.Respond(&tele.CallbackResponse{Text: privateChatEditError})
	}
	parts := strings.SplitN(c.Callback().Data, "|", 2)
	if len(parts) != 2 {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	songID, _ := strconv.Atoi(parts[0])
	noteID, _ := strconv.Atoi(parts[1])

	ctx := context.Background()
	song, _ := b.store.GetSongByID(ctx, songID)
	if song == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Песня не найдена"})
	}

	ccList, _ := b.store.GetNotifyList(ctx, song)
	text := RenderSongCard(song, nil, ccList)

	_ = b.store.DeleteNote(ctx, noteID)
	_ = b.store.AddHistory(ctx, &model.SongHistory{
		SongID: songID, Field: "note",
		OldValue: nil, NewValue: "удалено примечание",
		ChangedBy: senderDisplayName(c),
	})

	song, _ = b.store.GetSongByID(ctx, songID)
	isSubbed, _ := b.store.IsSubscribed(ctx, songID, c.Sender().ID)
	kb := SongCardKeyboard(song, isSubbed)

	_ = c.Respond()
	return c.Edit(text+"\n✏️ Примечание удалено", kb, tele.ModeHTML)
}

func (b *Bot) handleClearNotes(c tele.Context) error {
	if isPrivateChat(c) {
		return c.Respond(&tele.CallbackResponse{Text: privateChatEditError})
	}
	songID, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	ctx := context.Background()
	song, _ := b.store.GetSongByID(ctx, songID)
	if song == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Песня не найдена"})
	}

	ccList, _ := b.store.GetNotifyList(ctx, song)
	text := RenderSongCard(song, nil, ccList)

	_ = b.store.ClearNotes(ctx, songID)
	_ = b.store.AddHistory(ctx, &model.SongHistory{
		SongID: songID, Field: "note",
		OldValue: nil, NewValue: "очищены все примечания",
		ChangedBy: senderDisplayName(c),
	})

	song, _ = b.store.GetSongByID(ctx, songID)
	isSubbed, _ := b.store.IsSubscribed(ctx, songID, c.Sender().ID)
	kb := SongCardKeyboard(song, isSubbed)

	_ = c.Respond()
	return c.Edit(text+"\n✏️ Все примечания удалены", kb, tele.ModeHTML)
}

func (b *Bot) handleDeletePinSelect(c tele.Context) error {
	if isPrivateChat(c) {
		return c.Respond(&tele.CallbackResponse{Text: privateChatEditError})
	}
	songID, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	ctx := context.Background()
	song, _ := b.store.GetSongByID(ctx, songID)
	if song == nil || len(song.Pins) == 0 {
		return c.Respond(&tele.CallbackResponse{Text: "Закрепов нет"})
	}

	rm := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, p := range song.Pins {
		rows = append(rows, tele.Row{
			rm.Data("📎 "+p.Label, "rm_pin", fmt.Sprintf("%d|%d", songID, p.ID)),
		})
	}
	rows = append(rows, tele.Row{
		rm.Data("← Назад", "nav_back", fmt.Sprintf("song|%d", songID)),
	})
	rm.Inline(rows...)

	_ = c.Respond()
	return c.Edit(c.Message().Text, rm, tele.ModeHTML)
}

func (b *Bot) handleRemovePin(c tele.Context) error {
	if isPrivateChat(c) {
		return c.Respond(&tele.CallbackResponse{Text: privateChatEditError})
	}
	parts := strings.SplitN(c.Callback().Data, "|", 2)
	if len(parts) != 2 {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	songID, _ := strconv.Atoi(parts[0])
	pinID, _ := strconv.Atoi(parts[1])

	ctx := context.Background()
	song, _ := b.store.GetSongByID(ctx, songID)
	if song == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Песня не найдена"})
	}

	ccList, _ := b.store.GetNotifyList(ctx, song)
	text := RenderSongCard(song, nil, ccList)

	_ = b.store.DeletePin(ctx, pinID)
	_ = b.store.AddHistory(ctx, &model.SongHistory{
		SongID: songID, Field: "pin",
		OldValue: nil, NewValue: "удалён закреп",
		ChangedBy: senderDisplayName(c),
	})

	song, _ = b.store.GetSongByID(ctx, songID)
	isSubbed, _ := b.store.IsSubscribed(ctx, songID, c.Sender().ID)
	kb := SongCardKeyboard(song, isSubbed)

	_ = c.Respond()
	return c.Edit(text+"\n📎 Закреп удалён", kb, tele.ModeHTML)
}

func (b *Bot) handleClearPins(c tele.Context) error {
	if isPrivateChat(c) {
		return c.Respond(&tele.CallbackResponse{Text: privateChatEditError})
	}
	songID, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	ctx := context.Background()
	song, _ := b.store.GetSongByID(ctx, songID)
	if song == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Песня не найдена"})
	}

	ccList, _ := b.store.GetNotifyList(ctx, song)
	text := RenderSongCard(song, nil, ccList)

	_ = b.store.ClearPins(ctx, songID)
	_ = b.store.AddHistory(ctx, &model.SongHistory{
		SongID: songID, Field: "pin",
		OldValue: nil, NewValue: "очищены все закрепы",
		ChangedBy: senderDisplayName(c),
	})

	song, _ = b.store.GetSongByID(ctx, songID)
	isSubbed, _ := b.store.IsSubscribed(ctx, songID, c.Sender().ID)
	kb := SongCardKeyboard(song, isSubbed)

	_ = c.Respond()
	return c.Edit(text+"\n📎 Все закрепы удалены", kb, tele.ModeHTML)
}
