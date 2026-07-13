package zalo

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type qrRoundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip lets QR download tests stub the default HTTP transport.
func (f qrRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// newDeliveryDeps builds a configured delivery dependency set for direct delivery tests.
func newDeliveryDeps(t *testing.T, ctx context.Context, ctrl *gomock.Controller, zaloClient *flowZaloClient, room *model.Room, tenants []model.FullInfoTenant) zaloInvoiceDeliveryDeps {
	t.Helper()
	t.Cleanup(func() { _ = os.RemoveAll("uploads/zalo-invoices") })

	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)
	managerZaloID := "manager-zalo"

	userRepository := mock_model.NewMockUserRepository(ctrl)
	roomRepository := mock_model.NewMockRoomRepository(ctrl)
	tenantRepository := mock_model.NewMockTenantRepository(ctrl)
	invoiceRepository := mock_model.NewMockInvoiceRepository(ctrl)

	userRepository.EXPECT().GetByUserID(ctx, "manager-1").Return(&model.User{ZaloBotToken: &token, ZaloUserID: &managerZaloID}, nil)
	invoice := &model.InvoiceWithRoom{
		Invoice: model.Invoice{
			ID:          "invoice-1",
			RoomID:      "room-1",
			Period:      "2026-05",
			Status:      model.InvoiceStatusPaid,
			TotalAmount: 1500000,
		},
		RoomName: "P101",
		HouseID:  "house-1",
	}
	invoiceRepository.EXPECT().GetInvoiceByID(ctx, "manager-1", "invoice-1").Return(invoice, nil)
	roomRepository.EXPECT().GetRoomByID(ctx, "room-1", "house-1").Return(room, nil)
	tenantRepository.EXPECT().ListTenantByRoomID(ctx, "manager-1", "room-1").Return(tenants, nil)

	return zaloInvoiceDeliveryDeps{
		client:        zaloClient,
		userRepo:      userRepository,
		roomRepo:      roomRepository,
		tenantRepo:    tenantRepository,
		invoiceRepo:   invoiceRepository,
		imageService:  &flowImageService{image: []byte("png")},
		encryptionKey: encryptionKey,
		publicBaseURL: "https://example.com",
	}
}

// TestDeliverInvoiceToZaloRequiresConfiguredDeps verifies missing dependencies are rejected early.
func TestDeliverInvoiceToZaloRequiresConfiguredDeps(t *testing.T) {
	err := deliverInvoiceToZalo(context.Background(), zaloInvoiceDeliveryDeps{}, "manager-1", "invoice-1")
	require.Error(t, err)
}

// TestDeliverInvoiceToZaloSendsPrivateRecipients covers the non-group recipient loop.
func TestDeliverInvoiceToZaloSendsPrivateRecipients(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	zaloClient := &flowZaloClient{}
	deps := newDeliveryDeps(t, ctx, ctrl, zaloClient, &model.Room{ID: "room-1", HouseID: "house-1"}, []model.FullInfoTenant{
		{FullName: "Tenant A", ZaloUserID: "tenant-zalo"},
	})

	err := deliverInvoiceToZalo(ctx, deps, "manager-1", "invoice-1")

	require.NoError(t, err)
	assert.Len(t, zaloClient.photos, 2)
}

