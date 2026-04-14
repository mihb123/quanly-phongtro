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
	UpdateTenantInfo(ctx context.Context, managerID, tenantID string, in UpdateTenantInput) (*model.User, error)
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
		IdentityCard: in.IdentityCard,
	}
	newRoomTenant := model.Tenant{
		RoomID:    in.RoomID,
		ManagerID: in.ManagerID,
	}
	contractFileName := uuid.New().String() + filepath.Ext(in.ContractFileHeader.Filename)
	contractFilePath := filepath.Join("uploads", "tenants", contractFileName)
	if err := saveFile(in.ContractFile, contractFilePath); err == nil {
		newTenant.ContractPath = "/api/v1/tenant/files/" + contractFileName
	} else {
		return nil, err
	}
	cccdFileName := uuid.New().String() + filepath.Ext(in.CCCDFileHeader.Filename)
	cccdFilePath := filepath.Join("uploads", "tenants", cccdFileName)
	if err := saveFile(in.CCCDFile, cccdFilePath); err == nil {
		newTenant.CCCDPath = "/api/v1/tenant/files/" + cccdFileName
	} else {
		return nil, err
	}

	ok, err := s.CheckCapicityOfRoom(ctx, in.RoomID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, model.ErrMaxTenans
	}
	err = s.users.Create(ctx, newTenant)
	if err != nil {
		return nil, err
	}
	newRoomTenant.TenantID = newTenant.ID
	err = s.tenants.AssignRoom(ctx, newRoomTenant)
	if err != nil {
		return nil, err
	}

	err = s.rooms.UpdateRoomStatus(ctx, in.RoomID, "OCCUPIED")
	if err != nil {
		return nil, err
	}

	fullInfoTenant := &model.FullInfoTenant{
		TenantID:     newTenant.ID,
		RoomID:       newRoomTenant.RoomID,
		FullName:     newTenant.FullName,
		Email:        newTenant.Email,
		Phone:        newTenant.Phone,
		ManagerID:    in.ManagerID,
		CCCDPath:     newTenant.CCCDPath,
		IdentityCard: newTenant.IdentityCard,
		ContractPath: newTenant.ContractPath,
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
// partial update on the user record identified by tenantID.
// File uploads are optional: when provided they are saved to disk and the
// resulting path is stored; when omitted the existing paths are unchanged.
func (s *TenantServiceImpl) UpdateTenantInfo(ctx context.Context, managerID, tenantID string, in UpdateTenantInput) (*model.User, error) {
	if err := s.tenants.VerifyTenantOwnership(ctx, managerID, tenantID); err != nil {
		return nil, err
	}

	// Fetch existing record so we can clean up old files after the update.
	existing, err := s.users.GetByUserID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	userInput := model.UpdateUserInput{
		FullName:     in.FullName,
		Phone:        in.Phone,
		Email:        in.Email,
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
		userInput.CCCDPath = &path
	}

	// Save contract if a new file was uploaded.
	if in.ContractFile != nil && in.ContractFileHeader != nil {
		contractFileName := uuid.New().String() + filepath.Ext(in.ContractFileHeader.Filename)
		contractFilePath := filepath.Join("uploads", "tenants", contractFileName)
		if err := saveFile(in.ContractFile, contractFilePath); err != nil {
			return nil, err
		}
		path := "/api/v1/tenant/files/" + contractFileName
		userInput.ContractPath = &path
	}

	updated, err := s.users.UpdateUser(ctx, tenantID, userInput)
	if err != nil {
		return nil, err
	}

	// Delete old files from disk only after a successful DB update.
	const urlPrefix = "/api/v1/tenant/files/"
	deleteUpload := func(urlPath string) {
		if urlPath == "" {
			return
		}
		fileName := strings.TrimPrefix(urlPath, urlPrefix)
		if fileName == "" {
			return
		}
		diskPath := filepath.Join("uploads", "tenants", fileName)
		if err := os.Remove(diskPath); err != nil {
			return
		}
	}

	if userInput.CCCDPath != nil {
		deleteUpload(existing.CCCDPath)
	}
	if userInput.ContractPath != nil {
		deleteUpload(existing.ContractPath)
	}

	return updated, nil
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

// DeleteTenant removes the tenant from the tenants table.
// After deletion it counts how many tenants remain in the room;
// if the count reaches zero the room status is reset to AVAILABLE.
func (s *TenantServiceImpl) DeleteTenant(ctx context.Context, managerID, tenantID string) error {

	if err := s.tenants.VerifyTenantOwnership(ctx, managerID, tenantID); err != nil {
		return err
	}

	roomID, err := s.tenants.DeleteTenant(ctx, tenantID)
	if err != nil {
		return err
	}

	// Remove the user record and retrieve stored file paths in one query.
	_, _, err = s.users.DeactivateUser(ctx, tenantID)
	if err != nil {
		return err
	}

	// Delete uploaded files from disk.  Extract the filename from the URL path
	// (everything after "/api/v1/tenant/files/") and build the local disk path.
	// const urlPrefix = "/api/v1/tenant/files/"
	// deleteUpload := func(urlPath string) {
	// 	if urlPath == "" {
	// 		return
	// 	}
	// 	fileName := strings.TrimPrefix(urlPath, urlPrefix)
	// 	if fileName == "" {
	// 		return
	// 	}
	// 	diskPath := filepath.Join("uploads", "tenants", fileName)
	// 	if err := os.Remove(diskPath); err != nil {
	// 		return
	// 	}

	// }

	// deleteUpload(cccdPath)
	// deleteUpload(contractPath)

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
