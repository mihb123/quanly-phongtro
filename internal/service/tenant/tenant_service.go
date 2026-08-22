package tenant

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"strings"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/service/auth"
	"github.com/mihb123/quanly-phongtro/internal/service/shared"

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
	ResolveTenantFilePath(ctx context.Context, managerID, requestPath string) (string, error)
}

type TenantServiceImpl struct {
	hasher  auth.PasswordHasher
	users   model.UserRepository
	tenants model.TenantRepository
	rooms   model.RoomRepository
	houses  model.HouseRepository
}

type RegisterTenantInput struct {
	ManagerID    string
	Email        string
	Password     string
	FullName     string
	Phone        string
	RoomID       string
	CCCDFiles    []*multipart.FileHeader
	IdentityCard string
	StartDate    time.Time
}

// UpdateTenantInput carries the optional fields a manager can patch on a tenant.
// Nil pointer = field not provided = leave unchanged.
// File fields are optional — leave nil to keep the existing stored path.
type UpdateTenantInput struct {
	FullName      *string
	Phone         *string
	Email         *string
	IdentityCard  *string
	CCCDFiles     []*multipart.FileHeader
	KeptCCCDPaths *string
}

func NewTenantServiceImpl(user model.UserRepository, tenant model.TenantRepository, rooms model.RoomRepository, houses model.HouseRepository, hasher auth.PasswordHasher) *TenantServiceImpl {
	return &TenantServiceImpl{
		rooms:   rooms,
		users:   user,
		tenants: tenant,
		houses:  houses,
		hasher:  hasher,
	}
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

	if !auth.IsValidEmail(in.Email) {
		return nil, shared.ErrInvalidInput
	}

	if in.Phone != "" {
		existingUser, err := s.users.GetByPhone(ctx, in.Phone)
		if err == nil && existingUser != nil {
			return nil, model.ErrPhoneAlreadyExists
		}
	}

	if _, err := s.rooms.GetRoomByIDForManager(ctx, in.ManagerID, in.RoomID); err != nil {
		return nil, err
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
	if len(in.CCCDFiles) > 0 {
		cccdPaths, err := shared.SaveUploadedFiles(shared.TenantUploadDir, in.CCCDFiles)
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

	if in.Phone != nil && *in.Phone != "" && *in.Phone != existing.Phone {
		existingUser, err := s.users.GetByPhone(ctx, *in.Phone)
		if err == nil && existingUser != nil && existingUser.ID != existing.UserID {
			return nil, model.ErrPhoneAlreadyExists
		}
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
		newPaths, err := shared.SaveUploadedFiles(shared.TenantUploadDir, in.CCCDFiles)
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

// ResolveTenantFilePath verifies manager ownership before returning a local tenant upload path.
func (s *TenantServiceImpl) ResolveTenantFilePath(ctx context.Context, managerID, requestPath string) (string, error) {
	fileName, ok := shared.UploadFileName(requestPath)
	if !ok {
		return "", shared.ErrInvalidInput
	}

	storedPath := shared.TenantFileURLPrefix + fileName
	uploadDir, err := s.authorizeFileAccess(ctx, managerID, storedPath, fileName)
	if err != nil {
		return "", err
	}

	filePath, ok := shared.UploadFilePath(uploadDir, fileName)
	if !ok {
		return "", shared.ErrInvalidInput
	}
	if legacyPath, ok := legacyUploadPath(uploadDir, fileName, filePath); ok {
		return legacyPath, nil
	}
	return filePath, nil
}

// legacyUploadPath trả về đường dẫn cũ trong uploads/tenants cho file chủ nhà
// đã upload trước khi tách thư mục.
func legacyUploadPath(uploadDir, fileName, filePath string) (string, bool) {
	if uploadDir == shared.TenantUploadDir {
		return "", false
	}
	if _, err := os.Stat(filePath); err == nil || !os.IsNotExist(err) {
		return "", false
	}
	legacyPath, ok := shared.UploadFilePath(shared.TenantUploadDir, fileName)
	if !ok {
		return "", false
	}
	if _, err := os.Stat(legacyPath); err != nil {
		return "", false
	}
	return legacyPath, true
}

// authorizeFileAccess cho phép truy cập nếu file thuộc CCCD của khách thuê, hợp đồng của phòng,
// hoặc CCCD chủ nhà / hợp đồng thuê nguyên căn của nhà do manager quản lý.
// Trả về thư mục lưu trữ tương ứng với loại hồ sơ khớp.
func (s *TenantServiceImpl) authorizeFileAccess(ctx context.Context, managerID, storedPath, fileName string) (string, error) {
	for _, candidate := range []string{storedPath, fileName} {
		_, err := s.tenants.GetTenantByFilePath(ctx, managerID, candidate)
		if err == nil {
			return shared.TenantUploadDir, nil
		}
		if !errors.Is(err, model.ErrTenantNotFound) {
			return "", err
		}

		owned, roomErr := s.rooms.HasRoomWithFilePath(ctx, managerID, candidate)
		if roomErr != nil {
			return "", roomErr
		}
		if owned {
			return shared.TenantUploadDir, nil
		}

		if s.houses != nil {
			ownedHouse, houseErr := s.houses.HasHouseWithFilePath(ctx, managerID, candidate)
			if houseErr != nil {
				return "", houseErr
			}
			if ownedHouse {
				return shared.OwnerUploadDir, nil
			}
		}
	}
	return "", model.ErrTenantNotFound
}
