package thuchi

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	telegramAPI       = "https://api.telegram.org/bot"
	telegramCodeTTL   = 15 * time.Minute
	telegramHintEvery = 6 * time.Hour
	telegramMaxItems  = 20
	telegramPickLimit = 24
)

var (
	errTelegramDup      = errors.New("telegram: duplicate message")
	errTelegramInactive = errors.New("telegram: owner inactive")
)

const telegramInactiveText = "Tiện ích Thu chi của tài khoản này đang tắt. Bật lại trong app để ghi tiếp."

type TelegramBot struct {
	APIBase     string
	WebhookURL  string
	OwnerActive func(ctx context.Context, ownerID string) bool

	svc     *Service
	token   string
	secret  string
	chats   map[int64]bool
	client  *http.Client
	updates chan tgUpdate

	mu         sync.Mutex
	username   string
	readsGroup bool
	hinted     map[int64]time.Time
}

func NewTelegramBot(svc *Service, token string, chatIDs []int64, webhookURL string) *TelegramBot {
	chats := map[int64]bool{}
	for _, id := range chatIDs {
		chats[id] = true
	}
	b := &TelegramBot{
		APIBase:    telegramAPI,
		WebhookURL: strings.TrimSpace(webhookURL),
		svc:        svc,
		token:      token,
		secret:     randomSecret(),
		chats:      chats,
		client:     &http.Client{Timeout: 70 * time.Second},
		updates:    make(chan tgUpdate, 100),
		hinted:     map[int64]time.Time{},
	}
	svc.Telegram = b
	return b
}

func (b *TelegramBot) Username() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.username
}

func (b *TelegramBot) ReadsGroupMessages() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.readsGroup
}

func (b *TelegramBot) Mode() string {
	if b.WebhookURL != "" {
		return "webhook"
	}
	return "polling"
}

func (b *TelegramBot) ServeWebhook(w http.ResponseWriter, r *http.Request) {
	if b.WebhookURL == "" || !hmac.Equal([]byte(r.Header.Get("X-Telegram-Bot-Api-Secret-Token")), []byte(b.secret)) {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var u tgUpdate
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&u); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	select {
	case b.updates <- u:
		w.WriteHeader(http.StatusOK)
	default:
		writeErr(w, http.StatusServiceUnavailable, "busy")
	}
}

type tgUser struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

func (u *tgUser) displayName() string {
	return strings.TrimSpace(u.FirstName + " " + u.LastName)
}

type tgChat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

type tgMessage struct {
	MessageID int64      `json:"message_id"`
	From      *tgUser    `json:"from"`
	Chat      tgChat     `json:"chat"`
	Date      int64      `json:"date"`
	Text      string     `json:"text"`
	ReplyTo   *tgMessage `json:"reply_to_message"`
}

type tgCallback struct {
	ID      string     `json:"id"`
	From    tgUser     `json:"from"`
	Message *tgMessage `json:"message"`
	Data    string     `json:"data"`
}

type tgUpdate struct {
	UpdateID      int64       `json:"update_id"`
	Message       *tgMessage  `json:"message"`
	EditedMessage *tgMessage  `json:"edited_message"`
	CallbackQuery *tgCallback `json:"callback_query"`
}

type tgButton struct {
	Text string `json:"text"`
	Data string `json:"callback_data,omitempty"`
	URL  string `json:"url,omitempty"`
}

type tgMarkup struct {
	Keyboard [][]tgButton `json:"inline_keyboard"`
}

