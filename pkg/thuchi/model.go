package thuchi

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const (
	DirectionIn  = "in"
	DirectionOut = "out"

	KindNormal      = "normal"
	KindTransfer    = "transfer"
	KindRefund      = "refund"
	KindCardPayment = "card_payment"
	KindAdjustment  = "adjustment"
	KindBorrow      = "borrow"
	KindLend        = "lend"
	KindRepay       = "repay"
	KindCollect     = "collect"

	PairAuto      = "auto"
	PairSuggested = "suggested"
	PairConfirmed = "confirmed"
	PairRejected  = "rejected"

	SourceSePay   = "sepay"
	SourceEmail   = "email"
	SourceNotify  = "notify"
	SourceManual  = "manual"
	SourceDerived = "derived"

	RawPending   = "pending"
	RawParsed    = "parsed"
	RawDuplicate = "duplicate"
	RawUnparsed  = "unparsed"
	RawIgnored   = "ignored"
	RawError     = "error"
	RawDismissed = "dismissed"

	ConnectorManual = "manual"
	ConnectorSePay  = "sepay"
	ConnectorEmail  = "email"
	ConnectorNotify = "notify"

	MatchContains   = "contains"
	MatchRegex      = "regex"
	MatchMerchant   = "merchant"
	MatchOwnAccount = "own_account"

	OriginManual  = "manual"
	OriginLearned = "learned"
)

var (
	ErrNotFound  = errors.New("thuchi: not found")
	ErrForbidden = errors.New("thuchi: forbidden")
	ErrInvalid   = errors.New("thuchi: invalid input")
)

type FieldTS map[string]int64

func (m FieldTS) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(map[string]int64(m))
	return string(b), err
}

func (m *FieldTS) Scan(src any) error {
	return scanJSON(src, m)
}

type StringList []string

func (l StringList) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	b, err := json.Marshal([]string(l))
	return string(b), err
}

func (l *StringList) Scan(src any) error {
	return scanJSON(src, l)
}

func (l StringList) Has(v string) bool {
	for _, s := range l {
		if s == v {
			return true
		}
	}
	return false
}

func scanJSON(src any, dst any) error {
	switch v := src.(type) {
	case nil:
		return nil
	case []byte:
		return json.Unmarshal(v, dst)
	case string:
		return json.Unmarshal([]byte(v), dst)
	default:
		return fmt.Errorf("thuchi: cannot scan %T as json", src)
	}
}

