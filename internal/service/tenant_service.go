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
	ListTenantByHouseID(ctx context.Context, managerID, houseID string) ([]model.FullInfoTenant, error)
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
	ManagerID     string
	Email         string
	Password      string
	FullName      string
	Phone         string
	RoomID        string
	CCCDFiles     []*multipart.FileHeader
	ContractFiles []*multipart.FileHeader
	IdentityCard  string
	StartDate     time.Time
}

// UpdateTenantInput carries the optional fields a manager can patch on a tenant.
// Nil pointer = field not provided = leave unchanged.
// File fields are optional — leave nil to keep the existing stored path.
type UpdateTenantInput struct {
	FullName          *string
	Phone             *string
	Email             *string
	IdentityCard      *string
	CCCDFiles         []*multipart.FileHeader
	ContractFiles     []*multipart.FileHeader
	KeptCCCDPaths     *string
	KeptContractPaths *string
}

func NewTenantServiceImpl(user model.UserRepository, tenant model.TenantRepository, rooms model.RoomRepository, hasher PasswordHasher) *TenantServiceImpl {
	return &TenantServiceImpl{
		rooms:   rooms,
		users:   user,
		tenants: tenant,
		hasher:  hasher,
	}
}

func processUploadedFiles(headers []*multipart.FileHeader) (string, error) {
	var paths []string
	for _, header := range headers {
		file, err := header.Open()
		if err != nil {
			return "", err
		}
		
		ext := strings.ToLower(filepath.Ext(header.Filename))
		allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".pdf": true}
		if !allowedExts[ext] {
			file.Close()
			return "", fmt.Errorf("unsupported file extension: %s", ext)
		}

		fileName := uuid.New().String() + ext
		filePath := filepath.Join("uploads", "tenants", fileName)
		if err := saveFile(file, filePath); err != nil {
			file.Close()
			return "", err
		}
		file.Close()
		paths = append(paths, "/api/v1/tenant/files/"+fileName)
	}
	return strings.Join(paths, ","), nil
}

func removeAccents(s string) string {
	s = strings.ToLower(s)
	reps := []struct {
		new string
		old string
	}{
		{"a", "àáạảãâầấậẩẫăằắặẳẵ"},
		{"e", "èéẹẻẽêềếệểễ"},
		{"i", "ìíịỉĩ"},
		{"o", "òóọỏõôồốộổỗơờớợởỡ"},
		{"u", "ùúụủũưừứựửữ"},
		{"y", "ỳýỵỷỹ"},
		{"d", "đ"},
	}

	for _, r := range reps {
		for _, c := range r.old {
			s = strings.ReplaceAll(s, string(c), r.new)
		}
	}

	cleanName := ""
	for _, b := range []byte(s) {
		if (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') {
			cleanName += string(b)
		}
	}
	return cleanName
}

func (s *TenantServiceImpl) RegisterTenant(ctx context.Context, in RegisterTenantInput) (*model.FullInfoTenant, error) {
	if in.Email == "" {
		if in.Phone != "" {
			in.Email = in.Phone + "@tenant.local"
		} else {
			cleanName := removeAccents(in.FullName)
			if cleanName == "" {
				cleanName = "guest"
			}
			hhmm := time.Now().Format("1504")
			randomID := uuid.New().String()[:4]
			in.Email = fmt.Sprintf("%s_%s%s@guest.local", cleanName, hhmm, randomID)
		}
	}

	if in.Password == "" {
		if in.Phone != "" {
			in.Password = in.Phone
		} else {
			in.Password = uuid.New().String()[:8]
		}
	}

	if !isValidEmail(in.Email) {
		return nil, ErrInvalidInput
	}

	ok, err := s.CheckCapicityOfRoom(ctx, in.RoomID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, model.ErrMaxTenans
	}

	passwordHash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}
	newTenant := &model.User{
		Email:        in.Email,
		FullName:     in.FullName,
		PasswordHash: passwordHash,
		Phone:        in.Phone,
		Role:         model.RoleTenant,
	}
	newRoomTenant := &model.Tenant{
		RoomID:       in.RoomID,
		ManagerID:    in.ManagerID,
		IdentityCard: in.IdentityCard,
		StartDate:    in.StartDate,
		Status:       model.TenantStatusActive,
	}
	if len(in.ContractFiles) > 0 {
		contractPaths, err := processUploadedFiles(in.ContractFiles)
		if err != nil {
			return nil, err
		}
		newRoomTenant.ContractPath = contractPaths
	}

	if len(in.CCCDFiles) > 0 {
		cccdPaths, err := processUploadedFiles(in.CCCDFiles)
		if err != nil {
			return nil, err
		}
		newRoomTenant.CCCDPath = cccdPaths
	}

	err = s.tenants.CreateTenantWithAccount(ctx, newTenant, newRoomTenant)
	if err != nil {
		return nil, err
	}

	if err := s.rooms.UpdateRoomStatus(ctx, in.RoomID, "OCCUPIED"); err != nil {
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

func (s *TenantServiceImpl) ListTenantByHouseID(ctx context.Context, managerID, houseID string) ([]model.FullInfoTenant, error) {

	return s.tenants.ListTenantByHouseID(ctx, managerID, houseID)
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

	if in.KeptCCCDPaths != nil {
		tenantInput.CCCDPath = in.KeptCCCDPaths
	}
	if len(in.CCCDFiles) > 0 {
		newPaths, err := processUploadedFiles(in.CCCDFiles)
		if err != nil {
			return nil, err
		}
		if tenantInput.CCCDPath != nil && *tenantInput.CCCDPath != "" {
			combined := *tenantInput.CCCDPath + "," + newPaths
			tenantInput.CCCDPath = &combined
		} else {
			tenantInput.CCCDPath = &newPaths
		}
	}

	if in.KeptContractPaths != nil {
		tenantInput.ContractPath = in.KeptContractPaths
	}
	if len(in.ContractFiles) > 0 {
		newPaths, err := processUploadedFiles(in.ContractFiles)
		if err != nil {
			return nil, err
		}
		if tenantInput.ContractPath != nil && *tenantInput.ContractPath != "" {
			combined := *tenantInput.ContractPath + "," + newPaths
			tenantInput.ContractPath = &combined
		} else {
			tenantInput.ContractPath = &newPaths
		}
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

	// TODO: Clean up deleted files from disk if they were removed from kept_paths.
	// We skip deleting from disk right now since multiple paths need splitting.

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