func (b *TelegramBot) call(ctx context.Context, method string, params any, out any) error {
	body, err := json.Marshal(params)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.APIBase+b.token+"/"+method, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram %s: build request", method)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return fmt.Errorf("telegram %s: %w", method, err)
	}
	defer resp.Body.Close()
	var r struct {
		OK          bool            `json:"ok"`
		Result      json.RawMessage `json:"result"`
		Description string          `json:"description"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&r); err != nil {
		return fmt.Errorf("telegram %s: decode: %w", method, err)
	}
	if !r.OK {
		return fmt.Errorf("telegram %s: %s", method, r.Description)
	}
	if out != nil {
		return json.Unmarshal(r.Result, out)
	}
	return nil
}

func sleepCtx(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

func (b *TelegramBot) Run(ctx context.Context) {
	for ctx.Err() == nil {
		var me struct {
			Username   string `json:"username"`
			ReadsGroup bool   `json:"can_read_all_group_messages"`
		}
		if err := b.call(ctx, "getMe", map[string]any{}, &me); err != nil {
			log.Printf("thuchi: telegram getMe: %v", err)
			sleepCtx(ctx, 30*time.Second)
			continue
		}
		b.mu.Lock()
		b.username, b.readsGroup = me.Username, me.ReadsGroup
		b.mu.Unlock()
		if !me.ReadsGroup {
			log.Printf("thuchi: telegram @%s đang bật privacy mode: trong nhóm chỉ nhận lệnh và tin trả lời bot (BotFather /setprivacy → Disable)", me.Username)
		}
		break
	}
	_ = b.call(ctx, "setMyCommands", map[string]any{"commands": []map[string]string{
		{"command": "tong", "description": "Tổng thu chi tháng này"},
		{"command": "huongdan", "description": "Cách ghi thu chi"},
	}}, nil)
	allowed := []string{"message", "edited_message", "callback_query"}
	if b.WebhookURL != "" {
		b.runWebhook(ctx, allowed)
		return
	}
	if err := b.call(ctx, "deleteWebhook", map[string]any{"drop_pending_updates": false}, nil); err != nil {
		log.Printf("thuchi: telegram deleteWebhook: %v", err)
	}
	log.Printf("thuchi: telegram @%s nhận tin bằng long polling", b.Username())
	var offset int64
	for ctx.Err() == nil {
		var updates []tgUpdate
		err := b.call(ctx, "getUpdates", map[string]any{"offset": offset, "timeout": 50, "allowed_updates": allowed}, &updates)
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("thuchi: telegram getUpdates: %v", err)
				sleepCtx(ctx, 5*time.Second)
			}
			continue
		}
		for _, u := range updates {
			offset = u.UpdateID + 1
			b.handleUpdate(ctx, u)
		}
	}
}

func (b *TelegramBot) runWebhook(ctx context.Context, allowed []string) {
	for ctx.Err() == nil {
		err := b.call(ctx, "setWebhook", map[string]any{"url": b.WebhookURL, "secret_token": b.secret, "allowed_updates": allowed, "max_connections": 4}, nil)
		if err == nil {
			log.Printf("thuchi: telegram @%s nhận tin qua webhook %s", b.Username(), b.WebhookURL)
			break
		}
		log.Printf("thuchi: telegram setWebhook: %v", err)
		sleepCtx(ctx, 30*time.Second)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case u := <-b.updates:
			b.handleUpdate(ctx, u)
		}
	}
}

func (b *TelegramBot) handleUpdate(ctx context.Context, u tgUpdate) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("thuchi: telegram update %d panic: %v", u.UpdateID, r)
		}
	}()
	switch {
	case u.Message != nil:
		b.handleMessage(ctx, u.Message, false)
	case u.EditedMessage != nil:
		b.handleMessage(ctx, u.EditedMessage, true)
	case u.CallbackQuery != nil:
		b.handleCallback(ctx, u.CallbackQuery)
	}
}

func (b *TelegramBot) send(ctx context.Context, chatID, replyTo int64, text string, markup *tgMarkup) (*tgMessage, error) {
	params := map[string]any{
		"chat_id":              chatID,
		"text":                 text,
		"parse_mode":           "HTML",
		"disable_notification": true,
		"link_preview_options": map[string]bool{"is_disabled": true},
	}
	if replyTo != 0 {
		params["reply_parameters"] = map[string]any{"message_id": replyTo, "allow_sending_without_reply": true}
	}
	if markup != nil {
		params["reply_markup"] = markup
	}
	var msg tgMessage
	if err := b.call(ctx, "sendMessage", params, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (b *TelegramBot) edit(ctx context.Context, chatID, messageID int64, text string, markup *tgMarkup) {
	params := map[string]any{
		"chat_id":              chatID,
		"message_id":           messageID,
		"text":                 text,
		"parse_mode":           "HTML",
		"link_preview_options": map[string]bool{"is_disabled": true},
	}
	if markup == nil {
		markup = &tgMarkup{Keyboard: [][]tgButton{}}
	}
	params["reply_markup"] = markup
	if err := b.call(ctx, "editMessageText", params, nil); err != nil && !strings.Contains(err.Error(), "message is not modified") {
		log.Printf("thuchi: telegram edit: %v", err)
	}
}

func (b *TelegramBot) react(ctx context.Context, chatID, messageID int64, emoji string) {
	_ = b.call(ctx, "setMessageReaction", map[string]any{"chat_id": chatID, "message_id": messageID, "reaction": []map[string]string{{"type": "emoji", "emoji": emoji}}}, nil)
}

func (b *TelegramBot) answer(ctx context.Context, id, text string, alert bool) {
	_ = b.call(ctx, "answerCallbackQuery", map[string]any{"callback_query_id": id, "text": text, "show_alert": alert}, nil)
}

func (b *TelegramBot) reply(ctx context.Context, m *tgMessage, text string) {
	if _, err := b.send(ctx, m.Chat.ID, m.MessageID, text, nil); err != nil {
		log.Printf("thuchi: telegram send: %v", err)
	}
}

func (b *TelegramBot) findLink(ctx context.Context, telegramUserID int64) (*TelegramLink, error) {
	link, err := b.svc.Store.FindTelegramLink(ctx, telegramUserID)
	if err != nil {
		return nil, err
	}
	if b.OwnerActive != nil && !b.OwnerActive(ctx, link.OwnerID) {
		return nil, errTelegramInactive
	}
	return link, nil
}

func (b *TelegramBot) allowedChat(c tgChat) bool {
	if c.Type == "private" {
		return true
	}
	return len(b.chats) == 0 || b.chats[c.ID]
}

func (b *TelegramBot) handleMessage(ctx context.Context, m *tgMessage, edited bool) {
	if m.From == nil || m.From.IsBot || !b.allowedChat(m.Chat) {
		return
	}
	text := b.stripMention(m.Text)
	if text == "" {
		return
	}
	private := m.Chat.Type == "private"
	if strings.HasPrefix(text, "/") {
		if !edited {
			b.handleCommand(ctx, m, text, private)
		}
		return
	}
	link, err := b.findLink(ctx, m.From.ID)
	if errors.Is(err, ErrNotFound) {
		if private || len(ParseQuick(text, time.Unix(m.Date, 0))) > 0 {
			b.hintLink(ctx, m, private)
		}
		return
	}
	if errors.Is(err, errTelegramInactive) {
		if !edited {
			b.reply(ctx, m, telegramInactiveText)
		}
		return
	}
	if err != nil {
		log.Printf("thuchi: telegram find link: %v", err)
		return
	}
	if edited {
		b.handleEdit(ctx, link, m, text)
		return
	}
	if m.ReplyTo != nil && b.handleReply(ctx, link, m, text) {
		return
	}
	b.handleEntry(ctx, link, m, text, private)
}

const telegramHelp = `<b>Ghi thu chi bằng tin nhắn</b>, mỗi dòng một khoản:
• <code>cafe 15k</code>
• <code>ăn trưa 45k, gửi xe 5k</code>
• <code>+2tr lương</code> (dấu + là khoản thu)
• <code>hôm qua grab 32k</code> · <code>20/10 tiền điện 450k</code>

Mình tự chọn danh mục. Sai thì bấm ✏️ chọn lại, lần sau mình nhớ.
Sửa nhanh: trả lời tin của mình bằng tên danh mục, số tiền mới (<code>20k</code>) hoặc <code>xoá</code>. Sửa tin gốc thì giao dịch cũng sửa theo.
/tong — tổng thu chi tháng này`

func (b *TelegramBot) stripMention(text string) string {
	mention := "@" + strings.ToLower(b.Username())
	if mention == "@" {
		return strings.TrimSpace(text)
	}
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		var keep []string
		for _, f := range strings.Fields(line) {
			if strings.ToLower(strings.Trim(f, ",.:")) != mention {
				keep = append(keep, f)
			}
		}
		lines = append(lines, strings.Join(keep, " "))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func (b *TelegramBot) handleCommand(ctx context.Context, m *tgMessage, text string, private bool) {
	fields := strings.Fields(text)
	cmd, target, _ := strings.Cut(strings.ToLower(fields[0]), "@")
	if target != "" && !strings.EqualFold(target, b.Username()) {
		return
	}
	arg := ""
	if len(fields) > 1 {
		arg = fields[1]
	}
	switch cmd {
	case "/start", "/link", "/ketnoi":
		if arg != "" {
			b.claim(ctx, m, arg)
			return
		}
		if cmd == "/start" && private {
			if _, err := b.svc.Store.FindTelegramLink(ctx, m.From.ID); errors.Is(err, ErrNotFound) {
				b.reply(ctx, m, "👋 Chào bạn! Mở app Thu Chi → <b>Cài đặt → Telegram</b> → bấm <b>Kết nối</b> để mình biết bạn là ai.\n\n"+telegramHelp)
				return
			}
		}
		b.reply(ctx, m, telegramHelp)
	case "/help", "/huongdan":
		b.reply(ctx, m, telegramHelp)
	case "/tong", "/thang":
		link, err := b.findLink(ctx, m.From.ID)
		if errors.Is(err, errTelegramInactive) {
			b.reply(ctx, m, telegramInactiveText)
			return
		}
		if err != nil {
			b.hintLink(ctx, m, true)
			return
		}
		text, err := b.summary(ctx, link)
		if err != nil {
			log.Printf("thuchi: telegram summary: %v", err)
			return
		}
		b.reply(ctx, m, text)
	default:
		if private {
			b.reply(ctx, m, telegramHelp)
		}
	}
}

func (b *TelegramBot) claim(ctx context.Context, m *tgMessage, code string) {
	code = strings.ToUpper(strings.TrimSpace(code))
	_, err := b.svc.Store.ClaimTelegramCode(ctx, code, m.From.ID, m.From.Username, m.From.displayName())
	if errors.Is(err, ErrNotFound) {
		b.reply(ctx, m, "Mã kết nối không đúng hoặc đã hết hạn. Tạo mã mới trong app → Cài đặt → Telegram.")
		return
	}
	if err != nil {
		log.Printf("thuchi: telegram claim: %v", err)
		b.reply(ctx, m, "Chưa kết nối được, thử lại sau nhé.")
		return
	}
	b.mu.Lock()
	delete(b.hinted, m.From.ID)
	b.mu.Unlock()
	b.reply(ctx, m, "✅ Đã kết nối với Thu Chi. Từ giờ chỉ cần nhắn kiểu <code>cafe 15k</code>, mình ghi ngay.")
}

func (b *TelegramBot) hintLink(ctx context.Context, m *tgMessage, force bool) {
	b.mu.Lock()
	last, seen := b.hinted[m.From.ID]
	if !force && seen && time.Since(last) < telegramHintEvery {
		b.mu.Unlock()
		return
	}
	b.hinted[m.From.ID] = time.Now()
	b.mu.Unlock()
	b.reply(ctx, m, fmt.Sprintf("👋 %s ơi, mình chưa biết bạn dùng tài khoản Thu Chi nào. Mở app → <b>Cài đặt → Telegram</b> → bấm <b>Kết nối</b> (chỉ 1 lần).", html.EscapeString(firstNonEmpty(m.From.FirstName, m.From.Username, "Bạn"))))
}

func (ic *ingestCtx) quickAccount(link *TelegramLink) *string {
	if link.DefaultAccountID != nil {
		if a := ic.account(link.DefaultAccountID); a != nil {
			return &a.ID
		}
	}
	for _, a := range ic.accounts {
		if a.Kind == "cash" {
			return &a.ID
		}
	}
	return nil
}

func (s *Service) quickClassify(ctx context.Context, tx Tx, ic *ingestCtx, t *Transaction, explicit string) error {
	catID, rule := ic.classifyQuick(t.Merchant, explicit)
	t.Direction = firstNonEmpty(explicit, DirectionOut)
	if explicit == "" && catID != nil && ic.categories[*catID].Kind == "income" {
		t.Direction = DirectionIn
	}
	t.CategoryID = catID
	t.NeedsReview = catID == nil
	if rule != nil {
		rule.Hits++
		return tx.SaveRule(ctx, rule)
	}
	return nil
}

func (s *Service) quickCreate(ctx context.Context, tx Tx, ic *ingestCtx, link *TelegramLink, it QuickItem) (*Transaction, error) {
	t := &Transaction{
		ID:         NewID(),
		AccountID:  ic.quickAccount(link),
		Amount:     it.Amount,
		OccurredAt: it.OccurredAt,
		RawContent: it.Line,
		Merchant:   it.Desc,
		Kind:       KindNormal,
		Sources:    StringList{SourceManual},
	}
	if err := s.quickClassify(ctx, tx, ic, t, it.Direction); err != nil {
		return nil, err
	}
	return t, s.saveTx(ctx, tx, t)
}

func (b *TelegramBot) handleEntry(ctx context.Context, link *TelegramLink, m *tgMessage, text string, private bool) {
	items := ParseQuick(text, time.Unix(m.Date, 0))
	if len(items) == 0 {
		if private {
			b.reply(ctx, m, "Mình chưa thấy số tiền. Nhắn kiểu <code>cafe 15k</code> hoặc /huongdan.")
		}
		return
	}
	if len(items) > telegramMaxItems {
		items = items[:telegramMaxItems]
	}
	s := b.svc
	var view *entryView
	err := s.Store.RunOwner(ctx, link.OwnerID, func(ctx context.Context, tx Tx) error {
		if _, err := tx.GetTelegramMessage(ctx, m.Chat.ID, m.MessageID); err == nil {
			return errTelegramDup
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		if err := s.ensureSeeded(ctx, tx); err != nil {
			return err
		}
		ic, err := s.loadIngestCtx(ctx, tx)
		if err != nil {
			return err
		}
		rec := &TelegramMessage{ChatID: m.Chat.ID, MessageID: m.MessageID}
		for _, it := range items {
			t, err := s.quickCreate(ctx, tx, ic, link, it)
			if err != nil {
				return err
			}
			rec.TransactionIDs = append(rec.TransactionIDs, t.ID)
		}
		if err := tx.SaveTelegramMessage(ctx, rec); err != nil {
			return err
		}
		view, err = s.loadEntryView(ctx, tx, rec)
		return err
	})
	if errors.Is(err, errTelegramDup) {
		return
	}
	if err != nil {
		log.Printf("thuchi: telegram entry: %v", err)
		b.reply(ctx, m, "Lỗi khi ghi giao dịch, thử lại sau nhé.")
		return
	}
	if len(view.txs) == 1 && view.txs[0].CategoryID == nil {
		view.openPicker(view.txs[0])
	}
	if err := b.sendView(ctx, m, view); err != nil {
		log.Printf("thuchi: telegram send entry: %v", err)
	}
	s.Kick()
}

func (b *TelegramBot) sendView(ctx context.Context, m *tgMessage, view *entryView) error {
	if view.pickFor != "" {
		if err := b.svc.Store.RunOwner(ctx, view.owner, func(ctx context.Context, tx Tx) error {
			return b.svc.fillPicker(ctx, tx, view)
		}); err != nil {
			return err
		}
	}
	text, markup := view.render()
	msg, err := b.send(ctx, m.Chat.ID, m.MessageID, text, markup)
	if err != nil {
		return err
	}
	return b.svc.Store.SetTelegramReply(ctx, m.Chat.ID, m.MessageID, msg.MessageID)
}

func (b *TelegramBot) findRecord(ctx context.Context, tx Tx, chatID int64, replyTo *tgMessage) (*TelegramMessage, bool, error) {
	if replyTo.From != nil && replyTo.From.IsBot {
		rec, err := tx.FindTelegramReply(ctx, chatID, replyTo.MessageID)
		return rec, true, err
	}
	rec, err := tx.GetTelegramMessage(ctx, chatID, replyTo.MessageID)
	return rec, false, err
}

var deleteWords = set("XOA", "XOA DI", "XOA NHE", "HUY", "HUY DI", "BO", "DELETE", "DEL")

func matchCategoryName(cats []*Category, text, prefer string) *Category {
	norm := Normalize(text)
	if len([]rune(norm)) < 2 {
		return nil
	}
	pick := func(match func(n string) bool) *Category {
		var hits []*Category
		for _, c := range cats {
			if match(Normalize(c.Name)) {
				hits = append(hits, c)
			}
		}
		for _, c := range hits {
			if c.Kind == prefer {
				return c
			}
		}
		if len(hits) == 1 {
			return hits[0]
		}
		return nil
	}
	if c := pick(func(n string) bool { return n == norm }); c != nil {
		return c
	}
	if len(norm) < 3 {
		return nil
	}
	return pick(func(n string) bool { return strings.HasPrefix(n, norm) })
}

func (b *TelegramBot) handleReply(ctx context.Context, link *TelegramLink, m *tgMessage, text string) bool {
	text, _, _ = strings.Cut(strings.TrimSpace(text), "\n")
	text = strings.TrimSpace(text)
	s := b.svc
	ts := s.now().UnixMilli()
	var view *entryView
	toBot, handled := false, false
	err := s.Store.RunOwner(ctx, link.OwnerID, func(ctx context.Context, tx Tx) error {
		rec, isBot, err := b.findRecord(ctx, tx, m.Chat.ID, m.ReplyTo)
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		toBot = isBot
		if err != nil {
			return err
		}
		txs, err := loadRecordTxs(ctx, tx, rec)
		if err != nil {
			return err
		}
		var active []*Transaction
		for _, t := range txs {
			if t.DeletedAt == nil {
				active = append(active, t)
			}
		}
		norm := Normalize(text)
		note := ""
		switch {
		case deleteWords[norm]:
			for _, t := range active {
				if err := s.quickDelete(ctx, tx, t, true, ts); err != nil {
					return err
				}
			}
			handled = len(active) > 0
		case len(active) == 1:
			t := active[0]
			if amount, ok := ParseAmount(text); ok {
				if !t.ManualOnly() {
					return invalid("giao dịch ngân hàng không sửa được số tiền")
				}
				t.Amount = amount
				setField(t, "amount", ts)
				if err := s.saveTx(ctx, tx, t); err != nil {
					return err
				}
				handled = true
				break
			}
			cats, err := tx.ListCategories(ctx)
			if err != nil {
				return err
			}
			if c := matchCategoryName(cats, text, kindOf(t.Direction)); c != nil {
				if note, err = s.quickSetCategory(ctx, tx, t, c, ts); err != nil {
					return err
				}
				handled = true
			}
		}
		if !handled {
			return nil
		}
		view, err = s.loadEntryView(ctx, tx, rec)
		if view != nil {
			view.note = note
		}
		return err
	})
	if err != nil {
		if errors.Is(err, ErrInvalid) {
			b.reply(ctx, m, "Không sửa được: "+strings.TrimPrefix(err.Error(), ErrInvalid.Error()+": "))
			return true
		}
		log.Printf("thuchi: telegram reply: %v", err)
		return toBot
	}
	if !handled {
		if toBot {
			b.reply(ctx, m, "Mình chưa hiểu. Trả lời bằng tên danh mục (vd <code>ăn uống</code>), số tiền mới (vd <code>20k</code>) hoặc <code>xoá</code>.")
		}
		return toBot
	}
	b.react(ctx, m.Chat.ID, m.MessageID, "👌")
	if view.reply != nil {
		text, markup := view.render()
		b.edit(ctx, m.Chat.ID, *view.reply, text, markup)
	}
	s.Kick()
	return true
}

func (b *TelegramBot) handleEdit(ctx context.Context, link *TelegramLink, m *tgMessage, text string) {
	s := b.svc
	ts := s.now().UnixMilli()
	items := ParseQuick(text, time.Unix(m.Date, 0))
	if len(items) > telegramMaxItems {
		items = items[:telegramMaxItems]
	}
	var view *entryView
	found := true
	err := s.Store.RunOwner(ctx, link.OwnerID, func(ctx context.Context, tx Tx) error {
		rec, err := tx.GetTelegramMessage(ctx, m.Chat.ID, m.MessageID)
		if errors.Is(err, ErrNotFound) {
			found = false
			return nil
		}
		if err != nil {
			return err
		}
		ic, err := s.loadIngestCtx(ctx, tx)
		if err != nil {
			return err
		}
		ids := StringList{}
		for i, it := range items {
			if i < len(rec.TransactionIDs) {
				t, err := tx.GetTransaction(ctx, rec.TransactionIDs[i])
				if err != nil && !errors.Is(err, ErrNotFound) {
					return err
				}
				if err == nil && t.ManualOnly() {
					t.Amount, t.Merchant, t.RawContent, t.OccurredAt, t.DeletedAt = it.Amount, it.Desc, it.Line, it.OccurredAt, nil
					for _, f := range []string{"amount", "merchant", "raw_content", "occurred_at", "deleted_at"} {
						setField(t, f, ts)
					}
					if _, userSet := t.FieldTS["category_id"]; !userSet {
						if err := s.quickClassify(ctx, tx, ic, t, it.Direction); err != nil {
							return err
						}
					} else if it.Direction != "" {
						t.Direction = it.Direction
					}
					if err := s.saveTx(ctx, tx, t); err != nil {
						return err
					}
					ids = append(ids, t.ID)
					continue
				}
				if err == nil {
					ids = append(ids, t.ID)
					continue
				}
			}
			t, err := s.quickCreate(ctx, tx, ic, link, it)
			if err != nil {
				return err
			}
			ids = append(ids, t.ID)
		}
		for i := len(items); i < len(rec.TransactionIDs); i++ {
			t, err := tx.GetTransaction(ctx, rec.TransactionIDs[i])
			if errors.Is(err, ErrNotFound) {
				continue
			}
			if err != nil {
				return err
			}
			if err := s.quickDelete(ctx, tx, t, true, ts); err != nil && !errors.Is(err, ErrInvalid) {
				return err
			}
			ids = append(ids, t.ID)
		}
		rec.TransactionIDs = ids
		if err := tx.SaveTelegramMessage(ctx, rec); err != nil {
			return err
		}
		view, err = s.loadEntryView(ctx, tx, rec)
		return err
	})
	if err != nil {
		log.Printf("thuchi: telegram edit: %v", err)
		return
	}
	if !found {
		b.handleEntry(ctx, link, m, text, m.Chat.Type == "private")
		return
	}
	text2, markup := view.render()
	if view.reply != nil {
		b.edit(ctx, m.Chat.ID, *view.reply, text2, markup)
	} else if len(view.txs) > 0 {
		_ = b.sendView(ctx, m, view)
	}
	s.Kick()
}

func (s *Service) quickDelete(ctx context.Context, tx Tx, t *Transaction, deleted bool, ts int64) error {
	if t.Kind == KindTransfer && t.PairID != nil {
		return invalid("chuyển tiền sửa trong app")
	}
	if deleted == (t.DeletedAt != nil) {
		return nil
	}
	if deleted {
		now := s.now().UTC()
		t.DeletedAt = &now
	} else {
		t.DeletedAt = nil
	}
	setField(t, "deleted_at", ts)
	return s.saveTx(ctx, tx, t)
}

func (s *Service) quickSetCategory(ctx context.Context, tx Tx, t *Transaction, c *Category, ts int64) (string, error) {
	if t.DeletedAt != nil {
		return "", invalid("giao dịch đã xoá")
	}
	if want := map[string]string{"income": DirectionIn, "expense": DirectionOut}[c.Kind]; want != "" && t.Direction != want {
		if !t.ManualOnly() {
			return "", invalid("giao dịch ngân hàng không đổi được chiều thu/chi")
		}
		t.Direction = want
		setField(t, "direction", ts)
		if err := s.saveTx(ctx, tx, t); err != nil {
			return "", err
		}
	}
	id := c.ID
	if err := s.actSetCategory(ctx, tx, actionArgs{ID: t.ID, CategoryID: &id, Learn: true}, ts); err != nil {
		return "", err
	}
	name := firstNonEmpty(t.Merchant, t.CounterpartyName)
	if name == "" {
		return "", nil
	}
	return fmt.Sprintf("🧠 Đã nhớ: “%s” → %s", html.EscapeString(strings.ToLower(name)), html.EscapeString(c.Name)), nil
}

func compactID(id string) string { return strings.ReplaceAll(id, "-", "") }

func expandID(s string) string {
	if len(s) != 32 {
		return ""
	}
	return s[0:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:32]
}

func catShort(id string) string {
	c := compactID(id)
	return c[len(c)-10:]
}

func (b *TelegramBot) handleCallback(ctx context.Context, cb *tgCallback) {
	if cb.Message == nil {
		b.answer(ctx, cb.ID, "", false)
		return
	}
	parts := strings.Split(cb.Data, ":")
	op, txID := parts[0], ""
	if len(parts) > 1 {
		txID = expandID(parts[1])
	}
	if !ValidID(txID) {
		b.answer(ctx, cb.ID, "", false)
		return
	}
	link, err := b.findLink(ctx, cb.From.ID)
	if errors.Is(err, errTelegramInactive) {
		b.answer(ctx, cb.ID, telegramInactiveText, true)
		return
	}
	if err != nil {
		b.answer(ctx, cb.ID, "Bạn chưa kết nối Thu Chi: mở app → Cài đặt → Telegram.", true)
		return
	}
	s := b.svc
	ts := s.now().UnixMilli()
	var view *entryView
	toast := ""
	err = s.Store.RunOwner(ctx, link.OwnerID, func(ctx context.Context, tx Tx) error {
		rec, err := tx.FindTelegramReply(ctx, cb.Message.Chat.ID, cb.Message.MessageID)
		if err != nil {
			return err
		}
		if !rec.TransactionIDs.Has(txID) {
			return ErrNotFound
		}
		t, err := tx.GetTransaction(ctx, txID)
		if err != nil {
			return err
		}
		pickKind, note := "", ""
		switch op {
		case "p":
			pickKind = kindOf(t.Direction)
		case "P":
			pickKind = map[string]string{"expense": "income", "income": "expense"}[kindOf(t.Direction)]
		case "s":
			if len(parts) < 3 {
				return invalid("danh mục")
			}
			cats, err := tx.ListCategories(ctx)
			if err != nil {
				return err
			}
			var cat *Category
			for _, c := range cats {
				if catShort(c.ID) == parts[2] {
					cat = c
				}
			}
			if cat == nil {
				return invalid("danh mục không còn")
			}
			if note, err = s.quickSetCategory(ctx, tx, t, cat, ts); err != nil {
				return err
			}
			toast = "Đã đổi sang " + cat.Name
		case "d", "u":
			if err := s.quickDelete(ctx, tx, t, op == "d", ts); err != nil {
				return err
			}
			toast = map[string]string{"d": "Đã xoá", "u": "Đã khôi phục"}[op]
		}
		view, err = s.loadEntryView(ctx, tx, rec)
		if err != nil {
			return err
		}
		view.note = note
		if pickKind != "" {
			for _, x := range view.txs {
				if x.ID == txID {
					view.pickFor, view.pickKind = x.ID, pickKind
				}
			}
			return s.fillPicker(ctx, tx, view)
		}
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			b.answer(ctx, cb.ID, "Đây không phải giao dịch của bạn.", true)
		case errors.Is(err, ErrInvalid):
			b.answer(ctx, cb.ID, "Không sửa được: "+strings.TrimPrefix(err.Error(), ErrInvalid.Error()+": "), true)
		default:
			log.Printf("thuchi: telegram callback: %v", err)
			b.answer(ctx, cb.ID, "Lỗi, thử lại sau nhé.", true)
		}
		return
	}
	b.answer(ctx, cb.ID, toast, false)
	text, markup := view.render()
	b.edit(ctx, cb.Message.Chat.ID, cb.Message.MessageID, text, markup)
	if op == "s" || op == "d" || op == "u" {
		s.Kick()
	}
}

type entryView struct {
	owner    string
	reply    *int64
	txs      []*Transaction
	cats     map[string]*Category
	accounts map[string]*Account
	pickFor  string
	pickKind string
	pickCats []*Category
	note     string
}

func loadRecordTxs(ctx context.Context, tx Tx, rec *TelegramMessage) ([]*Transaction, error) {
	var out []*Transaction
	for _, id := range rec.TransactionIDs {
		t, err := tx.GetTransaction(ctx, id)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (s *Service) loadEntryView(ctx context.Context, tx Tx, rec *TelegramMessage) (*entryView, error) {
	txs, err := loadRecordTxs(ctx, tx, rec)
	if err != nil {
		return nil, err
	}
	cats, err := tx.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	accounts, err := tx.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	v := &entryView{owner: tx.Owner(), reply: rec.ReplyMessageID, txs: txs, cats: map[string]*Category{}, accounts: map[string]*Account{}}
	for _, c := range cats {
		v.cats[c.ID] = c
	}
	for _, a := range accounts {
		v.accounts[a.ID] = a
	}
	return v, nil
}

func (v *entryView) openPicker(t *Transaction) {
	v.pickFor, v.pickKind = t.ID, kindOf(t.Direction)
}

func (s *Service) fillPicker(ctx context.Context, tx Tx, v *entryView) error {
	dir := map[string]string{"income": DirectionIn, "expense": DirectionOut}[v.pickKind]
	recent, err := tx.FindTransactions(ctx, TxQuery{Direction: dir, From: s.now().AddDate(0, 0, -90), Limit: 500})
	if err != nil {
		return err
	}
	uses := map[string]int{}
	for _, t := range recent {
		if t.CategoryID != nil {
			uses[*t.CategoryID]++
		}
	}
	var cats []*Category
	for _, c := range v.cats {
		if c.Kind == v.pickKind && c.Key != CatPhiNganHang {
			cats = append(cats, c)
		}
	}
	sort.SliceStable(cats, func(i, j int) bool {
		if uses[cats[i].ID] != uses[cats[j].ID] {
			return uses[cats[i].ID] > uses[cats[j].ID]
		}
		return cats[i].Sort < cats[j].Sort
	})
	if len(cats) > telegramPickLimit {
		cats = cats[:telegramPickLimit]
	}
	v.pickCats = cats
	return nil
}

var categoryEmoji = map[string]string{
	"food": "🍜", "restaurant": "🍽", "shopping": "🛍", "cosmetic": "💄", "beauty": "💅", "car": "🚗", "fuel": "⛽", "medicine": "💊",
	"book": "📚", "cloud": "☁️", "game": "🎮", "bill": "🧾", "home": "🏠", "send": "📤", "bank": "🏦", "more": "📦", "salary": "💰",
	"key": "🔑", "receive": "📥", "coffee": "☕", "tag": "🏷", "wallet": "👛", "card": "💳", "cash": "💵",
}

func FormatVND(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	out := b.String() + "đ"
	if neg {
		return "-" + out
	}
	return out
}

func (v *entryView) catLabel(id *string) string {
	if id == nil {
		return "❓ Chưa phân loại"
	}
	c, ok := v.cats[*id]
	if !ok {
		return "❓ Chưa phân loại"
	}
	return firstNonEmpty(categoryEmoji[c.Icon], "•") + " " + c.Name
}

func signedAmount(t *Transaction) string {
	if t.Direction == DirectionIn {
		return "+" + FormatVND(t.Amount)
	}
	return "−" + FormatVND(t.Amount)
}

func (v *entryView) dayLabel(t *Transaction) string {
	local := t.OccurredAt.In(VNZone)
	today := time.Now().In(VNZone)
	if local.Year() == today.Year() && local.YearDay() == today.YearDay() {
		return ""
	}
	return " · " + local.Format("02/01")
}

func (v *entryView) render() (string, *tgMarkup) {
	var lines []string
	esc := html.EscapeString
	if len(v.txs) == 0 {
		return "🗑 Đã xoá", nil
	}
	if len(v.txs) == 1 {
		t := v.txs[0]
		if t.DeletedAt != nil {
			lines = append(lines, fmt.Sprintf("🗑 <s>%s %s</s> · đã xoá", esc(t.Merchant), signedAmount(t)))
		} else {
			icon := "✅"
			if t.CategoryID == nil {
				icon = "❓"
			}
			lines = append(lines, fmt.Sprintf("%s <b>%s</b> %s", icon, esc(t.Merchant), signedAmount(t)))
			meta := v.catLabel(t.CategoryID)
			if t.AccountID != nil {
				if a, ok := v.accounts[*t.AccountID]; ok {
					meta += " · " + esc(a.Name)
				}
			}
			lines = append(lines, meta+v.dayLabel(t))
		}
	} else {
		var out, in int64
		n := 0
		for _, t := range v.txs {
			if t.DeletedAt != nil {
				continue
			}
			n++
			if t.Direction == DirectionIn {
				in += t.Amount
			} else {
				out += t.Amount
			}
		}
		head := fmt.Sprintf("✅ Đã ghi %d khoản", n)
		if out > 0 {
			head += " · chi " + FormatVND(out)
		}
		if in > 0 {
			head += " · thu " + FormatVND(in)
		}
		lines = append(lines, head)
		for i, t := range v.txs {
			if t.DeletedAt != nil {
				lines = append(lines, fmt.Sprintf("%d. <s>%s %s</s> · đã xoá", i+1, esc(t.Merchant), signedAmount(t)))
				continue
			}
			lines = append(lines, fmt.Sprintf("%d. %s %s → %s%s", i+1, esc(t.Merchant), signedAmount(t), v.catLabel(t.CategoryID), v.dayLabel(t)))
		}
	}
	if v.note != "" {
		lines = append(lines, "", "<i>"+v.note+"</i>")
	}
	markup := &tgMarkup{Keyboard: [][]tgButton{}}
	if v.pickFor != "" {
		var target *Transaction
		for i, t := range v.txs {
			if t.ID == v.pickFor {
				target = t
				if len(v.txs) > 1 {
					lines = append(lines, "", fmt.Sprintf("👇 Chọn danh mục cho <b>%d. %s</b>", i+1, esc(t.Merchant)))
				} else {
					lines = append(lines, "", "👇 Chọn danh mục, mình sẽ nhớ cho lần sau")
				}
			}
		}
		if target != nil {
			tid := compactID(target.ID)
			var row []tgButton
			for _, c := range v.pickCats {
				label := firstNonEmpty(categoryEmoji[c.Icon], "•") + " " + c.Name
				if target.CategoryID != nil && *target.CategoryID == c.ID {
					label = "✓ " + label
				}
				row = append(row, tgButton{Text: label, Data: "s:" + tid + ":" + catShort(c.ID)})
				if len(row) == 3 {
					markup.Keyboard = append(markup.Keyboard, row)
					row = nil
				}
			}
			if len(row) > 0 {
				markup.Keyboard = append(markup.Keyboard, row)
			}
			swap := "⇄ Là khoản thu"
			if v.pickKind == "income" {
				swap = "⇄ Là khoản chi"
			}
			markup.Keyboard = append(markup.Keyboard, []tgButton{{Text: swap, Data: "P:" + tid}, {Text: "« Xong", Data: "b:" + tid}})
			return strings.Join(lines, "\n"), markup
		}
	}
	for i, t := range v.txs {
		tid := compactID(t.ID)
		if len(v.txs) == 1 {
			if t.DeletedAt != nil {
				markup.Keyboard = append(markup.Keyboard, []tgButton{{Text: "↩️ Hoàn tác", Data: "u:" + tid}})
			} else {
				markup.Keyboard = append(markup.Keyboard, []tgButton{{Text: "✏️ Đổi danh mục", Data: "p:" + tid}, {Text: "🗑 Xoá", Data: "d:" + tid}})
			}
			break
		}
		if t.DeletedAt != nil {
			markup.Keyboard = append(markup.Keyboard, []tgButton{{Text: fmt.Sprintf("↩️ %d. Hoàn tác", i+1), Data: "u:" + tid}})
			continue
		}
		label := "Chọn danh mục"
		if t.CategoryID != nil {
			if c, ok := v.cats[*t.CategoryID]; ok {
				label = c.Name
			}
		}
		markup.Keyboard = append(markup.Keyboard, []tgButton{{Text: fmt.Sprintf("✏️ %d. %s", i+1, label), Data: "p:" + tid}, {Text: "🗑", Data: "d:" + tid}})
	}
	return strings.Join(lines, "\n"), markup
}

func (b *TelegramBot) summary(ctx context.Context, link *TelegramLink) (string, error) {
	s := b.svc
	now := s.now().In(VNZone)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, VNZone)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, VNZone)
	var text string
	err := s.Store.RunOwner(ctx, link.OwnerID, func(ctx context.Context, tx Tx) error {
		txs, err := tx.FindTransactions(ctx, TxQuery{From: monthStart.UTC(), Limit: 500})
		if err != nil {
			return err
		}
		v, err := s.loadEntryView(ctx, tx, &TelegramMessage{})
		if err != nil {
			return err
		}
		var spend, income, today int64
		byCat := map[string]int64{}
		for _, t := range txs {
			if t.Excluded {
				continue
			}
			if t.AccountID != nil {
				if a, ok := v.accounts[*t.AccountID]; ok && !a.IncludeInStats {
					continue
				}
			}
			switch {
			case t.Kind == KindNormal && t.Direction == DirectionOut:
				spend += t.Amount
				if !t.OccurredAt.Before(dayStart) {
					today += t.Amount
				}
				if t.CategoryID != nil {
					byCat[*t.CategoryID] += t.Amount
				} else {
					byCat[""] += t.Amount
				}
			case t.Kind == KindRefund:
				spend -= t.Amount
				if t.RefundOfCategoryID != nil {
					byCat[*t.RefundOfCategoryID] -= t.Amount
				}
			case t.Kind == KindNormal && t.Direction == DirectionIn:
				income += t.Amount
			}
		}
		type kv struct {
			id  string
			sum int64
		}
		var top []kv
		for id, sum := range byCat {
			if sum > 0 {
				top = append(top, kv{id, sum})
			}
		}
		sort.Slice(top, func(i, j int) bool { return top[i].sum > top[j].sum })
		lines := []string{
			fmt.Sprintf("📊 <b>Tháng %s</b>", now.Format("01/2006")),
			fmt.Sprintf("Chi: <b>%s</b> · Thu: <b>%s</b>", FormatVND(spend), FormatVND(income)),
			"Hôm nay chi: " + FormatVND(today),
		}
		for i, e := range top {
			if i == 5 {
				break
			}
			var id *string
			if e.id != "" {
				id = &e.id
			}
			lines = append(lines, fmt.Sprintf("%s %s", v.catLabel(id), FormatVND(e.sum)))
		}
		text = strings.Join(lines, "\n")
		return nil
	})
	return text, err
}

func NewTelegramCode() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}
