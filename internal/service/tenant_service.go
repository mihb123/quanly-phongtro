package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

var (
	ErrInvalidTenantID = errors.New("invalid tenant id")
	ErrUnauthorized    = errors.New("unauthorized action")
	ErrRoomFull        = errors.New("room has reached maximum capacity")
)

type TenantService interface {
	CreateTenant(ctx context.Context, tenant *model.Tenant, userRole string, userID string) error
	GetTenant(ctx context.Context, id string, userRole, userID string) (*model.Tenant, error)
	GetTenantByRoomID(ctx context.Context, roomID string, userRole, userID string) ([]*model.Tenant, error)
	UpdateTenantInfo(ctx context.Context, id string, userRole string, userID string, params model.UpdateTenantParams) (*model.Tenant, error)
	DeleteTenant(ctx context.Context, id string, userRole string, userID string) error
	TerminateTenant(ctx context.Context, id string, userRole string, userID string) error
}

type TenantServiceImpl struct {
	tenantRepo model.TenantRepository
	roomRepo   model.RoomRepository
}

func NewTenantService(tenantRepo model.TenantRepository, roomRepo model.RoomRepository) TenantService {
	return &TenantServiceImpl{tenantRepo: tenantRepo, roomRepo: roomRepo}
}

func (s *TenantServiceImpl) CreateTenant(ctx context.Context, tenant *model.Tenant, userRole string, userID string) error {
	if strings.TrimSpace(tenant.RoomID) == "" {
		return ErrInvalidRoomID
	}
	if userRole != "MANAGER" {
		return ErrUnauthorized
	}

	// Check room capacity
	room, err := s.roomRepo.GetRoomByID(ctx, tenant.RoomID, "")
	if err != nil {
		return err
	}

	count, err := s.tenantRepo.CountActiveTenantsByRoomID(ctx, tenant.RoomID)
	if err != nil {
		return err
	}

	if count >= room.MaxTennants {
		return ErrRoomFull
	}

	// Create the tenant
	err = s.tenantRepo.CreateTenant(ctx, tenant)
	if err != nil {
		return err
	}

	// Automatically mark room as OCCUPIED
	status := "OCCUPIED"
	_, err = s.roomRepo.UpdateRoom(ctx, tenant.RoomID, "", model.UpdateRoomParams{
		Status: &status,
	})

	return err
}

func (s *TenantServiceImpl) GetTenant(ctx context.Context, id string, userRole, userID string) (*model.Tenant, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrInvalidTenantID
	}
	// For MVP, we simply allow MANAGER role or the TENANT themself to read it.
	tenant, err := s.tenantRepo.GetTenantByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if userRole == "TENANT" && tenant.CreatedBy != userID {
		// Just a simple check for MVP, in reality we'd link user table id directly to tenant
		return nil, ErrUnauthorized
	}

	return tenant, nil
}

func (s *TenantServiceImpl) GetTenantByRoomID(ctx context.Context, roomID string, userRole, userID string) ([]*model.Tenant, error) {
	if strings.TrimSpace(roomID) == "" {
		return nil, ErrInvalidRoomID
	}

	tenants, err := s.tenantRepo.ListActiveTenantsByRoomID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	// TODO: verify relationship depending on the caller userRole and userID

	return tenants, nil
}

func (s *TenantServiceImpl) TerminateTenant(ctx context.Context, id string, userRole string, userID string) error {
	if strings.TrimSpace(id) == "" {
		return ErrInvalidTenantID
	}
	if userRole != "MANAGER" {
		return ErrUnauthorized
	}

	tenant, err := s.tenantRepo.GetTenantByID(ctx, id)
	if err != nil {
		return err
	}

	if tenant.CreatedBy != userID {
		return ErrUnauthorized
	}

	now := time.Now()
	err = s.tenantRepo.UpdateTenantStatus(ctx, id, model.TenantStatusInactive, &now)
	if err != nil {
		return err
	}

	// Automatically mark room as AVAILABLE if no more active tenants
	count, err := s.tenantRepo.CountActiveTenantsByRoomID(ctx, tenant.RoomID)
	if err == nil && count == 0 {
		status := "AVAILABLE"
		_, err = s.roomRepo.UpdateRoom(ctx, tenant.RoomID, "", model.UpdateRoomParams{
			Status: &status,
		})
	}

	return err
}

func (s *TenantServiceImpl) UpdateTenantInfo(ctx context.Context, id string, userRole string, userID string, params model.UpdateTenantParams) (*model.Tenant, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrInvalidTenantID
	}
	if userRole != "MANAGER" {
		return nil, ErrUnauthorized
	}

	tenant, err := s.tenantRepo.GetTenantByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tenant.CreatedBy != userID {
		return nil, ErrUnauthorized
	}

	tenant, err = s.tenantRepo.UpdateTenantInfo(ctx, id, params)
	if err != nil {
		return nil, err
	}
	return tenant, nil
}

func (s *TenantServiceImpl) DeleteTenant(ctx context.Context, id string, userRole string, userID string) error {
	if strings.TrimSpace(id) == "" {
		return ErrInvalidTenantID
	}
	if userRole != "MANAGER" {
		return ErrUnauthorized
	}

	tenant, err := s.tenantRepo.GetTenantByID(ctx, id)
	if err != nil {
		return err
	}

	if tenant.CreatedBy != userID {
		return ErrUnauthorized
	}

	err = s.tenantRepo.DeleteTenant(ctx, id)
	if err != nil {
		return err
	}

	// Update room to AVAILABLE if it was the last tenant
	count, err := s.tenantRepo.CountActiveTenantsByRoomID(ctx, tenant.RoomID)
	if err == nil && count == 0 {
		status := "AVAILABLE"
		_, err = s.roomRepo.UpdateRoom(ctx, tenant.RoomID, "", model.UpdateRoomParams{
			Status: &status,
		})
	}

	return err
}