type Meta struct {
	OwnerID   string     `json:"-"`
	FieldTS   FieldTS    `json:"-"`
	Version   int        `json:"version"`
	ChangeSeq int64      `json:"change_seq"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

type Account struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Kind           string     `json:"kind"`
	BankCode       string     `json:"bank_code"`
	AccountNoLast4 string     `json:"account_no_last4"`
	AccountNoHash  string     `json:"-"`
	HolderName     string     `json:"holder_name"`
	Connector      string     `json:"connector"`
	Balance        int64      `json:"balance"`
	BalanceAt      *time.Time `json:"balance_at"`
	IncludeInStats bool       `json:"include_in_stats"`
	Sort           int        `json:"sort"`
	Meta
}

type Category struct {
	ID       string  `json:"id"`
	Key      string  `json:"key"`
	Name     string  `json:"name"`
	Icon     string  `json:"icon"`
	Color    string  `json:"color"`
	Kind     string  `json:"kind"`
	ParentID *string `json:"parent_id"`
	Sort     int     `json:"sort"`
	Meta
}

type Transaction struct {
	ID                  string     `json:"id"`
	AccountID           *string    `json:"account_id"`
	Direction           string     `json:"direction"`
	Amount              int64      `json:"amount"`
	OccurredAt          time.Time  `json:"occurred_at"`
	RawContent          string     `json:"raw_content"`
	Merchant            string     `json:"merchant"`
	CounterpartyName    string     `json:"counterparty_name"`
	CounterpartyAccount string     `json:"counterparty_account"`
	BalanceAfter        *int64     `json:"balance_after"`
	CategoryID          *string    `json:"category_id"`
	Note                string     `json:"note"`
	Lat                 *float64   `json:"lat"`
	Lng                 *float64   `json:"lng"`
	PlaceName           string     `json:"place_name"`
	Kind                string     `json:"kind"`
	PairID              *string    `json:"pair_id"`
	PairStatus          string     `json:"pair_status"`
	RefundOfCategoryID  *string    `json:"refund_of_category_id"`
	Excluded            bool       `json:"excluded"`
	Fingerprint         string     `json:"-"`
	Sources             StringList `json:"sources"`
	Reconciled          bool       `json:"reconciled"`
	NeedsReview         bool       `json:"needs_review"`
	Party               string     `json:"party"`
	RecurrenceID        *string    `json:"recurrence_id"`
	Meta
}

func (t *Transaction) ManualOnly() bool {
	for _, s := range t.Sources {
		if s != SourceManual {
			return false
		}
	}
	return true
}

type Rule struct {
	ID           string  `json:"id"`
	Pattern      string  `json:"pattern"`
	Match        string  `json:"match"`
	CategoryID   *string `json:"category_id"`
	MerchantName string  `json:"merchant_name"`
	Origin       string  `json:"origin"`
	Hits         int     `json:"hits"`
	Meta
}

type Budget struct {
	ID          string  `json:"id"`
	CategoryID  *string `json:"category_id"`
	Month       string  `json:"month"`
	LimitAmount int64   `json:"limit_amount"`
	Meta
}

type Recurrence struct {
	ID         string     `json:"id"`
	AccountID  *string    `json:"account_id"`
	Direction  string     `json:"direction"`
	Amount     int64      `json:"amount"`
	CategoryID *string    `json:"category_id"`
	Merchant   string     `json:"merchant"`
	Note       string     `json:"note"`
	Freq       string     `json:"freq"`
	Interval   int        `json:"interval"`
	StartAt    time.Time  `json:"start_at"`
	EndAt      *time.Time `json:"end_at"`
	NextAt     time.Time  `json:"next_at"`
	Active     bool       `json:"active"`
	Meta
}

type RawEvent struct {
	ID            string     `json:"id"`
	OwnerID       string     `json:"-"`
	Source        string     `json:"source"`
	SourceKey     string     `json:"source_key"`
	Payload       string     `json:"payload,omitempty"`
	ReceivedAt    time.Time  `json:"received_at"`
	Status        string     `json:"status"`
	Error         string     `json:"error"`
	TransactionID *string    `json:"transaction_id"`
	Attempts      int        `json:"-"`
	Version       int        `json:"version"`
	ChangeSeq     int64      `json:"change_seq"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at"`
}

type IngestToken struct {
	ID         string            `json:"id"`
	OwnerID    string            `json:"-"`
	Kind       string            `json:"kind"`
	Name       string            `json:"name"`
	SecretEnc  string            `json:"-"`
	Config     map[string]string `json:"config,omitempty"`
	LastUsedAt *time.Time        `json:"last_used_at"`
	CreatedAt  time.Time         `json:"created_at"`
	RevokedAt  *time.Time        `json:"revoked_at"`
}

type TxSource struct {
	TransactionID string
	Source        string
	SourceRef     string
	RawEventID    *string
}

type Incoming struct {
	Source              string
	SourceRef           string
	BankCode            string
	BankName            string
	AccountNumber       string
	Direction           string
	Amount              int64
	OccurredAt          time.Time
	Content             string
	CounterpartyName    string
	CounterpartyAccount string
	BalanceAfter        *int64
	Lat                 *float64
	Lng                 *float64
}

type Changes struct {
	Accounts     []*Account     `json:"accounts"`
	Categories   []*Category    `json:"categories"`
	Transactions []*Transaction `json:"transactions"`
	Rules        []*Rule        `json:"rules"`
	Budgets      []*Budget      `json:"budgets"`
	Recurrences  []*Recurrence  `json:"recurrences"`
	RawEvents    []*RawEvent    `json:"raw_events"`
}

type TelegramLink struct {
	ID               string     `json:"id"`
	OwnerID          string     `json:"-"`
	TelegramUserID   int64      `json:"telegram_user_id"`
	TelegramUsername string     `json:"telegram_username"`
	TelegramName     string     `json:"telegram_name"`
	DefaultAccountID *string    `json:"default_account_id"`
	CreatedAt        time.Time  `json:"created_at"`
	LinkedAt         *time.Time `json:"linked_at"`
}

type TelegramMessage struct {
	ChatID         int64
	MessageID      int64
	OwnerID        string
	ReplyMessageID *int64
	TransactionIDs StringList
}
