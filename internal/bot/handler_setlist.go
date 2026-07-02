package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/julesimf/bandbot/internal/fuzzy"
	"github.com/julesimf/bandbot/internal/model"
	"github.com/julesimf/bandbot/internal/normalize"
	tele "gopkg.in/telebot.v3"
)

func extractCommandPayload(text string) string {
	// Strip the /command (and optional @botname) prefix manually
	// to preserve newlines that Payload may lose
	i := 0
	for i < len(text) && text[i] != ' ' && text[i] != '\n' {
		i++
	}
	rest := text[i:]
	if len(rest) > 0 && rest[0] == ' ' {
		rest = rest[1:]
	}
	return rest
}

func (b *Bot) handleSetlist(c tele.Context) error {
	if isPrivateChat(c) {
		return b.handleSetlistPrivate(c)
	}

	if err := b.ensureChat(c); err != nil {
		return c.Send("Ошибка: " + err.Error())
	}

	text := extractCommandPayload(c.Text())
	if strings.TrimSpace(text) == "" {
		return b.showActiveSetlist(c)
	}

	lines := strings.Split(text, "\n")
	name := normalize.SongName(strings.TrimSpace(lines[0]))
	if name == "" {
		return c.Send("Укажите название сетлиста.")
	}

	ctx := context.Background()

	var songLines []string
	for _, l := range lines[1:] {
		l = strings.TrimSpace(l)
		if l != "" {
			songLines = append(songLines, l)
		}
	}

	if len(songLines) == 0 {
		sl, err := b.store.GetSetlist(ctx, c.Chat().ID, name)
		if err != nil {
			return c.Send("Ошибка: " + err.Error())
		}
		if sl != nil {
			return b.sendSetlistCard(c, sl)
		}
		msg, err := b.tele.Send(c.Chat(),
			fmt.Sprintf("Создаём сетлист «%s». Ответьте на это сообщение списком песен (по одной на строку).", name))
		if err != nil {
			return err
		}
		b.prompts.Set(c.Chat().ID, msg.ID, PromptContext{Type: PromptCreateSetlist, TargetName: name})
		return nil
	}

	return b.resolveAndCreateSetlist(c, name, songLines)
}

func (b *Bot) handleSetlistPrivate(c tele.Context) error {
	text := extractCommandPayload(c.Text())
	if strings.TrimSpace(text) == "" {
		return b.showActiveSetlistsPrivate(c)
	}

	lines := strings.Split(text, "\n")
	name := normalize.SongName(strings.TrimSpace(lines[0]))
	if name == "" {
		return c.Send("Укажите название сетлиста.")
	}

	for _, l := range lines[1:] {
		if strings.TrimSpace(l) != "" {
			return c.Send("Создание и изменение сетлистов доступно только в групповом чате.")
		}
	}

	chatIDs := b.getUserChatIDs(c.Sender().ID)
	if len(chatIDs) == 0 {
		return c.Send("Вы не состоите ни в одной группе с этим ботом.")
	}

	ctx := context.Background()
	setlists, err := b.store.GetSetlistByNameInChats(ctx, chatIDs, name)
	if err != nil {
		return c.Send("Ошибка: " + err.Error())
	}

	if len(setlists) == 0 {
		return c.Send(fmt.Sprintf("Сетлист «%s» не найден.", name))
	}

	if len(setlists) == 1 {
		return b.sendSetlistCardReadonly(c, &setlists[0])
	}

	rm := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, sl := range setlists {
		chatName := ""
		if n, _ := b.store.GetChatName(ctx, sl.ChatID); n != nil {
			chatName = *n
		}
		label := sl.Name
		if chatName != "" {
			label = fmt.Sprintf("%s (%s)", sl.Name, chatName)
		}
		rows = append(rows, tele.Row{
			rm.Data(label, "show_sl", fmt.Sprintf("%d", sl.ID)),
		})
	}
	rm.Inline(rows...)
	return c.Send(fmt.Sprintf("Сетлист «%s» найден в нескольких группах. Выберите:", name), rm)
}

