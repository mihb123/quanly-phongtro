package service_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/service"
	"go.uber.org/mock/gomock"
)

var errDB = errors.New("db error")

func TestRoomService_CreateRoom(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	tests := []struct {
		name      string
		room      *model.Room
		managerID string
		mock      func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository)
		wantErr   error
	}{
		{
			name:      "Happy path",
			room:      &model.Room{HouseID: "house-1", Name: "Room 1"},
			managerID: "manager-1",
			mock: func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {
				mockHouseRepo.EXPECT().GetByID(ctx, "house-1", "manager-1").Return(&model.House{ID: "house-1"}, nil)
				mockRoomRepo.EXPECT().CreateRoom(ctx, &model.Room{HouseID: "house-1", Name: "Room 1"}).Return(nil)
			},
			wantErr: nil,
		},
		{
			name:      "Empty House ID",
			room:      &model.Room{HouseID: "   ", Name: "Room 1"},
			managerID: "manager-1",
			mock:      func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {},
			wantErr:   service.ErrInvalidHouseID,
		},
		{
			name:      "Empty Manager ID",
			room:      &model.Room{HouseID: "house-1", Name: "Room 1"},
			managerID: "   ",
			mock:      func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {},
			wantErr:   service.ErrInvalidManagerID,
		},
		{
			name:      "Repo error on create",
			room:      &model.Room{HouseID: "house-1", Name: "Room 1"},
			managerID: "manager-1",
			mock: func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {
				mockHouseRepo.EXPECT().GetByID(ctx, "house-1", "manager-1").Return(&model.House{ID: "house-1"}, nil)
				mockRoomRepo.EXPECT().CreateRoom(ctx, &model.Room{HouseID: "house-1", Name: "Room 1"}).Return(errDB)
			},
			wantErr: errDB,
		},
		{
			name:      "House ownership error",
			room:      &model.Room{HouseID: "house-1", Name: "Room 1"},
			managerID: "manager-1",
			mock: func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {
				mockHouseRepo.EXPECT().GetByID(ctx, "house-1", "manager-1").Return(nil, model.ErrHouseNotFound)
			},
			wantErr: model.ErrHouseNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRoomRepo := mock_model.NewMockRoomRepository(ctrl)
			mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
			tt.mock(mockRoomRepo, mockHouseRepo)

			s := service.NewRoomService(mockRoomRepo, mockHouseRepo)
			err := s.CreateRoom(ctx, tt.room, tt.managerID)

			if err != tt.wantErr {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestRoomService_GetRoom(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	expectedRoom := &model.Room{ID: "room-1", HouseID: "house-1"}

	tests := []struct {
		name      string
		roomID    string
		houseID   string
		managerID string
		mock      func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository)
		want      *model.Room
		wantErr   error
	}{
		{
			name:      "Happy path",
			roomID:    "room-1",
			houseID:   "house-1",
			managerID: "manager-1",
			mock: func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "manager-1").Return(true, nil)
				mockRoomRepo.EXPECT().GetRoomByID(ctx, "room-1", "house-1").Return(expectedRoom, nil)
			},
			want:    expectedRoom,
			wantErr: nil,
		},
		{
			name:      "Empty Room ID",
			roomID:    "   ",
			houseID:   "house-1",
			managerID: "manager-1",
			mock:      func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {},
			want:      nil,
			wantErr:   service.ErrInvalidRoomID,
		},
		{
			name:      "Empty House ID",
			roomID:    "room-1",
			houseID:   "   ",
			managerID: "manager-1",
			mock:      func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {},
			want:      nil,
			wantErr:   service.ErrInvalidHouseID,
		},
		{
			name:      "Empty Manager ID",
			roomID:    "room-1",
			houseID:   "house-1",
			managerID: "   ",
			mock:      func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {},
			want:      nil,
			wantErr:   service.ErrInvalidManagerID,
		},
		{
			name:      "checkOwnership - repo error",
			roomID:    "room-1",
			houseID:   "house-1",
			managerID: "manager-1",
			mock: func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "manager-1").Return(false, errDB)
			},
			want:    nil,
			wantErr: errDB,
		},
		{
			name:      "checkOwnership - not owned",
			roomID:    "room-1",
			houseID:   "house-1",
			managerID: "manager-1",
			mock: func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "manager-1").Return(false, nil)
			},
			want:    nil,
			wantErr: model.ErrHouseNotFound,
		},
		{
			name:      "Repo get room error",
			roomID:    "room-1",
			houseID:   "house-1",
			managerID: "manager-1",
			mock: func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "manager-1").Return(true, nil)
				mockRoomRepo.EXPECT().GetRoomByID(ctx, "room-1", "house-1").Return(nil, errDB)
			},
			want:    nil,
			wantErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRoomRepo := mock_model.NewMockRoomRepository(ctrl)
			mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
			tt.mock(mockRoomRepo, mockHouseRepo)

			s := service.NewRoomService(mockRoomRepo, mockHouseRepo)
			got, err := s.GetRoom(ctx, tt.roomID, tt.houseID, tt.managerID)

			if err != tt.wantErr {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestRoomService_ListRoomsByHouseID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	expectedRooms := []model.Room{{ID: "room-1"}}

	tests := []struct {
		name      string
		houseID   string
		managerID string
		page      int
		limit     int
		mock      func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository)
		want      []model.Room
		wantErr   error
	}{
		{
			name:      "Happy path",
			houseID:   "house-1",
			managerID: "manager-1",
			page:      1,
			limit:     10,
			mock: func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "manager-1").Return(true, nil)
				mockRoomRepo.EXPECT().ListRoomsByHouseID(ctx, "house-1", 10, 0).Return(expectedRooms, nil)
			},
			want:    expectedRooms,
			wantErr: nil,
		},
		{
			name:      "Empty House ID",
			houseID:   "   ",
			managerID: "manager-1",
			page:      1,
			limit:     10,
			mock:      func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {},
			want:      nil,
			wantErr:   service.ErrInvalidHouseID,
		},
		{
			name:      "Empty Manager ID",
			houseID:   "house-1",
			managerID: "   ",
			page:      1,
			limit:     10,
			mock:      func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {},
			want:      nil,
			wantErr:   service.ErrInvalidManagerID,
		},
		{
			name:      "checkOwnership error",
			houseID:   "house-1",
			managerID: "manager-1",
			page:      1,
			limit:     10,
			mock: func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "manager-1").Return(false, errDB)
			},
			want:    nil,
			wantErr: errDB,
		},
		{
			name:      "Repo error on list",
			houseID:   "house-1",
			managerID: "manager-1",
			page:      2,
			limit:     5,
			mock: func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "manager-1").Return(true, nil)
				mockRoomRepo.EXPECT().ListRoomsByHouseID(ctx, "house-1", 5, 5).Return(nil, errDB)
			},
			want:    nil,
			wantErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRoomRepo := mock_model.NewMockRoomRepository(ctrl)
			mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
			tt.mock(mockRoomRepo, mockHouseRepo)

			s := service.NewRoomService(mockRoomRepo, mockHouseRepo)
			got, err := s.ListRoomsByHouseID(ctx, tt.houseID, tt.managerID, tt.page, tt.limit)

			if err != tt.wantErr {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestRoomService_UpdateRoom(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	expectedRoom := &model.Room{ID: "room-1", Name: "Updated Room"}
	input := service.UpdateRoomInput{Name: "Updated Room", Price: 1500}

	tests := []struct {
		name      string
		roomID    string
		houseID   string
		managerID string
		input     service.UpdateRoomInput
		mock      func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository)
		want      *model.Room
		wantErr   error
	}{
		{
			name:      "Happy path",
			roomID:    "room-1",
			houseID:   "house-1",
			managerID: "manager-1",
			input:     input,
			mock: func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "manager-1").Return(true, nil)
				mockRoomRepo.EXPECT().UpdateRoom(ctx, "room-1", "house-1", gomock.Any()).Return(expectedRoom, nil)
			},
			want:    expectedRoom,
			wantErr: nil,
		},
		{
			name:      "Empty Room ID",
			roomID:    "   ",
			houseID:   "house-1",
			managerID: "manager-1",
			input:     input,
			mock:      func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {},
			want:      nil,
			wantErr:   service.ErrInvalidRoomID,
		},
		{
			name:      "Empty House ID",
			roomID:    "room-1",
			houseID:   "   ",
			managerID: "manager-1",
			input:     input,
			mock:      func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {},
			want:      nil,
			wantErr:   service.ErrInvalidHouseID,
		},
		{
			name:      "Empty Manager ID",
			roomID:    "room-1",
			houseID:   "house-1",
			managerID: "   ",
			input:     input,
			mock:      func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {},
			want:      nil,
			wantErr:   service.ErrInvalidManagerID,
		},
		{
			name:      "checkOwnership error",
			roomID:    "room-1",
			houseID:   "house-1",
			managerID: "manager-1",
			input:     input,
			mock: func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "manager-1").Return(false, nil)
			},
			want:    nil,
			wantErr: model.ErrHouseNotFound,
		},
		{
			name:      "Repo error on update",
			roomID:    "room-1",
			houseID:   "house-1",
			managerID: "manager-1",
			input:     input,
			mock: func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "manager-1").Return(true, nil)
				mockRoomRepo.EXPECT().UpdateRoom(ctx, "room-1", "house-1", gomock.Any()).Return(nil, errDB)
			},
			want:    nil,
			wantErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRoomRepo := mock_model.NewMockRoomRepository(ctrl)
			mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
			tt.mock(mockRoomRepo, mockHouseRepo)

			s := service.NewRoomService(mockRoomRepo, mockHouseRepo)
			got, err := s.UpdateRoom(ctx, tt.roomID, tt.houseID, tt.managerID, tt.input)

			if err != tt.wantErr {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestRoomService_DeleteRoom(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	tests := []struct {
		name      string
		roomID    string
		houseID   string
		managerID string
		mock      func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository)
		wantErr   error
	}{
		{
			name:      "Happy path",
			roomID:    "room-1",
			houseID:   "house-1",
			managerID: "manager-1",
			mock: func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "manager-1").Return(true, nil)
				mockRoomRepo.EXPECT().DeleteRoom(ctx, "room-1", "house-1").Return(nil)
			},
			wantErr: nil,
		},
		{
			name:      "Empty Room ID",
			roomID:    "   ",
			houseID:   "house-1",
			managerID: "manager-1",
			mock:      func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {},
			wantErr:   service.ErrInvalidRoomID,
		},
		{
			name:      "Empty House ID",
			roomID:    "room-1",
			houseID:   "   ",
			managerID: "manager-1",
			mock:      func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {},
			wantErr:   service.ErrInvalidHouseID,
		},
		{
			name:      "Empty Manager ID",
			roomID:    "room-1",
			houseID:   "house-1",
			managerID: "   ",
			mock:      func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {},
			wantErr:   service.ErrInvalidManagerID,
		},
		{
			name:      "Not owned",
			roomID:    "room-1",
			houseID:   "house-1",
			managerID: "manager-1",
			mock: func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "manager-1").Return(false, nil)
			},
			wantErr: model.ErrHouseNotFound,
		},
		{
			name:      "checkOwnership error",
			roomID:    "room-1",
			houseID:   "house-1",
			managerID: "manager-1",
			mock: func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "manager-1").Return(false, errDB)
			},
			wantErr: errDB,
		},
		{
			name:      "Repo error on delete",
			roomID:    "room-1",
			houseID:   "house-1",
			managerID: "manager-1",
			mock: func(mockRoomRepo *mock_model.MockRoomRepository, mockHouseRepo *mock_model.MockHouseRepository) {
				mockHouseRepo.EXPECT().IsHouseOwnedBy(ctx, "house-1", "manager-1").Return(true, nil)
				mockRoomRepo.EXPECT().DeleteRoom(ctx, "room-1", "house-1").Return(errDB)
			},
			wantErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRoomRepo := mock_model.NewMockRoomRepository(ctrl)
			mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
			tt.mock(mockRoomRepo, mockHouseRepo)

			s := service.NewRoomService(mockRoomRepo, mockHouseRepo)
			err := s.DeleteRoom(ctx, tt.roomID, tt.houseID, tt.managerID)

			if err != tt.wantErr {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}
