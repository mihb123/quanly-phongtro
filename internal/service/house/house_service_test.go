package house_test

import (
	"context"
	"errors"
	"testing"

	sharedsvc "github.com/mihb123/quanly-phongtro/internal/service/shared"

	housesvc "github.com/mihb123/quanly-phongtro/internal/service/house"

	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"go.uber.org/mock/gomock"
)

func TestHouseService_CreateHouse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_model.NewMockHouseRepository(ctrl)
	mockCostRepo := mock_model.NewMockHouseCostRepository(ctrl)
	houseService := housesvc.NewHouseServiceImpt(mockRepo, mockCostRepo)
	ctx := context.Background()

	house := &model.House{
		Name:      "House 1",
		ManagerID: "manager-1",
	}

	// Happy path: house_code chưa tồn tại toàn hệ thống → cho phép tạo.
	mockRepo.EXPECT().IsHouseCodeTaken(ctx, "", "").Return(false, nil)
	mockRepo.EXPECT().CreateHouse(ctx, house).Return(nil)
	mockCostRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

	err := houseService.CreateHouse(ctx, house)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	// Error case
	expectedErr := errors.New("db error")
	mockRepo.EXPECT().IsHouseCodeTaken(ctx, "", "").Return(false, nil)
	mockRepo.EXPECT().CreateHouse(ctx, house).Return(expectedErr)

	err = houseService.CreateHouse(ctx, house)
	if err != expectedErr {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}

	// Seed cost error case
	mockRepo.EXPECT().IsHouseCodeTaken(ctx, "", "").Return(false, nil)
	mockRepo.EXPECT().CreateHouse(ctx, house).Return(nil)
	mockCostRepo.EXPECT().Create(ctx, gomock.Any()).Return(expectedErr)

	err = houseService.CreateHouse(ctx, house)
	if err != expectedErr {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}

	// Duplicate house code: mã đã bị nhà khác dùng → từ chối, không gọi CreateHouse.
	mockRepo.EXPECT().IsHouseCodeTaken(ctx, "", "").Return(true, nil)

	err = houseService.CreateHouse(ctx, house)
	if !errors.Is(err, model.ErrHouseCodeExists) {
		t.Errorf("expected ErrHouseCodeExists, got %v", err)
	}
}

func TestHouseService_GetHouseByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_model.NewMockHouseRepository(ctrl)
	houseService := housesvc.NewHouseServiceImpt(mockRepo, nil)
	ctx := context.Background()

	t.Run("Happy path", func(t *testing.T) {
		expectedHouse := &model.House{ID: "house-1", ManagerID: "manager-1"}
		mockRepo.EXPECT().GetByID(ctx, "house-1", "manager-1").Return(expectedHouse, nil)

		house, err := houseService.GetHouseByID(ctx, "house-1", "manager-1")
		if err != nil {
			t.Errorf("error was not expected: %s", err)
		}
		if house != expectedHouse {
			t.Errorf("expected %v, got %v", expectedHouse, house)
		}
	})

	t.Run("Empty ID", func(t *testing.T) {
		_, err := houseService.GetHouseByID(ctx, "", "manager-1")
		if err != sharedsvc.ErrInvalidHouseID {
			t.Errorf("expected sharedsvc.ErrInvalidHouseID, got %v", err)
		}
	})

	t.Run("Empty Manager ID", func(t *testing.T) {
		_, err := houseService.GetHouseByID(ctx, "house-1", "  ")
		if err != sharedsvc.ErrInvalidManagerID {
			t.Errorf("expected sharedsvc.ErrInvalidManagerID, got %v", err)
		}
	})

	t.Run("Not found / DB error", func(t *testing.T) {
		expectedErr := errors.New("db error")
		mockRepo.EXPECT().GetByID(ctx, "house-1", "manager-1").Return(nil, expectedErr)

		_, err := houseService.GetHouseByID(ctx, "house-1", "manager-1")
		if err != expectedErr {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
	})
}