func (b *Bot) showActiveSetlistsPrivate(c tele.Context) error {
	chatIDs := b.getUserChatIDs(c.Sender().ID)
	if len(chatIDs) == 0 {
		return c.Send("Вы не состоите ни в одной группе с этим ботом.")
	}

	ctx := context.Background()
	setlists, err := b.store.GetActiveSetlistsInChats(ctx, chatIDs)
	if err != nil {
		return c.Send("Ошибка: " + err.Error())
	}

	if len(setlists) == 0 {
		return c.Send("Ни в одной группе не установлен активный сетлист.")
	}

	for i := range setlists {
		if err := b.sendSetlistCardReadonly(c, &setlists[i]); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bot) showActiveSetlist(c tele.Context) error {
	ctx := context.Background()
	sl, err := b.store.GetActiveSetlist(ctx, c.Chat().ID)
	if err != nil {
		return c.Send("Ошибка: " + err.Error())
	}
	if sl == nil {
		return c.Send("Активный сетлист не установлен.")
	}
	return b.sendSetlistCard(c, sl)
}

func (b *Bot) sendSetlistCard(c tele.Context, sl *model.Setlist) error {
	return b.sendSetlistCardWithCC(c, sl, nil)
}

func (b *Bot) sendSetlistCardReadonly(c tele.Context, sl *model.Setlist) error {
	ctx := context.Background()
	chatName := ""
	if n, _ := b.store.GetChatName(ctx, sl.ChatID); n != nil {
		chatName = *n
	}
	text := RenderSetlistCard(sl)
	if chatName != "" {
		if isSupergroup(sl.ChatID) {
			link := chatDeepLink(sl.ChatID)
			text = fmt.Sprintf("📋 %s 🔗 <a href=\"%s\">%s</a>\n\n", sl.Name, link, escapeHTML(chatName)) + text[len(fmt.Sprintf("📋 %s\n\n", sl.Name)):]
		} else {
			text = fmt.Sprintf("📋 %s · %s\n\n", sl.Name, escapeHTML(chatName)) + text[len(fmt.Sprintf("📋 %s\n\n", sl.Name)):]
		}
	}
	kb := SetlistCardKeyboardReadonly(sl)
	return c.Send(text, kb, tele.ModeHTML)
}

func (b *Bot) sendSetlistCardWithCC(c tele.Context, sl *model.Setlist, ccList []string) error {
	text := RenderSetlistCard(sl)
	if cc := FormatCCLine(ccList); cc != "" {
		text = cc + "\n\n" + text
	}
	kb := SetlistCardKeyboard(sl)
	return c.Send(text, kb, tele.ModeHTML)
}

type songResolution struct {
	input      string
	normalized string
	songID     int
	candidates []fuzzy.Match
}

func (b *Bot) resolveAndCreateSetlist(c tele.Context, name string, songLines []string) error {
	ctx := context.Background()

	allNames, err := b.store.ListSongNames(ctx, c.Chat().ID)
	if err != nil {
		return c.Send("Ошибка: " + err.Error())
	}

	nameToSong := make(map[string]*model.Song)
	allSongs, _ := b.store.ListSongs(ctx, c.Chat().ID)
	for i := range allSongs {
		nameToSong[allSongs[i].Name] = &allSongs[i]
	}

	resolutions := make([]songResolution, len(songLines))
	var needDisambig []int
	var notFound []string

	for i, line := range songLines {
		norm := normalize.SongName(line)
		resolutions[i] = songResolution{input: line, normalized: norm}

		if s, ok := nameToSong[norm]; ok {
			resolutions[i].songID = s.ID
			continue
		}

		candidates := fuzzy.FindCandidates(norm, allNames, 0.4)
		if len(candidates) == 0 {
			notFound = append(notFound, line)
		} else if len(candidates) == 1 {
			if s, ok := nameToSong[candidates[0].Name]; ok {
				resolutions[i].songID = s.ID
			}
		} else {
			resolutions[i].candidates = candidates
			needDisambig = append(needDisambig, i)
		}
	}

	if len(notFound) > 0 {
		return c.Send("Не удалось найти песни:\n• " + strings.Join(notFound, "\n• "))
	}

	if len(needDisambig) > 0 {
		return b.startDisambiguation(c, name, resolutions, needDisambig, nameToSong)
	}

	return b.finalizeSetlist(c, name, resolutions)
}

type disambigState struct {
	setlistName string
	resolutions []songResolution
	pending     []int
	current     int
	nameToSong  map[string]*model.Song
}

var disambigStore = make(map[string]*disambigState)

func disambigKey(chatID int64, msgID int) string {
	return fmt.Sprintf("%d:%d", chatID, msgID)
}

func (b *Bot) startDisambiguation(c tele.Context, name string, resolutions []songResolution, pending []int, nameToSong map[string]*model.Song) error {
	idx := pending[0]
	res := resolutions[idx]

	rm := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, cand := range res.candidates {
		rows = append(rows, tele.Row{
			rm.Data(cand.Name, "sl_pick", fmt.Sprintf("%d|%s", idx, cand.Name)),
		})
	}
	rm.Inline(rows...)

	msg, err := b.tele.Send(c.Chat(),
		fmt.Sprintf("Для «%s» найдено несколько вариантов. Выберите:", res.input), rm)
	if err != nil {
		return err
	}

	disambigStore[disambigKey(c.Chat().ID, msg.ID)] = &disambigState{
		setlistName: name,
		resolutions: resolutions,
		pending:     pending,
		current:     0,
		nameToSong:  nameToSong,
	}

	return nil
}

func (b *Bot) handleSetlistPick(c tele.Context) error {
	data := c.Callback().Data
	parts := strings.SplitN(data, "|", 2)
	if len(parts) != 2 {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	idx, _ := strconv.Atoi(parts[0])
	chosenName := parts[1]

	key := disambigKey(c.Chat().ID, c.Message().ID)
	state, ok := disambigStore[key]
	if !ok {
		return c.Respond(&tele.CallbackResponse{Text: "Сессия выбора истекла"})
	}

	if s, ok := state.nameToSong[chosenName]; ok {
		state.resolutions[idx].songID = s.ID
	}

	state.current++
	if state.current < len(state.pending) {
		nextIdx := state.pending[state.current]
		res := state.resolutions[nextIdx]

		rm := &tele.ReplyMarkup{}
		var rows []tele.Row
		for _, cand := range res.candidates {
			rows = append(rows, tele.Row{
				rm.Data(cand.Name, "sl_pick", fmt.Sprintf("%d|%s", nextIdx, cand.Name)),
			})
		}
		rm.Inline(rows...)

		_ = c.Respond()
		return c.Edit(
			fmt.Sprintf("Для «%s» найдено несколько вариантов. Выберите:", res.input),
			rm, tele.ModeHTML)
	}

	delete(disambigStore, key)
	_ = c.Respond()

	return b.finalizeSetlistFromState(c, state)
}

func (b *Bot) finalizeSetlist(c tele.Context, name string, resolutions []songResolution) error {
	ctx := context.Background()

	var songIDs []int
	for _, r := range resolutions {
		songIDs = append(songIDs, r.songID)
	}

	existing, _ := b.store.GetSetlist(ctx, c.Chat().ID, name)
	if existing != nil {
		_ = b.store.UpdateSetlistSongs(ctx, existing.ID, songIDs)
		existing, _ = b.store.GetSetlistByID(ctx, existing.ID)
		return b.sendSetlistCard(c, existing)
	}

	sl := &model.Setlist{
		ChatID: c.Chat().ID,
		Name:   name,
	}
	if err := b.store.CreateSetlist(ctx, sl, songIDs); err != nil {
		return c.Send("Ошибка создания сетлиста: " + err.Error())
	}

	sl, _ = b.store.GetSetlistByID(ctx, sl.ID)
	return b.sendSetlistCard(c, sl)
}

func (b *Bot) finalizeSetlistFromState(c tele.Context, state *disambigState) error {
	ctx := context.Background()

	var songIDs []int
	for _, r := range state.resolutions {
		songIDs = append(songIDs, r.songID)
	}

	if strings.HasPrefix(state.setlistName, "__update__") {
		slID, _ := strconv.Atoi(strings.TrimPrefix(state.setlistName, "__update__"))
		_ = b.store.UpdateSetlistSongs(ctx, slID, songIDs)
		sl, _ := b.store.GetSetlistByID(ctx, slID)
		if sl == nil {
			return c.Send("Сетлист не найден.")
		}
		ccList, _ := b.store.GetSubscribeAllUsers(ctx, sl.ChatID)
		return b.sendSetlistCardWithCC(c, sl, ccList)
	}

	existing, _ := b.store.GetSetlist(ctx, c.Chat().ID, state.setlistName)
	if existing != nil {
		_ = b.store.UpdateSetlistSongs(ctx, existing.ID, songIDs)
		existing, _ = b.store.GetSetlistByID(ctx, existing.ID)
		ccList, _ := b.store.GetSubscribeAllUsers(ctx, existing.ChatID)
		return b.sendSetlistCardWithCC(c, existing, ccList)
	}

	sl := &model.Setlist{
		ChatID: c.Chat().ID,
		Name:   state.setlistName,
	}
	if err := b.store.CreateSetlist(ctx, sl, songIDs); err != nil {
		return c.Send("Ошибка создания сетлиста: " + err.Error())
	}

	sl, _ = b.store.GetSetlistByID(ctx, sl.ID)
	return b.sendSetlistCard(c, sl)
}

func (b *Bot) handleRenameSetlistPrompt(c tele.Context) error {
	if isPrivateChat(c) {
		return c.Respond(&tele.CallbackResponse{Text: privateChatEditError})
	}
	slID, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	ctx := context.Background()
	sl, _ := b.store.GetSetlistByID(ctx, slID)
	if sl == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Сетлист не найден"})
	}

	_ = c.Respond()
	msg, err := b.tele.Send(c.Chat(),
		fmt.Sprintf("Переименование сетлиста\nТекущее название: %s\n\nОтветьте на это сообщение новым названием.", sl.Name))
	if err != nil {
		return err
	}
	b.prompts.Set(c.Chat().ID, msg.ID, PromptContext{Type: PromptRenameSetlist, TargetID: slID})
	return nil
}

func (b *Bot) handleEditSetlistPrompt(c tele.Context) error {
	if isPrivateChat(c) {
		return c.Respond(&tele.CallbackResponse{Text: privateChatEditError})
	}
	slID, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	ctx := context.Background()
	sl, _ := b.store.GetSetlistByID(ctx, slID)
	if sl == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Сетлист не найден"})
	}

	var songList strings.Builder
	for _, e := range sl.Songs {
		if e.Song != nil {
			songList.WriteString(fmt.Sprintf("  %d. %s\n", e.Position, e.Song.Name))
		}
	}
	current := songList.String()
	if current == "" {
		current = "  (пусто)\n"
	}

	_ = c.Respond()
	msg, err := b.tele.Send(c.Chat(),
		fmt.Sprintf("Изменение списка песен · %s\nТекущий состав:\n%s\nОтветьте на это сообщение новым списком песен (по одной на строку).", sl.Name, current))
	if err != nil {
		return err
	}
	b.prompts.Set(c.Chat().ID, msg.ID, PromptContext{Type: PromptSetlistSongs, TargetID: slID})
	return nil
}

