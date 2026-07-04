package tenant_test

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"os"
	"testing"
	"time"

	sharedsvc "github.com/mihb123/quanly-phongtro/internal/service/shared"

	tenantsvc "github.com/mihb123/quanly-phongtro/internal/service/tenant"

	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"go.uber.org/mock/gomock"
)

func createMultipartFileHeader(filename string, content []byte) *multipart.FileHeader {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", filename)
	part.Write(content)
	writer.Close()

	reader := multipart.NewReader(body, writer.Boundary())
	form, _ := reader.ReadForm(1024)
	return form.File["file"][0]
}

func setupTenantTestDir() {
	os.MkdirAll("uploads/tenants", 0755)
}

func teardownTenantTestDir() {
	os.RemoveAll("uploads")
}

func TestTenantService_RegisterTenant(t *testing.T) {
	setupTenantTestDir()
	defer teardownTenantTestDir()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)
	tenantRepo := mock_model.NewMockTenantRepository(ctrl)
	roomRepo := mock_model.NewMockRoomRepository(ctrl)
	hasher := &mockPasswordHasher{}

	svc := tenantsvc.NewTenantServiceImpl(userRepo, tenantRepo, roomRepo, hasher)
	ctx := context.Background()
	expectTenantRoomOwnership := func() {
		roomRepo.EXPECT().GetRoomByIDForManager(ctx, gomock.Any(), "r1").Return(&model.Room{ID: "r1"}, nil)
	}

	tests := []struct {
		name    string
		input   tenantsvc.RegisterTenantInput
		setup   func()
		wantErr bool
	}{
		{
			name: "Success with all fields",
			input: tenantsvc.RegisterTenantInput{
				ManagerID:     "m1",
				Email:         "test@example.com",
				Password:      "pass",
				FullName:      "Test Tenant",
				Phone:         "0123456789",
				RoomID:        "r1",
				IdentityCard:  "ID123",
				StartDate:     time.Now(),
				CCCDFiles:     []*multipart.FileHeader{createMultipartFileHeader("cccd.png", []byte("img"))},
				ContractFiles: []*multipart.FileHeader{createMultipartFileHeader("contract.pdf", []byte("pdf"))},
			},
			setup: func() {
				userRepo.EXPECT().GetByPhone(ctx, "0123456789").Return(nil, errors.New("not found"))
				expectTenantRoomOwnership()
				roomRepo.EXPECT().GetMaxTenants(ctx, "r1").Return(int64(4), nil)
				tenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "r1").Return(int64(1), nil)
				tenantRepo.EXPECT().CreateTenantWithAccount(ctx, gomock.Any(), gomock.Any()).Return(nil)
				roomRepo.EXPECT().UpdateRoomStatus(ctx, "r1", "OCCUPIED").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Success without email/password",
			input: tenantsvc.RegisterTenantInput{
				ManagerID: "m1",
				FullName:  "No Email",
				Phone:     "0999999999",
				RoomID:    "r1",
			},
			setup: func() {
				userRepo.EXPECT().GetByPhone(ctx, "0999999999").Return(nil, errors.New("not found"))
				expectTenantRoomOwnership()
				roomRepo.EXPECT().GetMaxTenants(ctx, "r1").Return(int64(4), nil)
				tenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "r1").Return(int64(1), nil)
				tenantRepo.EXPECT().CreateTenantWithAccount(ctx, gomock.Any(), gomock.Any()).Return(nil)
				roomRepo.EXPECT().UpdateRoomStatus(ctx, "r1", "OCCUPIED").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Invalid email",
			input: tenantsvc.RegisterTenantInput{
				Email: "bad-email",
			},
			setup:   func() {},
			wantErr: true,
		},
		{
			name: "Room full",
			input: tenantsvc.RegisterTenantInput{
				Email:  "test2@example.com",
				RoomID: "r1",
			},
			setup: func() {
				expectTenantRoomOwnership()
				roomRepo.EXPECT().GetMaxTenants(ctx, "r1").Return(int64(2), nil)
				tenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "r1").Return(int64(2), nil)
			},
			wantErr: true,
		},
		{
			name: "Create user error",
			input: tenantsvc.RegisterTenantInput{
				Email:  "test3@example.com",
				RoomID: "r1",
			},
			setup: func() {
				expectTenantRoomOwnership()
				roomRepo.EXPECT().GetMaxTenants(ctx, "r1").Return(int64(4), nil)
				tenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "r1").Return(int64(1), nil)
				tenantRepo.EXPECT().CreateTenantWithAccount(ctx, gomock.Any(), gomock.Any()).Return(errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name: "Unsupported file extension",
			input: tenantsvc.RegisterTenantInput{
				Email:     "test4@example.com",
				RoomID:    "r1",
				CCCDFiles: []*multipart.FileHeader{createMultipartFileHeader("cccd.txt", []byte("txt"))},
			},
			setup: func() {
				expectTenantRoomOwnership()
				roomRepo.EXPECT().GetMaxTenants(ctx, "r1").Return(int64(4), nil)
				tenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "r1").Return(int64(1), nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			res, err := svc.RegisterTenant(ctx, tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if res == nil {
					t.Errorf("expected result, got nil")
				}
			}
		})
	}

	t.Run("Auto-generate email from FullName (no phone, no email)", func(t *testing.T) {
		expectTenantRoomOwnership()
		roomRepo.EXPECT().GetMaxTenants(ctx, "r1").Return(int64(4), nil)
		tenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "r1").Return(int64(1), nil)
		tenantRepo.EXPECT().CreateTenantWithAccount(ctx, gomock.Any(), gomock.Any()).Return(nil)
		roomRepo.EXPECT().UpdateRoomStatus(ctx, "r1", "OCCUPIED").Return(nil)
		_, err := svc.RegisterTenant(ctx, tenantsvc.RegisterTenantInput{FullName: "John Doe", RoomID: "r1"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Auto-generate email with empty FullName (guest)", func(t *testing.T) {
		expectTenantRoomOwnership()
		roomRepo.EXPECT().GetMaxTenants(ctx, "r1").Return(int64(4), nil)
		tenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "r1").Return(int64(1), nil)
		tenantRepo.EXPECT().CreateTenantWithAccount(ctx, gomock.Any(), gomock.Any()).Return(nil)
		roomRepo.EXPECT().UpdateRoomStatus(ctx, "r1", "OCCUPIED").Return(nil)
		_, err := svc.RegisterTenant(ctx, tenantsvc.RegisterTenantInput{FullName: "", RoomID: "r1"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Hash password fails", func(t *testing.T) {
		hasherErr := &mockPasswordHasher{hashErr: errors.New("hash err")}
		svcWithErr := tenantsvc.NewTenantServiceImpl(userRepo, tenantRepo, roomRepo, hasherErr)
		expectTenantRoomOwnership()
		roomRepo.EXPECT().GetMaxTenants(ctx, "r1").Return(int64(4), nil)
		tenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "r1").Return(int64(1), nil)
		_, err := svcWithErr.RegisterTenant(ctx, tenantsvc.RegisterTenantInput{Email: "test@example.com", Password: "pass", RoomID: "r1"})
		if err == nil || err.Error() != "hash err" {
			t.Errorf("expected hash err, got %v", err)
		}
	})

	t.Run("UpdateRoomStatus fails", func(t *testing.T) {
		expectTenantRoomOwnership()
		roomRepo.EXPECT().GetMaxTenants(ctx, "r1").Return(int64(4), nil)
		tenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "r1").Return(int64(1), nil)
		tenantRepo.EXPECT().CreateTenantWithAccount(ctx, gomock.Any(), gomock.Any()).Return(nil)
		roomRepo.EXPECT().UpdateRoomStatus(ctx, "r1", "OCCUPIED").Return(errors.New("db error"))
		_, err := svc.RegisterTenant(ctx, tenantsvc.RegisterTenantInput{Email: "test@example.com", Password: "pass", RoomID: "r1"})
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})

	t.Run("Unsupported contract file extension", func(t *testing.T) {
		expectTenantRoomOwnership()
		roomRepo.EXPECT().GetMaxTenants(ctx, "r1").Return(int64(4), nil)
		tenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "r1").Return(int64(1), nil)
		_, err := svc.RegisterTenant(ctx, tenantsvc.RegisterTenantInput{
			Email: "test@example.com", RoomID: "r1",
			ContractFiles: []*multipart.FileHeader{createMultipartFileHeader("contract.txt", []byte("txt"))},
		})
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("Room ownership fails", func(t *testing.T) {
		roomRepo.EXPECT().GetRoomByIDForManager(ctx, "m1", "r1").Return(nil, model.ErrRoomNotFound)
		_, err := svc.RegisterTenant(ctx, tenantsvc.RegisterTenantInput{ManagerID: "m1", Email: "test@example.com", Password: "pass", RoomID: "r1"})
		if !errors.Is(err, model.ErrRoomNotFound) {
			t.Errorf("expected room not found, got %v", err)
		}
	})
}

