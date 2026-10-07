package zalo

import (
	"context"

	"github.com/mihb123/quanly-phongtro/internal/model"
	invoicesvc "github.com/mihb123/quanly-phongtro/internal/service/invoice"
	tenantsvc "github.com/mihb123/quanly-phongtro/internal/service/tenant"
)

type invoiceCommands interface {
	CreateInvoice(ctx context.Context, managerID string, input invoicesvc.CreateInvoiceInput) (*model.InvoiceWithRoom, error)
}

type commandInvoices interface {
	GetInvoiceByID(ctx context.Context, managerID, id string) (*model.InvoiceWithRoom, error)
	GetInvoiceByRoomAndPeriod(ctx context.Context, roomID, period string) (*model.Invoice, error)
	GetPreviousInvoice(ctx context.Context, roomID, period string) (*model.Invoice, error)
}

type commandRooms interface {
	GetRoomByID(ctx context.Context, id, houseID string) (*model.Room, error)
	ListAllRoomsByHouseID(ctx context.Context, houseID string) ([]model.Room, error)
	UpdateRoom(ctx context.Context, id, houseID string, params model.UpdateRoomParams) (*model.Room, error)
	GetRoomByIDOnly(ctx context.Context, id string) (*model.Room, error)
	GetRoomByGroupChatID(ctx context.Context, groupChatID string) (*model.Room, error)
}

type commandHouses interface {
	GetByID(ctx context.Context, id, managerID string) (*model.House, error)
	GetHouseByCode(ctx context.Context, managerID, houseCode string) (*model.House, error)
	ListHouseByManagerID(ctx context.Context, managerID string, limit, offset int, search string) ([]model.House, error)
}

type commandTenants interface {
	ListTenantByRoomID(ctx context.Context, managerID, roomID string) ([]model.FullInfoTenant, error)
	GetFirstTenantByUserID(ctx context.Context, managerID, userID string) (*model.FullInfoTenant, error)
}

type tenantCommands interface {
	UpdateTenantInfo(ctx context.Context, managerID, tenantID string, in tenantsvc.UpdateTenantInput) (*model.FullInfoTenant, error)
}

type commandUsers interface {
	GetByUserID(context.Context, string) (*model.User, error)
	UpdateUser(ctx context.Context, userID string, input model.UpdateUserInput) (*model.User, error)
	GetByZaloUserID(ctx context.Context, zaloUserID string) (*model.User, error)
}

type pendingCommands interface {
	Create(ctx context.Context, pending *model.PendingInvoiceUpdate) error
	GetByChatID(ctx context.Context, managerID, chatID string) (*model.PendingInvoiceUpdate, error)
	DeleteByChatID(ctx context.Context, managerID, chatID string) error
	DeleteByID(ctx context.Context, id string) error
}

type commandPayments interface {
	CreatePreferredPaymentLinkForInvoice(ctx context.Context, managerID string, invoice *model.InvoiceWithRoom, tenantName string) (*model.InvoicePaymentLink, error)
}

type deliveryUserReader interface {
	GetByUserID(context.Context, string) (*model.User, error)
}
type deliveryRoomReader interface {
	GetRoomByID(context.Context, string, string) (*model.Room, error)
}
type deliveryTenantReader interface {
	ListTenantByRoomID(context.Context, string, string) ([]model.FullInfoTenant, error)
}
type deliveryInvoiceReader interface {
	GetInvoiceByID(context.Context, string, string) (*model.InvoiceWithRoom, error)
}