func (b *Bot) handleSetActiveSetlist(c tele.Context) error {
	if isPrivateChat(c) {
		return c.Respond(&tele.CallbackResponse{Text: privateChatEditError})
	}
	slID, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	ctx := context.Background()
	sl, _ := b.store.GetSetlistByID(ctx, slID)
	if sl == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Сетлист не найден"})
	}

	_ = b.store.SetActiveSetlist(ctx, sl.ChatID, slID)
	return c.Respond(&tele.CallbackResponse{Text: "Сетлист «" + sl.Name + "» назначен активным"})
}

func (b *Bot) handleDeleteSetlist(c tele.Context) error {
	if isPrivateChat(c) {
		return c.Respond(&tele.CallbackResponse{Text: privateChatEditError})
	}
	slID, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	ctx := context.Background()
	sl, _ := b.store.GetSetlistByID(ctx, slID)
	if sl == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Сетлист не найден"})
	}

	if !b.auth.CanDeleteSetlist(ctx, c.Sender().ID, sl) {
		return c.Respond(&tele.CallbackResponse{Text: "Недостаточно прав"})
	}

	remaining := pressDelete(c.Chat().ID, c.Message().ID)
	if remaining > 0 {
		text := RenderSetlistCard(sl)
		kb := SetlistCardKeyboard(sl, KeyboardOpts{DeleteRemaining: &remaining})
		_ = c.Respond()
		return c.Edit(text, kb, tele.ModeHTML)
	}

	deleter := senderDisplayName(c)
	text := RenderSetlistDeleted(sl, deleter)
	ccList, _ := b.store.GetSubscribeAllUsers(ctx, sl.ChatID)
	if cc := FormatCCLine(ccList); cc != "" {
		text += "\n\n" + cc
	}

	_ = b.store.DeleteSetlist(ctx, slID)

	_ = c.Respond()
	return c.Edit(text, tele.ModeHTML)
}

