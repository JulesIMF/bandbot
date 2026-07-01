package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/julesimf/bandbot/internal/model"
	tele "gopkg.in/telebot.v3"
)

func isSupergroup(chatID int64) bool {
	return chatID < -1000000000000
}

func pinLink(chatID, messageID int64) string {
	s := strconv.FormatInt(chatID, 10)
	if strings.HasPrefix(s, "-100") {
		s = s[4:]
	}
	return fmt.Sprintf("https://t.me/c/%s/%d", s, messageID)
}

type ChangeHeader struct {
	TempoOld       *int
	TempoNew       *int
	KeyOld         *string
	KeyNew         *string
	NameOld        string
	NameNew        string
	ResponsibleOld string
	ResponsibleNew string
}

func renderChangeHeader(ch *ChangeHeader) string {
	if ch == nil {
		return ""
	}
	var lines []string

	if ch.NameOld != "" && ch.NameNew != "" {
		lines = append(lines, fmt.Sprintf("Название: %s → %s", ch.NameOld, ch.NameNew))
	}

	if ch.TempoNew != nil {
		if ch.TempoOld != nil {
			lines = append(lines, fmt.Sprintf("Темп: %d bpm → %d bpm", *ch.TempoOld, *ch.TempoNew))
		} else {
			lines = append(lines, fmt.Sprintf("Темп: %d bpm", *ch.TempoNew))
		}
	}

	if ch.KeyNew != nil {
		if ch.KeyOld != nil {
			lines = append(lines, fmt.Sprintf("Тональность: %s → %s", *ch.KeyOld, *ch.KeyNew))
		} else {
			lines = append(lines, fmt.Sprintf("Тональность: %s", *ch.KeyNew))
		}
	}

	if ch.ResponsibleNew != "" {
		if ch.ResponsibleOld != "" {
			lines = append(lines, fmt.Sprintf("Ответственный: %s → %s", ch.ResponsibleOld, ch.ResponsibleNew))
		} else {
			lines = append(lines, fmt.Sprintf("Ответственный: %s", ch.ResponsibleNew))
		}
	}

	return strings.Join(lines, "\n")
}

func RenderSongCard(song *model.Song, changes *ChangeHeader, ccList []string) string {
	var b strings.Builder

	if changes != nil {
		header := renderChangeHeader(changes)
		if header != "" {
			b.WriteString(header)
			b.WriteString("\n")
			if len(ccList) > 0 {
				b.WriteString("\ncc")
				for _, u := range ccList {
					b.WriteString(" @")
					b.WriteString(u)
				}
				b.WriteString("\n")
			}
			b.WriteString("\n──────────────────\n")
		}
	}

	b.WriteString(fmt.Sprintf("🎵 %s\n", song.Name))

	if song.Tempo != nil {
		b.WriteString(fmt.Sprintf("Темп: %d bpm\n", *song.Tempo))
	}
	if song.Key != nil {
		b.WriteString(fmt.Sprintf("Тональность: %s\n", *song.Key))
	}
	if song.Responsible != "" {
		b.WriteString(fmt.Sprintf("Ответственный: %s\n", song.Responsible))
	}

	if len(song.Notes) > 0 {
		b.WriteString("\n")
		for _, n := range song.Notes {
			b.WriteString(fmt.Sprintf("• %s\n", n.Content))
		}
	}

	if len(song.Pins) > 0 {
		b.WriteString("\n")
		for _, p := range song.Pins {
			if isSupergroup(p.ChatID) {
				link := pinLink(p.ChatID, p.MessageID)
				b.WriteString(fmt.Sprintf("📎 <a href=\"%s\">%s</a>\n", link, escapeHTML(p.Label)))
			} else {
				b.WriteString(fmt.Sprintf("📎 %s\n", escapeHTML(p.Label)))
			}
		}
	}

	return b.String()
}

