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

	mockRepo.EXPECT().CreateHouse(ctx, house).Return(nil)
	mockCostRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

	err := houseService.CreateHouse(ctx, house)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	// Error case
	expectedErr := errors.New("db error")
	mockRepo.EXPECT().CreateHouse(ctx, house).Return(expectedErr)

	err = houseService.CreateHouse(ctx, house)
	if err != expectedErr {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}

	// Seed cost error case
	mockRepo.EXPECT().CreateHouse(ctx, house).Return(nil)
	mockCostRepo.EXPECT().Create(ctx, gomock.Any()).Return(expectedErr)

	err = houseService.CreateHouse(ctx, house)
	if err != expectedErr {
		t.Errorf("expected %v, got %v", expectedErr, err)
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
			t.Errorf("expected shared.ErrInvalidHouseID, got %v", err)
		}
	})

	t.Run("Empty Manager ID", func(t *testing.T) {
		_, err := houseService.GetHouseByID(ctx, "house-1", "  ")
		if err != sharedsvc.ErrInvalidManagerID {
			t.Errorf("expected shared.ErrInvalidManagerID, got %v", err)
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
		mockRepo.EXPECT().UpdateHouse(ctx, "house-1", "manager-1", params).Return(nil, expectedErr)

		_, err := houseService.UpdateHouse(ctx, "house-1", "manager-1", input)
		if err != expectedErr {
			t.Errorf("expected %v, got %v", expectedErr, err)
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
			t.Errorf("expected shared.ErrInvalidHouseID, got %v", err)
		}
	})

	t.Run("Empty Manager ID", func(t *testing.T) {
		err := houseService.DeleteHouse(ctx, "house-1", "")
		if err != sharedsvc.ErrInvalidManagerID {
			t.Errorf("expected shared.ErrInvalidManagerID, got %v", err)
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
