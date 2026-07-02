package bot

import (
	"context"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/julesimf/bandbot/internal/auth"
	"github.com/julesimf/bandbot/internal/storage"
	tele "gopkg.in/telebot.v3"
)

type Bot struct {
	tele       *tele.Bot
	store      storage.Storage
	auth       auth.Authorizer
	prompts    *PromptStore
	membership *MembershipCache
}

func New(token string, store storage.Storage) (*Bot, error) {
	pref := tele.Settings{
		Token:     token,
		Poller:    &tele.LongPoller{Timeout: 10 * time.Second},
		ParseMode: tele.ModeHTML,
	}

	tb, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	b := &Bot{
		tele:       tb,
		store:      store,
		auth:       auth.AllowAll{},
		prompts:    NewPromptStore(),
		membership: NewMembershipCache(tb),
	}

	b.registerHandlers()
	return b, nil
}

func (b *Bot) Start() {
	log.Println("Bot started")
	b.tele.Start()
}

func (b *Bot) Stop() {
	b.tele.Stop()
}

func (b *Bot) registerHandlers() {
	b.tele.Handle("/start", b.handleStart)
	b.tele.Handle("/help", b.handleHelp)
	b.tele.Handle("/song", b.handleSong)
	b.tele.Handle("/setlist", b.handleSetlist)
	b.tele.Handle("/all_songs", b.handleAllSongs)
	b.tele.Handle("/all_setlists", b.handleAllSetlists)
	b.tele.Handle("/toggle_subscribe_all", b.handleToggleSubscribeAll)

	b.tele.Handle(tele.OnQuery, b.handleInlineQuery)

	b.tele.Handle(&tele.InlineButton{Unique: "key"}, b.handleKeySelect)
	b.tele.Handle(&tele.InlineButton{Unique: "set_key"}, b.handleSetKey)
	b.tele.Handle(&tele.InlineButton{Unique: "key_back"}, b.handleKeyBack)
	b.tele.Handle(&tele.InlineButton{Unique: "nav_back"}, b.handleNavBack)
	b.tele.Handle(&tele.InlineButton{Unique: "tempo"}, b.handleTempoPrompt)
	b.tele.Handle(&tele.InlineButton{Unique: "rename_song"}, b.handleRenameSongPrompt)
	b.tele.Handle(&tele.InlineButton{Unique: "responsible"}, b.handleResponsiblePrompt)
	b.tele.Handle(&tele.InlineButton{Unique: "sub"}, b.handleSubscribe)
	b.tele.Handle(&tele.InlineButton{Unique: "unsub"}, b.handleUnsubscribe)
	b.tele.Handle(&tele.InlineButton{Unique: "history"}, b.handleHistory)
	b.tele.Handle(&tele.InlineButton{Unique: "delete_song"}, b.handleDeleteSong)
	b.tele.Handle(&tele.InlineButton{Unique: "del_note"}, b.handleDeleteNoteSelect)
	b.tele.Handle(&tele.InlineButton{Unique: "clear_notes"}, b.handleClearNotes)
	b.tele.Handle(&tele.InlineButton{Unique: "del_pin"}, b.handleDeletePinSelect)
	b.tele.Handle(&tele.InlineButton{Unique: "clear_pins"}, b.handleClearPins)
	b.tele.Handle(&tele.InlineButton{Unique: "rm_note"}, b.handleRemoveNote)
	b.tele.Handle(&tele.InlineButton{Unique: "rm_pin"}, b.handleRemovePin)

	b.tele.Handle(&tele.InlineButton{Unique: "show_song"}, b.handleShowSongCallback)
	b.tele.Handle(&tele.InlineButton{Unique: "rename_sl"}, b.handleRenameSetlistPrompt)
	b.tele.Handle(&tele.InlineButton{Unique: "edit_sl"}, b.handleEditSetlistPrompt)
	b.tele.Handle(&tele.InlineButton{Unique: "active_sl"}, b.handleSetActiveSetlist)
	b.tele.Handle(&tele.InlineButton{Unique: "delete_sl"}, b.handleDeleteSetlist)
	b.tele.Handle(&tele.InlineButton{Unique: "sl_pick"}, b.handleSetlistPick)
	b.tele.Handle(&tele.InlineButton{Unique: "show_sl"}, b.handleShowSetlistCallback)

	b.tele.Handle(tele.OnText, b.handleText)
}

func (b *Bot) ensureChat(c tele.Context) error {
	return b.store.EnsureChat(context.Background(), c.Chat().ID, c.Chat().Title)
}

func isPrivateChat(c tele.Context) bool {
	return c.Chat().Type == tele.ChatPrivate
}

const privateChatEditError = "Доступно только в групповом чате"

func (b *Bot) getUserChatIDs(userID int64) []int64 {
	ctx := context.Background()
	allIDs, err := b.store.ListAllChatIDs(ctx)
	if err != nil {
		return nil
	}
	return b.membership.GetUserChats(userID, allIDs)
}

func senderUsername(c tele.Context) string {
	if c.Sender() != nil {
		return c.Sender().Username
	}
	return ""
}

func senderDisplayName(c tele.Context) string {
	if c.Sender() != nil {
		if c.Sender().Username != "" {
			return c.Sender().Username
		}
		name := c.Sender().FirstName
		if c.Sender().LastName != "" {
			name += " " + c.Sender().LastName
		}
		return name
	}
	return "unknown"
}

var notePattern = regexp.MustCompile(`(?si)^#примечание\s+(.+?)\n(.+)$`)
var pinPattern = regexp.MustCompile(`(?si)^#закреп\s+(.+?)\n(.+)$`)

func (b *Bot) handleText(c tele.Context) error {
	text := c.Text()

	if strings.HasPrefix(strings.ToLower(text), "#примечание") {
		if isPrivateChat(c) {
			return c.Send(privateChatEditError)
		}
		return b.handleNote(c)
	}
	if strings.HasPrefix(strings.ToLower(text), "#закреп") {
		if isPrivateChat(c) {
			return c.Send(privateChatEditError)
		}
		return b.handlePin(c)
	}

	if c.Message().ReplyTo != nil {
		return b.handlePromptReply(c)
	}

	return nil
}
