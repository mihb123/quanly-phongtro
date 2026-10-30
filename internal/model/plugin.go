package model

import (
	"errors"
	"time"

	"github.com/uptrace/bun"
)

var (
	ErrPluginNotFound          = errors.New("không tìm thấy plugin")
	ErrPluginDisabled          = errors.New("plugin đang tắt")
	ErrPluginPaymentRecorded   = errors.New("giao dịch đã được ghi nhận trước đó")
	ErrPluginInvoiceNotPayable = errors.New("hóa đơn không còn ở trạng thái chưa thanh toán")
)

type PluginSetting struct {
	bun.BaseModel `bun:"table:plugin_settings"`
	ManagerID     string    `json:"manager_id"`
	PluginID      string    `json:"plugin_id"`
	Enabled       *bool     `json:"enabled"`
	EncryptedData *string   `json:"-"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type OutstandingInvoice struct {
	ID          string   `json:"id"`
	RoomID      string   `json:"room_id"`
	RoomName    string   `json:"room_name"`
	HouseID     string   `json:"house_id"`
	HouseName   string   `json:"house_name"`
	Period      string   `json:"period"`
	TotalAmount float64  `json:"total_amount"`
	Status      string   `json:"status"`
	TenantNames []string `json:"tenant_names" bun:",array"`
}

type PluginPayment struct {
	Source               string
	InvoiceID            string
	Amount               int
	AccountRef           string
	TransactionReference string
	MatchingMethod       string
	PaymentMethod        string
	RawPayload           string
}