func TestHouseService_ListHouseByManagerID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_model.NewMockHouseRepository(ctrl)
	houseService := housesvc.NewHouseServiceImpt(mockRepo, nil)
	ctx := context.Background()

	t.Run("Happy path", func(t *testing.T) {
		expectedHouses := []model.House{
			{ID: "house-1"},
		}
		mockRepo.EXPECT().ListHouseByManagerID(ctx, "manager-1", 10, 0, "search").Return(expectedHouses, nil)

		houses, err := houseService.ListHouseByManagerID(ctx, "manager-1", 1, 10, "search")
		if err != nil {
			t.Errorf("error was not expected: %s", err)
		}
		if len(houses) != 1 {
			t.Errorf("expected 1 house, got %d", len(houses))
		}
	})

	t.Run("DB error", func(t *testing.T) {
		expectedErr := errors.New("db error")
		mockRepo.EXPECT().ListHouseByManagerID(ctx, "manager-1", 10, 0, "search").Return(nil, expectedErr)

		_, err := houseService.ListHouseByManagerID(ctx, "manager-1", 1, 10, "search")
		if err != expectedErr {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
	})
}

func TestHouseService_UpdateHouse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_model.NewMockHouseRepository(ctrl)
	houseService := housesvc.NewHouseServiceImpt(mockRepo, nil)
	ctx := context.Background()

	t.Run("Happy path", func(t *testing.T) {
		input := housesvc.UpdateHouseInput{
			Name: "Updated Name",
		}
		params := model.UpdateHouseParams{
			Name: "Updated Name",
		}
		expectedHouse := &model.House{ID: "house-1", Name: "Updated Name"}
		mockRepo.EXPECT().IsHouseCodeTaken(ctx, "", "house-1").Return(false, nil)
		mockRepo.EXPECT().UpdateHouse(ctx, "house-1", "manager-1", params).Return(expectedHouse, nil)

		house, err := houseService.UpdateHouse(ctx, "house-1", "manager-1", input)
		if err != nil {
			t.Errorf("error was not expected: %s", err)
		}
		if house != expectedHouse {
			t.Errorf("expected %v, got %v", expectedHouse, house)
		}
	})

	t.Run("DB error", func(t *testing.T) {
		input := housesvc.UpdateHouseInput{
			Name: "Updated Name",
		}
		params := model.UpdateHouseParams{
			Name: "Updated Name",
		}
		expectedErr := errors.New("db error")
		mockRepo.EXPECT().IsHouseCodeTaken(ctx, "", "house-1").Return(false, nil)
		mockRepo.EXPECT().UpdateHouse(ctx, "house-1", "manager-1", params).Return(nil, expectedErr)

		_, err := houseService.UpdateHouse(ctx, "house-1", "manager-1", input)
		if err != expectedErr {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
	})

	t.Run("Duplicate code on another house", func(t *testing.T) {
		input := housesvc.UpdateHouseInput{Name: "Updated Name", HouseCode: "dup"}
		mockRepo.EXPECT().IsHouseCodeTaken(ctx, "dup", "house-1").Return(true, nil)

		_, err := houseService.UpdateHouse(ctx, "house-1", "manager-1", input)
		if !errors.Is(err, model.ErrHouseCodeExists) {
			t.Errorf("expected ErrHouseCodeExists, got %v", err)
		}
	})

	t.Run("Keeps its own code", func(t *testing.T) {
		input := housesvc.UpdateHouseInput{Name: "Updated Name", HouseCode: "same"}
		params := model.UpdateHouseParams{Name: "Updated Name", HouseCode: "same"}
		expectedHouse := &model.House{ID: "house-1", Name: "Updated Name", HouseCode: "same"}
		// Mã bỏ qua chính nhà đang sửa (excludeID) → không coi là trùng → cho phép cập nhật.
		mockRepo.EXPECT().IsHouseCodeTaken(ctx, "same", "house-1").Return(false, nil)
		mockRepo.EXPECT().UpdateHouse(ctx, "house-1", "manager-1", params).Return(expectedHouse, nil)

		_, err := houseService.UpdateHouse(ctx, "house-1", "manager-1", input)
		if err != nil {
			t.Errorf("error was not expected: %s", err)
		}
	})
}

