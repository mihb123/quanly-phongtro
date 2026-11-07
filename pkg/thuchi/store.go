package thuchi

import (
	"context"
	"time"
)

type TxQuery struct {
	AccountID  string
	NotAccount string
	Direction  string
	AmountMin  int64
	AmountMax  int64
	From       time.Time
	To         time.Time
	ExcludeID  string
	Merchant   string
	NoCategory bool
	Limit      int
}

type Store interface {
	RunOwner(ctx context.Context, ownerID string, fn func(ctx context.Context, tx Tx) error) error
	PendingRawEvents(ctx context.Context, limit int) ([]*RawEvent, error)
	FindIngestToken(ctx context.Context, id string) (*IngestToken, error)
	TouchIngestToken(ctx context.Context, id string) error
	ListIngestTokens(ctx context.Context, ownerID string) ([]*IngestToken, error)
	CreateIngestToken(ctx context.Context, token *IngestToken) error
	RevokeIngestToken(ctx context.Context, ownerID, id string) error
	OwnerExists(ctx context.Context, ownerID string) (bool, error)
	PurgeTombstones(ctx context.Context, olderThan time.Duration) error
	DueRecurrenceOwners(ctx context.Context, now time.Time, limit int) ([]string, error)

	CreateTelegramCode(ctx context.Context, ownerID, code string, expiresAt time.Time) error
	ClaimTelegramCode(ctx context.Context, code string, userID int64, username, name string) (string, error)
	FindTelegramLink(ctx context.Context, userID int64) (*TelegramLink, error)
	ListTelegramLinks(ctx context.Context, ownerID string) ([]*TelegramLink, error)
	DeleteTelegramLink(ctx context.Context, ownerID, id string) error
	SetTelegramLinkAccount(ctx context.Context, ownerID, id string, accountID *string) error
	SetTelegramReply(ctx context.Context, chatID, messageID, replyID int64) error
}

type Tx interface {
	Owner() string
	NextSeq(ctx context.Context) (int64, error)
	CurrentSeq(ctx context.Context) (int64, error)
	PurgedSeq(ctx context.Context) (int64, error)
	Seeded(ctx context.Context) (bool, error)
	MarkSeeded(ctx context.Context) error
	Savepoint(ctx context.Context, fn func() error) error

	GetAccount(ctx context.Context, id string) (*Account, error)
	ListAccounts(ctx context.Context) ([]*Account, error)
	SaveAccount(ctx context.Context, a *Account) error

	GetCategory(ctx context.Context, id string) (*Category, error)
	ListCategories(ctx context.Context) ([]*Category, error)
	SaveCategory(ctx context.Context, c *Category) error

	GetTransaction(ctx context.Context, id string) (*Transaction, error)
	FindTransactions(ctx context.Context, q TxQuery) ([]*Transaction, error)
	SaveTransaction(ctx context.Context, t *Transaction) error

	FindSource(ctx context.Context, source, sourceRef string) (string, error)
	AddSource(ctx context.Context, s TxSource) error
	ReplaceSource(ctx context.Context, s TxSource) error

	GetRule(ctx context.Context, id string) (*Rule, error)
	ListRules(ctx context.Context) ([]*Rule, error)
	SaveRule(ctx context.Context, r *Rule) error

	GetBudget(ctx context.Context, id string) (*Budget, error)
	SaveBudget(ctx context.Context, b *Budget) error

	GetRecurrence(ctx context.Context, id string) (*Recurrence, error)
	DueRecurrences(ctx context.Context, now time.Time, limit int) ([]*Recurrence, error)
	SaveRecurrence(ctx context.Context, r *Recurrence) error

	GetRawEvent(ctx context.Context, id string) (*RawEvent, error)
	SaveRawEvent(ctx context.Context, e *RawEvent) error

	Changes(ctx context.Context, since int64, limit int) (*Changes, int64, bool, error)

	GetTelegramMessage(ctx context.Context, chatID, messageID int64) (*TelegramMessage, error)
	FindTelegramReply(ctx context.Context, chatID, replyID int64) (*TelegramMessage, error)
	SaveTelegramMessage(ctx context.Context, m *TelegramMessage) error
}