func parseShowSetlistData(data string) (int, string) {
	parts := strings.SplitN(data, "|", 2)
	slID, _ := strconv.Atoi(parts[0])
	origin := ""
	if len(parts) > 1 {
		origin = parts[1]
	}
	return slID, origin
}

func (b *Bot) handleShowSetlistCallback(c tele.Context) error {
	slID, origin := parseShowSetlistData(c.Callback().Data)
	if slID == 0 {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	ctx := context.Background()
	sl, _ := b.store.GetSetlistByID(ctx, slID)
	if sl == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Сетлист не найден"})
	}

	_ = c.Respond()
	if isPrivateChat(c) {
		chatName := ""
		if n, _ := b.store.GetChatName(ctx, sl.ChatID); n != nil {
			chatName = *n
		}
		text := RenderSetlistCard(sl)
		if chatName != "" {
			if isSupergroup(sl.ChatID) {
				link := chatDeepLink(sl.ChatID)
				text = fmt.Sprintf("📋 %s 🔗 <a href=\"%s\">%s</a>\n\n", sl.Name, link, escapeHTML(chatName)) + text[len(fmt.Sprintf("📋 %s\n\n", sl.Name)):]
			} else {
				text = fmt.Sprintf("📋 %s · %s\n\n", sl.Name, escapeHTML(chatName)) + text[len(fmt.Sprintf("📋 %s\n\n", sl.Name)):]
			}
		}
		kb := SetlistCardKeyboardReadonly(sl, KeyboardOpts{BackOrigin: origin})
		return c.Edit(text, kb, tele.ModeHTML)
	}

	text := RenderSetlistCard(sl)
	kb := SetlistCardKeyboard(sl, KeyboardOpts{BackOrigin: origin})
	return c.Edit(text, kb, tele.ModeHTML)
}