func SongCardKeyboard(song *model.Song, isSubscribed bool) *tele.ReplyMarkup {
	rm := &tele.ReplyMarkup{}

	row1 := tele.Row{
		rm.Data("Тональность", "key", fmt.Sprintf("%d", song.ID)),
		rm.Data("Темп", "tempo", fmt.Sprintf("%d", song.ID)),
	}
	row2 := tele.Row{
		rm.Data("Переименовать", "rename_song", fmt.Sprintf("%d", song.ID)),
		rm.Data("Ответственный", "responsible", fmt.Sprintf("%d", song.ID)),
	}

	subText := "Подписаться"
	subAction := "sub"
	if isSubscribed {
		subText = "Отписаться"
		subAction = "unsub"
	}
	row3 := tele.Row{
		rm.Data(subText, subAction, fmt.Sprintf("%d", song.ID)),
		rm.Data("История", "history", fmt.Sprintf("%d", song.ID)),
	}

	rows := []tele.Row{row1, row2, row3}

	if len(song.Notes) > 0 {
		rows = append(rows, tele.Row{
			rm.Data("Удалить примечание", "del_note", fmt.Sprintf("%d", song.ID)),
			rm.Data("Очистить примечания", "clear_notes", fmt.Sprintf("%d", song.ID)),
		})
	}

	if len(song.Pins) > 0 {
		rows = append(rows, tele.Row{
			rm.Data("Удалить закреп", "del_pin", fmt.Sprintf("%d", song.ID)),
			rm.Data("Очистить закрепы", "clear_pins", fmt.Sprintf("%d", song.ID)),
		})
	}

	rows = append(rows, tele.Row{
		rm.Data("Удалить песню", "delete_song", fmt.Sprintf("%d", song.ID)),
	})

	rm.Inline(rows...)
	return rm
}

func FormatSongLine(song *model.Song) string {
	var details []string
	if song.Tempo != nil {
		details = append(details, fmt.Sprintf("%d bpm", *song.Tempo))
	}
	if song.Key != nil {
		details = append(details, *song.Key)
	}
	if len(details) > 0 {
		return song.Name + " · " + strings.Join(details, ", ")
	}
	return song.Name
}

func RenderSongList(songs []model.Song) string {
	var b strings.Builder
	for _, s := range songs {
		b.WriteString(FormatSongLine(&s))
		b.WriteString("\n")
	}
	return b.String()
}

func RenderSetlistCard(sl *model.Setlist) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("📋 %s\n\n", sl.Name))

	for _, e := range sl.Songs {
		if e.Song == nil {
			continue
		}
		b.WriteString(fmt.Sprintf("%d. %s\n", e.Position, FormatSongLine(e.Song)))
	}

	return b.String()
}

func SetlistCardKeyboard(sl *model.Setlist) *tele.ReplyMarkup {
	rm := &tele.ReplyMarkup{}

	var songRows []tele.Row
	for _, e := range sl.Songs {
		if e.Song == nil {
			continue
		}
		songRows = append(songRows, tele.Row{
			rm.Data(
				fmt.Sprintf("%d. %s", e.Position, e.Song.Name),
				"show_song",
				fmt.Sprintf("%d", e.Song.ID),
			),
		})
	}

	actionRows := []tele.Row{
		{
			rm.Data("Переименовать", "rename_sl", fmt.Sprintf("%d", sl.ID)),
			rm.Data("Изменить список", "edit_sl", fmt.Sprintf("%d", sl.ID)),
		},
		{
			rm.Data("Назначить активным", "active_sl", fmt.Sprintf("%d", sl.ID)),
			rm.Data("Удалить", "delete_sl", fmt.Sprintf("%d", sl.ID)),
		},
	}

	allRows := append(songRows, actionRows...)
	rm.Inline(allRows...)
	return rm
}

func RenderSongDeleted(song *model.Song) string {
	card := RenderSongCard(song, nil, nil)
	return card + "\n❌ Песня удалена"
}

func RenderSetlistDeleted(sl *model.Setlist) string {
	card := RenderSetlistCard(sl)
	return card + "\n❌ Сетлист удалён"
}

func FormatCCLine(usernames []string) string {
	if len(usernames) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("cc")
	for _, u := range usernames {
		b.WriteString(" @")
		b.WriteString(u)
	}
	return b.String()
}

func RenderHistory(song *model.Song, history []model.SongHistory) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("📜 История изменений: %s\n\n", song.Name))

	if len(history) == 0 {
		b.WriteString("Изменений пока нет.")
		return b.String()
	}

	fieldNames := map[string]string{
		"tempo":       "Темп",
		"key":         "Тональность",
		"name":        "Название",
		"responsible": "Ответственный",
		"note":        "Примечание",
		"pin":         "Закреп",
	}

	for _, h := range history {
		fname := fieldNames[h.Field]
		if fname == "" {
			fname = h.Field
		}
		ts := h.ChangedAt.Format("02.01.2006 15:04")
		if h.OldValue != nil {
			b.WriteString(fmt.Sprintf("%s — %s: %s → %s (%s)\n",
				ts, fname, *h.OldValue, h.NewValue, h.ChangedBy))
		} else {
			b.WriteString(fmt.Sprintf("%s — %s: %s (%s)\n",
				ts, fname, h.NewValue, h.ChangedBy))
		}
	}

	return b.String()
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