// UpdateHouseDocuments phải giữ lại đúng các file cũ được chọn khi không có file mới tải lên.
func TestHouseService_UpdateHouseDocuments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_model.NewMockHouseRepository(ctrl)
	houseService := housesvc.NewHouseServiceImpt(mockRepo, nil)
	ctx := context.Background()

	existing := &model.House{
		ID:                "house-1",
		OwnerCCCDPath:     "a.png,b.png",
		OwnerContractPath: "c.pdf",
	}

	t.Run("Keeps current paths when nothing is sent", func(t *testing.T) {
		mockRepo.EXPECT().GetByID(ctx, "house-1", "manager-1").Return(existing, nil)
		mockRepo.EXPECT().UpdateHouseDocuments(ctx, "house-1", "manager-1", "a.png,b.png", "c.pdf").Return(existing, nil)

		if _, err := houseService.UpdateHouseDocuments(ctx, "house-1", "manager-1", housesvc.UpdateHouseDocumentsInput{}); err != nil {
			t.Errorf("error was not expected: %s", err)
		}
	})

	t.Run("Drops removed files and clears a group", func(t *testing.T) {
		keptCCCD := "b.png"
		emptyContract := ""
		mockRepo.EXPECT().GetByID(ctx, "house-1", "manager-1").Return(existing, nil)
		mockRepo.EXPECT().UpdateHouseDocuments(ctx, "house-1", "manager-1", "b.png", "").Return(existing, nil)

		_, err := houseService.UpdateHouseDocuments(ctx, "house-1", "manager-1", housesvc.UpdateHouseDocumentsInput{
			KeptCCCDPaths:     &keptCCCD,
			KeptContractPaths: &emptyContract,
		})
		if err != nil {
			t.Errorf("error was not expected: %s", err)
		}
	})

	t.Run("Invalid house ID", func(t *testing.T) {
		if _, err := houseService.UpdateHouseDocuments(ctx, "  ", "manager-1", housesvc.UpdateHouseDocumentsInput{}); !errors.Is(err, sharedsvc.ErrInvalidHouseID) {
			t.Errorf("expected ErrInvalidHouseID, got %v", err)
		}
	})

	t.Run("Invalid manager ID", func(t *testing.T) {
		if _, err := houseService.UpdateHouseDocuments(ctx, "house-1", "", housesvc.UpdateHouseDocumentsInput{}); !errors.Is(err, sharedsvc.ErrInvalidManagerID) {
			t.Errorf("expected ErrInvalidManagerID, got %v", err)
		}
	})
}

func TestHouseService_DeleteHouse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_model.NewMockHouseRepository(ctrl)
	houseService := housesvc.NewHouseServiceImpt(mockRepo, nil)
	ctx := context.Background()

	t.Run("Happy path", func(t *testing.T) {
		mockRepo.EXPECT().DeleteHouse(ctx, "house-1", "manager-1").Return(nil)

		err := houseService.DeleteHouse(ctx, "house-1", "manager-1")
		if err != nil {
			t.Errorf("error was not expected: %s", err)
		}
	})

	t.Run("Empty ID", func(t *testing.T) {
		err := houseService.DeleteHouse(ctx, "", "manager-1")
		if err != sharedsvc.ErrInvalidHouseID {
			t.Errorf("expected sharedsvc.ErrInvalidHouseID, got %v", err)
		}
	})

	t.Run("Empty Manager ID", func(t *testing.T) {
		err := houseService.DeleteHouse(ctx, "house-1", "")
		if err != sharedsvc.ErrInvalidManagerID {
			t.Errorf("expected sharedsvc.ErrInvalidManagerID, got %v", err)
		}
	})

	t.Run("DB error", func(t *testing.T) {
		expectedErr := errors.New("db error")
		mockRepo.EXPECT().DeleteHouse(ctx, "house-1", "manager-1").Return(expectedErr)

		err := houseService.DeleteHouse(ctx, "house-1", "manager-1")
		if err != expectedErr {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
	})
}
