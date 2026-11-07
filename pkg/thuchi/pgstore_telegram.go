package thuchi

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

const telegramLinkCols = "id, owner_id, COALESCE(telegram_user_id, 0), telegram_username, telegram_name, default_account_id, created_at, linked_at"

func scanTelegramLink(row interface{ Scan(...any) error }) (*TelegramLink, error) {
	l := &TelegramLink{}
	err := row.Scan(&l.ID, &l.OwnerID, &l.TelegramUserID, &l.TelegramUsername, &l.TelegramName, &l.DefaultAccountID, &l.CreatedAt, &l.LinkedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return l, err
}

func (s *PGStore) CreateTelegramCode(ctx context.Context, ownerID, code string, expiresAt time.Time) error {
	if !ValidID(ownerID) {
		return ErrInvalid
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM thuchi_telegram_links WHERE owner_id = $1 AND telegram_user_id IS NULL`, ownerID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO thuchi_telegram_links (owner_id, code, code_expires_at) VALUES ($1, $2, $3)`, ownerID, code, expiresAt)
	return err
}

func (s *PGStore) ClaimTelegramCode(ctx context.Context, code string, userID int64, username, name string) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()
	var id, owner string
	err = tx.QueryRowContext(ctx, `SELECT id, owner_id FROM thuchi_telegram_links WHERE code = $1 AND telegram_user_id IS NULL AND code_expires_at > NOW() FOR UPDATE`, code).Scan(&id, &owner)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM thuchi_telegram_links WHERE telegram_user_id = $1`, userID); err != nil {
		return "", err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE thuchi_telegram_links SET telegram_user_id = $2, telegram_username = $3, telegram_name = $4, code = NULL, code_expires_at = NULL, linked_at = NOW() WHERE id = $1`, id, userID, username, name); err != nil {
		return "", err
	}
	return owner, tx.Commit()
}

func (s *PGStore) FindTelegramLink(ctx context.Context, userID int64) (*TelegramLink, error) {
	return scanTelegramLink(s.db.QueryRowContext(ctx, `SELECT `+telegramLinkCols+` FROM thuchi_telegram_links WHERE telegram_user_id = $1`, userID))
}

func (s *PGStore) ListTelegramLinks(ctx context.Context, ownerID string) ([]*TelegramLink, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+telegramLinkCols+` FROM thuchi_telegram_links WHERE owner_id = $1 AND telegram_user_id IS NOT NULL ORDER BY linked_at`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*TelegramLink{}
	for rows.Next() {
		l, err := scanTelegramLink(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *PGStore) DeleteTelegramLink(ctx context.Context, ownerID, id string) error {
	if !ValidID(id) {
		return ErrNotFound
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM thuchi_telegram_links WHERE id = $1 AND owner_id = $2`, id, ownerID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PGStore) SetTelegramLinkAccount(ctx context.Context, ownerID, id string, accountID *string) error {
	if !ValidID(id) {
		return ErrNotFound
	}
	res, err := s.db.ExecContext(ctx, `UPDATE thuchi_telegram_links SET default_account_id = $3 WHERE id = $1 AND owner_id = $2 AND telegram_user_id IS NOT NULL`, id, ownerID, accountID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PGStore) SetTelegramReply(ctx context.Context, chatID, messageID, replyID int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE thuchi_telegram_messages SET reply_message_id = $3 WHERE chat_id = $1 AND message_id = $2`, chatID, messageID, replyID)
	return err
}

const telegramMessageCols = "chat_id, message_id, owner_id, reply_message_id, transaction_ids"

func (t *pgTx) scanTelegramMessage(row *sql.Row) (*TelegramMessage, error) {
	m := &TelegramMessage{}
	err := row.Scan(&m.ChatID, &m.MessageID, &m.OwnerID, &m.ReplyMessageID, &m.TransactionIDs)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return m, err
}

func (t *pgTx) GetTelegramMessage(ctx context.Context, chatID, messageID int64) (*TelegramMessage, error) {
	return t.scanTelegramMessage(t.tx.QueryRowContext(ctx, `SELECT `+telegramMessageCols+` FROM thuchi_telegram_messages WHERE chat_id = $1 AND message_id = $2 AND owner_id = $3`, chatID, messageID, t.owner))
}

func (t *pgTx) FindTelegramReply(ctx context.Context, chatID, replyID int64) (*TelegramMessage, error) {
	return t.scanTelegramMessage(t.tx.QueryRowContext(ctx, `SELECT `+telegramMessageCols+` FROM thuchi_telegram_messages WHERE chat_id = $1 AND reply_message_id = $2 AND owner_id = $3`, chatID, replyID, t.owner))
}

func (t *pgTx) SaveTelegramMessage(ctx context.Context, m *TelegramMessage) error {
	if m.TransactionIDs == nil {
		m.TransactionIDs = StringList{}
	}
	m.OwnerID = t.owner
	_, err := t.tx.ExecContext(ctx, `INSERT INTO thuchi_telegram_messages (chat_id, message_id, owner_id, reply_message_id, transaction_ids) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (chat_id, message_id) DO UPDATE SET transaction_ids = EXCLUDED.transaction_ids, reply_message_id = COALESCE(EXCLUDED.reply_message_id, thuchi_telegram_messages.reply_message_id) WHERE thuchi_telegram_messages.owner_id = EXCLUDED.owner_id`,
		m.ChatID, m.MessageID, t.owner, m.ReplyMessageID, m.TransactionIDs)
	return err
}
