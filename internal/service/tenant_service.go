package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mihb123/quanly-phongtro/internal/model"
)

type TenantService interface {
	RegisterTenant(ctx context.Context, in RegisterTenantInput) (*model.FullInfoTenant, error)
	CheckCapicityOfRoom(ctx context.Context, roomID string) (bool, error)
	ListTenantByRoomID(ctx context.Context, managerID, roomID string) ([]model.FullInfoTenant, error)
	UpdateTenantInfo(ctx context.Context, managerID, tenantID string, in UpdateTenantInput) (*model.FullInfoTenant, error)
	DeleteTenant(ctx context.Context, managerID, tenantID string) error
}

type TenantServiceImpl struct {
	hasher  PasswordHasher
	users   model.UserRepository
	tenants model.TenantRepository
	rooms   model.RoomRepository
}

type RegisterTenantInput struct {
	ManagerID          string
	Email              string
	Password           string
	FullName           string
	Phone              string
	RoomID             string
	CCCDFile           multipart.File
	CCCDFileHeader     *multipart.FileHeader
	ContractFile       multipart.File
	ContractFileHeader *multipart.FileHeader
	IdentityCard       string
	StartDate          time.Time
}

// UpdateTenantInput carries the optional fields a manager can patch on a tenant.
// Nil pointer = field not provided = leave unchanged.
// File fields are optional — leave nil to keep the existing stored path.
type UpdateTenantInput struct {
	FullName           *string
	Phone              *string
	Email              *string
	IdentityCard       *string
	CCCDFile           multipart.File
	CCCDFileHeader     *multipart.FileHeader
	ContractFile       multipart.File
	ContractFileHeader *multipart.FileHeader
}

func NewTenantServiceImpl(user model.UserRepository, tenant model.TenantRepository, rooms model.RoomRepository, hasher PasswordHasher) *TenantServiceImpl {
	return &TenantServiceImpl{
		rooms:   rooms,
		users:   user,
		tenants: tenant,
		hasher:  hasher,
	}
}

func (s *TenantServiceImpl) RegisterTenant(ctx context.Context, in RegisterTenantInput) (*model.FullInfoTenant, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	password := strings.TrimSpace(in.Password)
	fullName := strings.TrimSpace(in.FullName)
	phone := strings.TrimSpace(in.Phone)

	if !isValidEmail(email) || len(password) < 6 {
		return nil, ErrInvalidInput
	}

	ok, err := s.CheckCapicityOfRoom(ctx, in.RoomID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, model.ErrMaxTenans
	}

	passwordHash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, err
	}
	newTenant := &model.User{
		Email:        email,
		FullName:     fullName,
		PasswordHash: passwordHash,
		Phone:        phone,
		Role:         model.RoleTenant,
	}
	newRoomTenant := &model.Tenant{
		RoomID:       in.RoomID,
		ManagerID:    in.ManagerID,
		IdentityCard: in.IdentityCard,
		StartDate:    in.StartDate,
		Status:       model.TenantStatusActive,
	}
	if in.ContractFile != nil && in.ContractFileHeader != nil {
		contractFileName := uuid.New().String() + filepath.Ext(in.ContractFileHeader.Filename)
		contractFilePath := filepath.Join("uploads", "tenants", contractFileName)
		if err := saveFile(in.ContractFile, contractFilePath); err == nil {
			newRoomTenant.ContractPath = "/api/v1/tenant/files/" + contractFileName
		} else {
			return nil, err
		}
	}

	if in.CCCDFile != nil && in.CCCDFileHeader != nil {
		cccdFileName := uuid.New().String() + filepath.Ext(in.CCCDFileHeader.Filename)
		cccdFilePath := filepath.Join("uploads", "tenants", cccdFileName)
		if err := saveFile(in.CCCDFile, cccdFilePath); err == nil {
			newRoomTenant.CCCDPath = "/api/v1/tenant/files/" + cccdFileName
		} else {
			if newRoomTenant.ContractPath != "" {
				contractFilePath := filepath.Join("uploads", "tenants", filepath.Base(newRoomTenant.ContractPath))
				if removeErr := os.Remove(contractFilePath); removeErr != nil && !os.IsNotExist(removeErr) {
					return nil, fmt.Errorf("%w; cleanup contract upload: %v", err, removeErr)
				}
			}
			return nil, err
		}
	}

	err = s.tenants.CreateTenantWithAccount(ctx, newTenant, newRoomTenant)
	if err != nil {
		if newRoomTenant.ContractPath != "" {
			contractFilePath := filepath.Join("uploads", "tenants", filepath.Base(newRoomTenant.ContractPath))
			if removeErr := os.Remove(contractFilePath); removeErr != nil && !os.IsNotExist(removeErr) {
				return nil, fmt.Errorf("%w; cleanup contract upload: %v", err, removeErr)
			}
		}
		if newRoomTenant.CCCDPath != "" {
			cccdFilePath := filepath.Join("uploads", "tenants", filepath.Base(newRoomTenant.CCCDPath))
			if removeErr := os.Remove(cccdFilePath); removeErr != nil && !os.IsNotExist(removeErr) {
				return nil, fmt.Errorf("%w; cleanup cccd upload: %v", err, removeErr)
			}
		}
		return nil, err
	}

	fullInfoTenant := &model.FullInfoTenant{
		TenantID:     newRoomTenant.ID,
		UserID:       newTenant.ID,
		RoomID:       newRoomTenant.RoomID,
		FullName:     newTenant.FullName,
		Email:        newTenant.Email,
		Phone:        newTenant.Phone,
		ManagerID:    in.ManagerID,
		CCCDPath:     newRoomTenant.CCCDPath,
		IdentityCard: newRoomTenant.IdentityCard,
		ContractPath: newRoomTenant.ContractPath,
		StartDate:    newRoomTenant.StartDate.Format("2006-01-02"),
		Status:       string(newRoomTenant.Status),
	}

	return fullInfoTenant, nil
}