func (b *Bot) processRenameSetlistReply(c tele.Context, slID int) error {
	newName := normalize.SongName(strings.TrimSpace(c.Text()))
	if newName == "" {
		return c.Send("Название не может быть пустым.")
	}

	ctx := context.Background()
	sl, _ := b.store.GetSetlistByID(ctx, slID)
	if sl == nil {
		return c.Send("Сетлист не найден.")
	}

	if !b.auth.CanEditSetlist(ctx, c.Sender().ID, sl) {
		return c.Send("Недостаточно прав.")
	}

	existing, _ := b.store.GetSetlist(ctx, sl.ChatID, newName)
	if existing != nil {
		return c.Send(fmt.Sprintf("Сетлист «%s» уже существует.", newName))
	}

	oldName := sl.Name
	_ = b.store.UpdateSetlistName(ctx, slID, newName)

	sl, _ = b.store.GetSetlistByID(ctx, slID)
	ccList, _ := b.store.GetSubscribeAllUsers(ctx, sl.ChatID)
	header := fmt.Sprintf("Сетлист переименован: %s → %s", oldName, newName)
	if cc := FormatCCLine(ccList); cc != "" {
		header += "\n" + cc
	}
	text := header + "\n\n──────────────────\n" + RenderSetlistCard(sl)
	kb := SetlistCardKeyboard(sl)
	return c.Send(text, kb, tele.ModeHTML)
}