func TestTenantService_CheckCapicityOfRoom(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)
	tenantRepo := mock_model.NewMockTenantRepository(ctrl)
	roomRepo := mock_model.NewMockRoomRepository(ctrl)
	hasher := &mockPasswordHasher{}

	svc := tenantsvc.NewTenantServiceImpl(userRepo, tenantRepo, roomRepo, hasher)
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func()
		want    bool
		wantErr bool
	}{
		{
			name: "Has capacity",
			setup: func() {
				roomRepo.EXPECT().GetMaxTenants(ctx, "r1").Return(int64(4), nil)
				tenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "r1").Return(int64(2), nil)
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Full capacity",
			setup: func() {
				roomRepo.EXPECT().GetMaxTenants(ctx, "r1").Return(int64(4), nil)
				tenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "r1").Return(int64(4), nil)
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "Room error",
			setup: func() {
				roomRepo.EXPECT().GetMaxTenants(ctx, "r1").Return(int64(0), errors.New("db err"))
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "Tenant error",
			setup: func() {
				roomRepo.EXPECT().GetMaxTenants(ctx, "r1").Return(int64(4), nil)
				tenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "r1").Return(int64(0), errors.New("db err"))
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			got, err := svc.CheckCapicityOfRoom(ctx, "r1")
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckCapicityOfRoom() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("CheckCapicityOfRoom() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTenantService_ListTenant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)
	tenantRepo := mock_model.NewMockTenantRepository(ctrl)
	roomRepo := mock_model.NewMockRoomRepository(ctrl)
	hasher := &mockPasswordHasher{}

	svc := tenantsvc.NewTenantServiceImpl(userRepo, tenantRepo, roomRepo, hasher)
	ctx := context.Background()

	t.Run("ByRoomID", func(t *testing.T) {
		tenantRepo.EXPECT().ListTenantByRoomID(ctx, "m1", "r1").Return([]model.FullInfoTenant{{}}, nil)
		res, err := svc.ListTenantByRoomID(ctx, "m1", "r1")
		if err != nil || len(res) != 1 {
			t.Errorf("ListTenantByRoomID failed")
		}
	})

	t.Run("ByHouseID", func(t *testing.T) {
		tenantRepo.EXPECT().ListTenantByHouseID(ctx, "m1", "h1").Return([]model.FullInfoTenant{{}}, nil)
		res, err := svc.ListTenantByHouseID(ctx, "m1", "h1")
		if err != nil || len(res) != 1 {
			t.Errorf("ListTenantByHouseID failed")
		}
	})
}

func TestTenantService_UpdateTenantInfo(t *testing.T) {
	setupTenantTestDir()
	defer teardownTenantTestDir()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)
	tenantRepo := mock_model.NewMockTenantRepository(ctrl)
	roomRepo := mock_model.NewMockRoomRepository(ctrl)
	hasher := &mockPasswordHasher{}

	svc := tenantsvc.NewTenantServiceImpl(userRepo, tenantRepo, roomRepo, hasher)
	ctx := context.Background()

	tests := []struct {
		name    string
		input   tenantsvc.UpdateTenantInput
		setup   func()
		wantErr bool
	}{
		{
			name: "Success update with new files",
			input: tenantsvc.UpdateTenantInput{
				FullName:      ptr("New Name"),
				Phone:         ptr("123"),
				Email:         ptr("new@email.com"),
				IdentityCard:  ptr("ID456"),
				CCCDFiles:     []*multipart.FileHeader{createMultipartFileHeader("cccd.png", []byte("img"))},
				KeptCCCDPaths: ptr("old_cccd.png"),
			},
			setup: func() {
				tenantRepo.EXPECT().GetTenantByID(ctx, "m1", "t1").Return(&model.FullInfoTenant{UserID: "u1"}, nil)
				userRepo.EXPECT().GetByPhone(ctx, "123").Return(nil, errors.New("not found"))
				userRepo.EXPECT().UpdateUser(ctx, "u1", gomock.Any()).Return(&model.User{}, nil)
				tenantRepo.EXPECT().UpdateTenant(ctx, "t1", gomock.Any()).Return(&model.Tenant{}, nil)
				tenantRepo.EXPECT().GetTenantByID(ctx, "m1", "t1").Return(&model.FullInfoTenant{}, nil)
			},
			wantErr: false,
		},
		{
			name: "Tenant not found",
			setup: func() {
				tenantRepo.EXPECT().GetTenantByID(ctx, "m1", "t1").Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
		{
			name: "Update user error",
			input: tenantsvc.UpdateTenantInput{
				FullName: ptr("New Name"),
			},
			setup: func() {
				tenantRepo.EXPECT().GetTenantByID(ctx, "m1", "t1").Return(&model.FullInfoTenant{UserID: "u1"}, nil)
				userRepo.EXPECT().UpdateUser(ctx, "u1", gomock.Any()).Return(nil, errors.New("err"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			_, err := svc.UpdateTenantInfo(ctx, "m1", "t1", tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateTenantInfo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	t.Run("UpdateTenant repo error", func(t *testing.T) {
		tenantRepo.EXPECT().GetTenantByID(ctx, "m1", "t1").Return(&model.FullInfoTenant{UserID: "u1"}, nil)
		userRepo.EXPECT().UpdateUser(ctx, "u1", gomock.Any()).Return(&model.User{}, nil)
		tenantRepo.EXPECT().UpdateTenant(ctx, "t1", gomock.Any()).Return(nil, errors.New("db error"))
		_, err := svc.UpdateTenantInfo(ctx, "m1", "t1", tenantsvc.UpdateTenantInput{FullName: ptr("Name")})
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})

	t.Run("GetTenantByID (final re-fetch) error", func(t *testing.T) {
		tenantRepo.EXPECT().GetTenantByID(ctx, "m1", "t1").Return(&model.FullInfoTenant{UserID: "u1"}, nil)
		userRepo.EXPECT().UpdateUser(ctx, "u1", gomock.Any()).Return(&model.User{}, nil)
		tenantRepo.EXPECT().UpdateTenant(ctx, "t1", gomock.Any()).Return(&model.Tenant{}, nil)
		tenantRepo.EXPECT().GetTenantByID(ctx, "m1", "t1").Return(nil, errors.New("db error 2"))
		_, err := svc.UpdateTenantInfo(ctx, "m1", "t1", tenantsvc.UpdateTenantInput{FullName: ptr("Name")})
		if err == nil || err.Error() != "db error 2" {
			t.Errorf("expected db error 2, got %v", err)
		}
	})

	t.Run("New CCCD files without kept paths", func(t *testing.T) {
		tenantRepo.EXPECT().GetTenantByID(ctx, "m1", "t1").Return(&model.FullInfoTenant{UserID: "u1", CCCDPath: "old.png,old2.png"}, nil)
		userRepo.EXPECT().UpdateUser(ctx, "u1", gomock.Any()).Return(&model.User{}, nil)
		tenantRepo.EXPECT().UpdateTenant(ctx, "t1", gomock.Any()).Return(&model.Tenant{}, nil)
		tenantRepo.EXPECT().GetTenantByID(ctx, "m1", "t1").Return(&model.FullInfoTenant{}, nil)
		_, err := svc.UpdateTenantInfo(ctx, "m1", "t1", tenantsvc.UpdateTenantInput{
			CCCDFiles: []*multipart.FileHeader{createMultipartFileHeader("cccd.png", []byte("img"))},
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("New contract files with kept paths", func(t *testing.T) {
		tenantRepo.EXPECT().GetTenantByID(ctx, "m1", "t1").Return(&model.FullInfoTenant{UserID: "u1"}, nil)
		userRepo.EXPECT().UpdateUser(ctx, "u1", gomock.Any()).Return(&model.User{}, nil)
		tenantRepo.EXPECT().UpdateTenant(ctx, "t1", gomock.Any()).Return(&model.Tenant{}, nil)
		tenantRepo.EXPECT().GetTenantByID(ctx, "m1", "t1").Return(&model.FullInfoTenant{}, nil)
		_, err := svc.UpdateTenantInfo(ctx, "m1", "t1", tenantsvc.UpdateTenantInput{
			ContractFiles:     []*multipart.FileHeader{createMultipartFileHeader("contract.pdf", []byte("pdf"))},
			KeptContractPaths: ptr("old_contract.pdf"),
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("New contract files without kept paths", func(t *testing.T) {
		tenantRepo.EXPECT().GetTenantByID(ctx, "m1", "t1").Return(&model.FullInfoTenant{UserID: "u1", ContractPath: "old_c1.pdf,old_c2.pdf"}, nil)
		userRepo.EXPECT().UpdateUser(ctx, "u1", gomock.Any()).Return(&model.User{}, nil)
		tenantRepo.EXPECT().UpdateTenant(ctx, "t1", gomock.Any()).Return(&model.Tenant{}, nil)
		tenantRepo.EXPECT().GetTenantByID(ctx, "m1", "t1").Return(&model.FullInfoTenant{}, nil)
		_, err := svc.UpdateTenantInfo(ctx, "m1", "t1", tenantsvc.UpdateTenantInput{
			ContractFiles: []*multipart.FileHeader{createMultipartFileHeader("contract.pdf", []byte("pdf"))},
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestTenantService_DeleteTenant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)
	tenantRepo := mock_model.NewMockTenantRepository(ctrl)
	roomRepo := mock_model.NewMockRoomRepository(ctrl)
	hasher := &mockPasswordHasher{}

	svc := tenantsvc.NewTenantServiceImpl(userRepo, tenantRepo, roomRepo, hasher)
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "Success, room becomes available",
			setup: func() {
				tenantRepo.EXPECT().GetTenantByID(ctx, "m1", "t1").Return(&model.FullInfoTenant{UserID: "u1"}, nil)
				tenantRepo.EXPECT().DeleteTenant(ctx, "t1").Return("r1", nil)
				userRepo.EXPECT().DeactivateUser(ctx, "u1").Return(nil)
				tenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "r1").Return(int64(0), nil)
				roomRepo.EXPECT().UpdateRoomStatus(ctx, "r1", "AVAILABLE").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Success, room still occupied",
			setup: func() {
				tenantRepo.EXPECT().GetTenantByID(ctx, "m1", "t1").Return(&model.FullInfoTenant{UserID: "u1"}, nil)
				tenantRepo.EXPECT().DeleteTenant(ctx, "t1").Return("r1", nil)
				userRepo.EXPECT().DeactivateUser(ctx, "u1").Return(nil)
				tenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "r1").Return(int64(1), nil)
			},
			wantErr: false,
		},
		{
			name: "Tenant not found",
			setup: func() {
				tenantRepo.EXPECT().GetTenantByID(ctx, "m1", "t1").Return(nil, errors.New("err"))
			},
			wantErr: true,
		},
		{
			name: "Delete tenant repo error",
			setup: func() {
				tenantRepo.EXPECT().GetTenantByID(ctx, "m1", "t1").Return(&model.FullInfoTenant{UserID: "u1"}, nil)
				tenantRepo.EXPECT().DeleteTenant(ctx, "t1").Return("", errors.New("err"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := svc.DeleteTenant(ctx, "m1", "t1")
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteTenant() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	t.Run("DeactivateUser fails", func(t *testing.T) {
		tenantRepo.EXPECT().GetTenantByID(ctx, "m1", "t1").Return(&model.FullInfoTenant{UserID: "u1"}, nil)
		tenantRepo.EXPECT().DeleteTenant(ctx, "t1").Return("r1", nil)
		userRepo.EXPECT().DeactivateUser(ctx, "u1").Return(errors.New("db error"))
		err := svc.DeleteTenant(ctx, "m1", "t1")
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})

	t.Run("GetCurrentNumTenantInRoom fails", func(t *testing.T) {
		tenantRepo.EXPECT().GetTenantByID(ctx, "m1", "t1").Return(&model.FullInfoTenant{UserID: "u1"}, nil)
		tenantRepo.EXPECT().DeleteTenant(ctx, "t1").Return("r1", nil)
		userRepo.EXPECT().DeactivateUser(ctx, "u1").Return(nil)
		tenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "r1").Return(int64(0), errors.New("db error 2"))
		err := svc.DeleteTenant(ctx, "m1", "t1")
		if err == nil || err.Error() != "db error 2" {
			t.Errorf("expected db error 2, got %v", err)
		}
	})

	t.Run("UpdateRoomStatus fails (remaining=0 branch)", func(t *testing.T) {
		tenantRepo.EXPECT().GetTenantByID(ctx, "m1", "t1").Return(&model.FullInfoTenant{UserID: "u1"}, nil)
		tenantRepo.EXPECT().DeleteTenant(ctx, "t1").Return("r1", nil)
		userRepo.EXPECT().DeactivateUser(ctx, "u1").Return(nil)
		tenantRepo.EXPECT().GetCurrentNumTenantInRoom(ctx, "r1").Return(int64(0), nil)
		roomRepo.EXPECT().UpdateRoomStatus(ctx, "r1", "AVAILABLE").Return(errors.New("db error 3"))
		err := svc.DeleteTenant(ctx, "m1", "t1")
		if err == nil || err.Error() != "db error 3" {
			t.Errorf("expected db error 3, got %v", err)
		}
	})
}

// TestTenantServiceResolveTenantFilePath verifies tenant upload ownership and traversal checks.
func TestTenantServiceResolveTenantFilePath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	tenantRepository := mock_model.NewMockTenantRepository(ctrl)
	tenantService := tenantsvc.NewTenantServiceImpl(nil, tenantRepository, nil, nil)

	tenantRepository.EXPECT().
		GetTenantByFilePath(ctx, "m1", "/api/v1/tenant/files/cccd.png").
		Return(&model.FullInfoTenant{TenantID: "t1"}, nil)

	filePath, err := tenantService.ResolveTenantFilePath(ctx, "m1", "cccd.png")
	if err != nil {
		t.Fatalf("ResolveTenantFilePath error = %v", err)
	}
	if filePath != "uploads/tenants/cccd.png" {
		t.Fatalf("filePath = %q, want uploads/tenants/cccd.png", filePath)
	}

	if _, err := tenantService.ResolveTenantFilePath(ctx, "m1", "../secret.png"); !errors.Is(err, sharedsvc.ErrInvalidInput) {
		t.Fatalf("traversal err = %v, want shared.ErrInvalidInput", err)
	}
}

// mockPasswordHasher is a lightweight PasswordHasher stub for tenant service tests.
type mockPasswordHasher struct {
	hashErr    error
	compareErr error
}

func (m *mockPasswordHasher) Hash(password string) (string, error) {
	if m.hashErr != nil {
		return "", m.hashErr
	}
	return "hashed-" + password, nil
}

func (m *mockPasswordHasher) Compare(hash, password string) error {
	return m.compareErr
}

// ptr returns a pointer to v, for building optional test inputs.
func ptr[T any](v T) *T { return &v }