// TestDeliverInvoiceToZaloReturnsRecipientErrors covers aggregated private send failures.
func TestDeliverInvoiceToZaloReturnsRecipientErrors(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	zaloClient := &flowZaloClient{sendPhotoErr: errors.New("send failed")}
	deps := newDeliveryDeps(t, ctx, ctrl, zaloClient, &model.Room{ID: "room-1", HouseID: "house-1"}, []model.FullInfoTenant{
		{FullName: "Tenant A", ZaloUserID: "tenant-zalo"},
	})

	err := deliverInvoiceToZalo(ctx, deps, "manager-1", "invoice-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "some messages failed")
}

// TestDeliverInvoiceToZaloGroupAuthErrorMarksInactive covers invalid token handling for group sends.
func TestDeliverInvoiceToZaloGroupAuthErrorMarksInactive(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	groupChatID := "group-1"
	markedInactive := false
	zaloClient := &flowZaloClient{sendPhotoErr: errors.New("invalid access token -216")}
	deps := newDeliveryDeps(t, ctx, ctrl, zaloClient, &model.Room{ID: "room-1", HouseID: "house-1", GroupChatID: &groupChatID}, nil)
	deps.markTokenInactive = func(context.Context, string) {
		markedInactive = true
	}

	err := deliverInvoiceToZalo(ctx, deps, "manager-1", "invoice-1")

	require.Error(t, err)
	assert.True(t, markedInactive)
	assert.Contains(t, err.Error(), "invalid or expired")
}

// TestDeliverInvoiceToZaloReturnsNoRecipientError covers unlinked rooms without any linked users.
func TestDeliverInvoiceToZaloReturnsNoRecipientError(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	t.Cleanup(func() { _ = os.RemoveAll("uploads/zalo-invoices") })

	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)
	userRepository := mock_model.NewMockUserRepository(ctrl)
	roomRepository := mock_model.NewMockRoomRepository(ctrl)
	tenantRepository := mock_model.NewMockTenantRepository(ctrl)
	invoiceRepository := mock_model.NewMockInvoiceRepository(ctrl)

	userRepository.EXPECT().GetByUserID(ctx, "manager-1").Return(&model.User{ZaloBotToken: &token}, nil)
	invoice := &model.InvoiceWithRoom{Invoice: model.Invoice{ID: "invoice-1", RoomID: "room-1"}, HouseID: "house-1"}
	invoiceRepository.EXPECT().GetInvoiceByID(ctx, "manager-1", "invoice-1").Return(invoice, nil)
	roomRepository.EXPECT().GetRoomByID(ctx, "room-1", "house-1").Return(&model.Room{ID: "room-1", HouseID: "house-1"}, nil)
	tenantRepository.EXPECT().ListTenantByRoomID(ctx, "manager-1", "room-1").Return(nil, nil)

	err := deliverInvoiceToZalo(ctx, zaloInvoiceDeliveryDeps{
		client:        &flowZaloClient{},
		userRepo:      userRepository,
		roomRepo:      roomRepository,
		tenantRepo:    tenantRepository,
		invoiceRepo:   invoiceRepository,
		imageService:  &flowImageService{image: []byte("png")},
		encryptionKey: encryptionKey,
		publicBaseURL: "https://example.com",
	}, "manager-1", "invoice-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "linked zalo")
}

// TestDownloadQRCodeImage covers QuickChart download success and non-200 failures without network.
func TestDownloadQRCodeImage(t *testing.T) {
	previousTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previousTransport })

	http.DefaultTransport = qrRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "quickchart.io" {
			t.Fatalf("unexpected QR host: %s", req.URL.Host)
		}
		if req.URL.Query().Get("text") != "hello world" {
			t.Fatalf("unexpected QR text: %s", req.URL.Query().Get("text"))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("qr-image")),
			Request:    req,
		}, nil
	})

	got, err := downloadQRCodeImage("hello world")
	require.NoError(t, err)
	assert.Equal(t, []byte("qr-image"), got)

	http.DefaultTransport = qrRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Body:       io.NopCloser(strings.NewReader("bad gateway")),
			Request:    req,
		}, nil
	})
	_, err = downloadQRCodeImage("hello world")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status")
}

// TestDeliverInvoiceMethodDelegatesDelivery covers the method wrapper around shared delivery.
func TestDeliverInvoiceMethodDelegatesDelivery(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	groupChatID := "group-1"
	zaloClient := &flowZaloClient{}
	deps := newDeliveryDeps(t, ctx, ctrl, zaloClient, &model.Room{ID: "room-1", HouseID: "house-1", GroupChatID: &groupChatID}, nil)
	service := &zaloInvoiceCommandServiceImpl{
		userRepo:      deps.userRepo,
		roomRepo:      deps.roomRepo,
		tenantRepo:    deps.tenantRepo,
		invoiceRepo:   deps.invoiceRepo,
		zaloClient:    deps.client,
		imageService:  deps.imageService,
		encryptionKey: deps.encryptionKey,
		publicBaseURL: deps.publicBaseURL,
	}

	err := service.deliverInvoice(ctx, "manager-1", "invoice-1")

	require.NoError(t, err)
	assert.Len(t, zaloClient.photos, 1)
}

