package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/julesimf/bandbot/internal/model"
	tele "gopkg.in/telebot.v3"
)

var screenOriginStore = struct {
	sync.Mutex
	m map[string]string
}{m: make(map[string]string)}

func originKey(chatID int64, msgID int, screenType string) string {
	return fmt.Sprintf("%d:%d:%s", chatID, msgID, screenType)
}

func setScreenOrigin(chatID int64, msgID int, screenType, origin string) {
	key := originKey(chatID, msgID, screenType)
	screenOriginStore.Lock()
	defer screenOriginStore.Unlock()
	if origin == "" {
		delete(screenOriginStore.m, key)
	} else {
		screenOriginStore.m[key] = origin
	}
}

func getScreenOrigin(chatID int64, msgID int, screenType string) string {
	key := originKey(chatID, msgID, screenType)
	screenOriginStore.Lock()
	defer screenOriginStore.Unlock()
	return screenOriginStore.m[key]
}

func songOriginOpts(c tele.Context) KeyboardOpts {
	origin := getScreenOrigin(c.Chat().ID, c.Message().ID, "song")
	return KeyboardOpts{BackOrigin: origin}
}

func setlistOriginOpts(c tele.Context) KeyboardOpts {
	origin := getScreenOrigin(c.Chat().ID, c.Message().ID, "setlist")
	return KeyboardOpts{BackOrigin: origin}
}

func (b *Bot) handleNavBack(c tele.Context) error {
	data := c.Callback().Data
	parts := strings.SplitN(data, "|", 2)
	if len(parts) < 2 {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	screenType := parts[0]
	screenID := parts[1]
	_ = c.Respond()

	switch screenType {
	case "song":
		return b.navToSong(c, screenID)
	case "setlist":
		return b.navToSetlist(c, screenID)
	case "song_list":
		return b.navToSongList(c, screenID)
	case "setlist_list":
		return b.navToSetlistList(c, screenID)
	default:
		return c.Respond(&tele.CallbackResponse{Text: "Неизвестный экран"})
	}
}

func (b *Bot) navToSong(c tele.Context, idStr string) error {
	songID, err := strconv.Atoi(idStr)
	if err != nil {
		return nil
	}

	ctx := context.Background()
	song, err := b.store.GetSongByID(ctx, songID)
	if err != nil || song == nil {
		return nil
	}

	opts := songOriginOpts(c)

	if isPrivateChat(c) {
		chatName := ""
		if n, _ := b.store.GetChatName(ctx, song.ChatID); n != nil {
			chatName = *n
		}
		text := RenderSongCardWithChat(song, nil, nil, chatName)
		isSubbed, _ := b.store.IsSubscribed(ctx, song.ID, c.Sender().ID)
		kb := SongCardKeyboardReadonly(song, isSubbed, opts)
		return c.Edit(text, kb, tele.ModeHTML)
	}

	text := RenderSongCard(song, nil, nil)
	isSubbed, _ := b.store.IsSubscribed(ctx, song.ID, c.Sender().ID)
	kb := SongCardKeyboard(song, isSubbed, opts)
	return c.Edit(text, kb, tele.ModeHTML)
}

func (b *Bot) navToSetlist(c tele.Context, idStr string) error {
	slID, err := strconv.Atoi(idStr)
	if err != nil {
		return nil
	}

	ctx := context.Background()
	sl, _ := b.store.GetSetlistByID(ctx, slID)
	if sl == nil {
		return nil
	}

	opts := setlistOriginOpts(c)

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
		kb := SetlistCardKeyboardReadonly(sl, opts)
		return c.Edit(text, kb, tele.ModeHTML)
	}

	text := RenderSetlistCard(sl)
	kb := SetlistCardKeyboard(sl, opts)
	return c.Edit(text, kb, tele.ModeHTML)
}

func (b *Bot) navToSongList(c tele.Context, chatIDStr string) error {
	chatID, _ := strconv.ParseInt(chatIDStr, 10, 64)
	ctx := context.Background()

	if chatID == 0 || isPrivateChat(c) {
		chatIDs := b.getUserChatIDs(c.Sender().ID)
		songs, err := b.store.SearchSongsInChats(ctx, chatIDs, "", 20)
		if err != nil || len(songs) == 0 {
			return c.Edit("Песен пока нет.")
		}
		text := "🎵 Недавние песни:\n\n" + RenderSongList(songs)
		rm := songListKeyboard(songs, "song_list|0")
		return c.Edit(text, rm, tele.ModeHTML)
	}

	songs, err := b.store.SearchSongs(ctx, chatID, "", 20)
	if err != nil || len(songs) == 0 {
		return c.Edit("Песен пока нет.")
	}
	text := "🎵 Недавние песни:\n\n" + RenderSongList(songs)
	origin := fmt.Sprintf("song_list|%d", chatID)
	rm := songListKeyboard(songs, origin)
	return c.Edit(text, rm, tele.ModeHTML)
}

func (b *Bot) navToSetlistList(c tele.Context, chatIDStr string) error {
	chatID, _ := strconv.ParseInt(chatIDStr, 10, 64)
	ctx := context.Background()

	if chatID == 0 || isPrivateChat(c) {
		chatIDs := b.getUserChatIDs(c.Sender().ID)
		setlists, err := b.store.ListSetlistsInChats(ctx, chatIDs)
		if err != nil || len(setlists) == 0 {
			return c.Edit("Сетлистов пока нет.")
		}
		rm := setlistListKeyboard(setlists, "setlist_list|0")
		return c.Edit("📋 Все сетлисты:", rm, tele.ModeHTML)
	}

	setlists, err := b.store.ListSetlists(ctx, chatID)
	if err != nil || len(setlists) == 0 {
		return c.Edit("Сетлистов пока нет.")
	}
	origin := fmt.Sprintf("setlist_list|%d", chatID)
	rm := setlistListKeyboard(setlists, origin)
	return c.Edit("📋 Все сетлисты:", rm, tele.ModeHTML)
}

func songListKeyboard(songs []model.Song, listOrigin string) *tele.ReplyMarkup {
	rm := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, s := range songs {
		rows = append(rows, tele.Row{
			rm.Data(s.Name, "show_song", fmt.Sprintf("%d|%s", s.ID, listOrigin)),
		})
	}
	rm.Inline(rows...)
	return rm
}

func setlistListKeyboard(setlists []model.Setlist, listOrigin string) *tele.ReplyMarkup {
	rm := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, sl := range setlists {
		rows = append(rows, tele.Row{
			rm.Data(sl.Name, "show_sl", fmt.Sprintf("%d|%s", sl.ID, listOrigin)),
		})
	}
	rm.Inline(rows...)
	return rm
}