func (b *Bot) processSetlistSongsReply(c tele.Context, slID int) error {
	lines := strings.Split(c.Text(), "\n")
	var songLines []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			songLines = append(songLines, l)
		}
	}
	if len(songLines) == 0 {
		return c.Send("Укажите хотя бы одну песню.")
	}

	ctx := context.Background()
	sl, _ := b.store.GetSetlistByID(ctx, slID)
	if sl == nil {
		return c.Send("Сетлист не найден.")
	}

	allNames, _ := b.store.ListSongNames(ctx, sl.ChatID)
	allSongs, _ := b.store.ListSongs(ctx, sl.ChatID)
	nameToSong := make(map[string]*model.Song)
	for i := range allSongs {
		nameToSong[allSongs[i].Name] = &allSongs[i]
	}

	resolutions := make([]songResolution, len(songLines))
	var needDisambig []int
	var notFound []string

	for i, line := range songLines {
		norm := normalize.SongName(line)
		resolutions[i] = songResolution{input: line, normalized: norm}

		if s, ok := nameToSong[norm]; ok {
			resolutions[i].songID = s.ID
			continue
		}

		candidates := fuzzy.FindCandidates(norm, allNames, 0.4)
		if len(candidates) == 0 {
			notFound = append(notFound, line)
		} else if len(candidates) == 1 {
			if s, ok := nameToSong[candidates[0].Name]; ok {
				resolutions[i].songID = s.ID
			}
		} else {
			resolutions[i].candidates = candidates
			needDisambig = append(needDisambig, i)
		}
	}

	if len(notFound) > 0 {
		return c.Send("Не удалось найти песни:\n• " + strings.Join(notFound, "\n• "))
	}

	if len(needDisambig) > 0 {
		return b.startDisambiguationForUpdate(c, slID, resolutions, needDisambig, nameToSong)
	}

	var songIDs []int
	for _, r := range resolutions {
		songIDs = append(songIDs, r.songID)
	}
	_ = b.store.UpdateSetlistSongs(ctx, slID, songIDs)

	sl, _ = b.store.GetSetlistByID(ctx, slID)
	ccList, _ := b.store.GetSubscribeAllUsers(ctx, sl.ChatID)
	return b.sendSetlistCardWithCC(c, sl, ccList)
}

func (b *Bot) processCreateSetlistReply(c tele.Context, name string) error {
	lines := strings.Split(c.Text(), "\n")
	var songLines []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			songLines = append(songLines, l)
		}
	}
	if len(songLines) == 0 {
		return c.Send("Укажите хотя бы одну песню.")
	}

	return b.resolveAndCreateSetlist(c, name, songLines)
}

func (b *Bot) startDisambiguationForUpdate(c tele.Context, slID int, resolutions []songResolution, pending []int, nameToSong map[string]*model.Song) error {
	idx := pending[0]
	res := resolutions[idx]

	rm := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, cand := range res.candidates {
		rows = append(rows, tele.Row{
			rm.Data(cand.Name, "sl_pick", fmt.Sprintf("%d|%s", idx, cand.Name)),
		})
	}
	rm.Inline(rows...)

	msg, err := b.tele.Send(c.Chat(),
		fmt.Sprintf("Для «%s» найдено несколько вариантов. Выберите:", res.input), rm)
	if err != nil {
		return err
	}

	disambigStore[disambigKey(c.Chat().ID, msg.ID)] = &disambigState{
		setlistName: fmt.Sprintf("__update__%d", slID),
		resolutions: resolutions,
		pending:     pending,
		current:     0,
		nameToSong:  nameToSong,
	}

	return nil
}