func (s TenantServiceImpl) CheckCapicityOfRoom(ctx context.Context, roomID string) (bool, error) {
	maxTenant, err := s.rooms.GetMaxTenants(ctx, roomID)
	if err != nil {
		return false, err
	}
	numTenant, err := s.tenants.GetCurrentNumTenantInRoom(ctx, roomID)
	if err != nil {
		return false, err
	}
	if numTenant < maxTenant {
		return true, nil
	}
	return false, nil
}

func (s *TenantServiceImpl) ListTenantByRoomID(ctx context.Context, managerID, roomID string) ([]model.FullInfoTenant, error) {

	return s.tenants.ListTenantByRoomID(ctx, managerID, roomID)
}

// UpdateTenantInfo verifies that managerID owns the tenant, then performs a
// partial update across account fields and tenant profile fields.
// File uploads are optional: when provided they are saved to disk and the
// resulting path is stored; when omitted the existing paths are unchanged.
func (s *TenantServiceImpl) UpdateTenantInfo(ctx context.Context, managerID, tenantID string, in UpdateTenantInput) (*model.FullInfoTenant, error) {
	existing, err := s.tenants.GetTenantByID(ctx, managerID, tenantID)
	if err != nil {
		return nil, err
	}

	userInput := model.UpdateUserInput{
		FullName: in.FullName,
		Phone:    in.Phone,
		Email:    in.Email,
	}
	tenantInput := model.UpdateTenantInput{
		IdentityCard: in.IdentityCard,
	}

	// Save CCCD image if a new file was uploaded.
	if in.CCCDFile != nil && in.CCCDFileHeader != nil {
		cccdFileName := uuid.New().String() + filepath.Ext(in.CCCDFileHeader.Filename)
		cccdFilePath := filepath.Join("uploads", "tenants", cccdFileName)
		if err := saveFile(in.CCCDFile, cccdFilePath); err != nil {
			return nil, err
		}
		path := "/api/v1/tenant/files/" + cccdFileName
		tenantInput.CCCDPath = &path
	}

	// Save contract if a new file was uploaded.
	if in.ContractFile != nil && in.ContractFileHeader != nil {
		contractFileName := uuid.New().String() + filepath.Ext(in.ContractFileHeader.Filename)
		contractFilePath := filepath.Join("uploads", "tenants", contractFileName)
		if err := saveFile(in.ContractFile, contractFilePath); err != nil {
			return nil, err
		}
		path := "/api/v1/tenant/files/" + contractFileName
		tenantInput.ContractPath = &path
	}

	if _, err := s.users.UpdateUser(ctx, existing.UserID, userInput); err != nil {
		return nil, err
	}

	if _, err := s.tenants.UpdateTenant(ctx, tenantID, tenantInput); err != nil {
		return nil, err
	}

	updated, err := s.tenants.GetTenantByID(ctx, managerID, tenantID)
	if err != nil {
		return nil, err
	}

	// Delete old files from disk only after a successful DB update.
	const urlPrefix = "/api/v1/tenant/files/"
	deleteUpload := func(urlPath string) error {
		if urlPath == "" {
			return nil
		}
		fileName := strings.TrimPrefix(urlPath, urlPrefix)
		if fileName == "" {
			return nil
		}
		diskPath := filepath.Join("uploads", "tenants", fileName)
		if err := os.Remove(diskPath); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		return nil
	}

	if tenantInput.CCCDPath != nil {
		if err := deleteUpload(existing.CCCDPath); err != nil {
			return nil, err
		}
	}
	if tenantInput.ContractPath != nil {
		if err := deleteUpload(existing.ContractPath); err != nil {
			return nil, err
		}
	}

	return updated, nil
}

// DeleteTenant marks the tenant inactive and deactivates the login account.
// After that it counts how many active tenants remain in the room;
// if the count reaches zero the room status is reset to AVAILABLE.
func (s *TenantServiceImpl) DeleteTenant(ctx context.Context, managerID, tenantID string) error {

	existing, err := s.tenants.GetTenantByID(ctx, managerID, tenantID)
	if err != nil {
		return err
	}

	roomID, err := s.tenants.DeleteTenant(ctx, tenantID)
	if err != nil {
		return err
	}

	if err := s.users.DeactivateUser(ctx, existing.UserID); err != nil {
		return err
	}

	remaining, err := s.tenants.GetCurrentNumTenantInRoom(ctx, roomID)
	if err != nil {
		return err
	}

	if remaining == 0 {
		if err := s.rooms.UpdateRoomStatus(ctx, roomID, "AVAILABLE"); err != nil {
			return err
		}
	}

	return nil
}

func saveFile(file io.Reader, path string) error {
	dst, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("cannot create path %v", err)
	}
	defer dst.Close()
	_, err = io.Copy(dst, file)
	if err != nil {
		return fmt.Errorf("cannot save file :%v", err)
	}
	return nil
}