// TestDeliverInvoiceMethodMarksInvalidTokenInactive covers the wrapper's inactive-token callback.
func TestDeliverInvoiceMethodMarksInvalidTokenInactive(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	groupChatID := "group-1"
	zaloClient := &flowZaloClient{sendPhotoErr: errors.New("invalid access token -216")}
	deps := newDeliveryDeps(t, ctx, ctrl, zaloClient, &model.Room{ID: "room-1", HouseID: "house-1", GroupChatID: &groupChatID}, nil)
	userRepository := deps.userRepo.(*mock_model.MockUserRepository)
	userRepository.EXPECT().UpdateUser(ctx, "manager-1", gomock.Any()).Return(&model.User{}, nil)
	service := &zaloInvoiceCommandServiceImpl{
		userRepo:      deps.userRepo,
		roomRepo:      deps.roomRepo,
		tenantRepo:    deps.tenantRepo,
		invoiceRepo:   deps.invoiceRepo,
		zaloClient:    deps.client,
		imageService:  deps.imageService,
		encryptionKey: deps.encryptionKey,
		publicBaseURL: deps.publicBaseURL,
	}

	err := service.deliverInvoice(ctx, "manager-1", "invoice-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or expired")
}

// TestDeliverInvoiceToZaloSendsPaymentQR covers optional payment link and QR delivery.
func TestDeliverInvoiceToZaloSendsPaymentQR(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	t.Cleanup(func() { _ = os.RemoveAll("uploads/zalo-invoices") })

	previousTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previousTransport })
	http.DefaultTransport = qrRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("qr-image")),
			Request:    req,
		}, nil
	})

	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)
	groupChatID := "group-1"
	userRepository := mock_model.NewMockUserRepository(ctrl)
	roomRepository := mock_model.NewMockRoomRepository(ctrl)
	tenantRepository := mock_model.NewMockTenantRepository(ctrl)
	invoiceRepository := mock_model.NewMockInvoiceRepository(ctrl)
	zaloClient := &flowZaloClient{}
	invoice := &model.InvoiceWithRoom{
		Invoice: model.Invoice{
			ID:          "invoice-1",
			RoomID:      "room-1",
			Period:      "2026-05",
			Status:      model.InvoiceStatusUnpaid,
			TotalAmount: 1500000,
		},
		RoomName: "P101",
		HouseID:  "house-1",
	}

	userRepository.EXPECT().GetByUserID(ctx, "manager-1").Return(&model.User{ZaloBotToken: &token}, nil)
	invoiceRepository.EXPECT().GetInvoiceByID(ctx, "manager-1", "invoice-1").Return(invoice, nil)
	roomRepository.EXPECT().GetRoomByID(ctx, "room-1", "house-1").Return(&model.Room{ID: "room-1", HouseID: "house-1", GroupChatID: &groupChatID}, nil)
	tenantRepository.EXPECT().ListTenantByRoomID(ctx, "manager-1", "room-1").Return([]model.FullInfoTenant{{FullName: "Tenant A"}}, nil)

	err := deliverInvoiceToZalo(ctx, zaloInvoiceDeliveryDeps{
		client:       zaloClient,
		userRepo:     userRepository,
		roomRepo:     roomRepository,
		tenantRepo:   tenantRepository,
		invoiceRepo:  invoiceRepository,
		imageService: &flowImageService{image: []byte("png")},
		paymentService: &fakePaymentService{link: &model.InvoicePaymentLink{
			CheckoutURL: "https://pay.example/checkout",
			QRCode:      "qr payload",
		}},
		encryptionKey: encryptionKey,
		publicBaseURL: "https://example.com",
	}, "manager-1", "invoice-1")

	require.NoError(t, err)
	assert.Len(t, zaloClient.photos, 2)
}
