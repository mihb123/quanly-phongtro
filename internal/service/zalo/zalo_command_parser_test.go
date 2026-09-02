package zalo

import (
	"errors"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name string
		text string
		want ParsedCommand
	}{
		{
			name: "electricity single",
			text: "#dien 750",
			want: ParsedCommand{Type: CommandUtilitySingle, UtilityType: "dien", Entries: []RoomUtilityEntry{{NewIndex: 750, HasNewIndex: true}}},
		},
		{
			name: "water single with Vietnamese accents",
			text: "#Nước 150",
			want: ParsedCommand{Type: CommandUtilitySingle, UtilityType: "nuoc", Entries: []RoomUtilityEntry{{NewIndex: 150, HasNewIndex: true}}},
		},
		{
			name: "batch",
			text: "#dien 679qt\nP101 750\nP201 900",
			want: ParsedCommand{
				Type:        CommandUtilityBatch,
				UtilityType: "dien",
				HouseCode:   "679qt",
				Entries: []RoomUtilityEntry{
					{RoomName: "P101", NewIndex: 750, HasNewIndex: true},
					{RoomName: "P201", NewIndex: 900, HasNewIndex: true},
				},
			},
		},
		{
			name: "electricity single with bot mention prefix",
			text: "@Quản Lý Trọ #dien 750",
			want: ParsedCommand{Type: CommandUtilitySingle, UtilityType: "dien", Entries: []RoomUtilityEntry{{NewIndex: 750, HasNewIndex: true}}},
		},
		{
			name: "batch with bot mention prefix",
			text: "@Bot #dien 679qt\nP101 750\nP201 900",
			want: ParsedCommand{
				Type:        CommandUtilityBatch,
				UtilityType: "dien",
				HouseCode:   "679qt",
				Entries: []RoomUtilityEntry{
					{RoomName: "P101", NewIndex: 750, HasNewIndex: true},
					{RoomName: "P201", NewIndex: 900, HasNewIndex: true},
				},
			},
		},
		{
			name: "room target with house code",
			text: "#dien 679qt P201 661",
			want: ParsedCommand{
				Type:        CommandUtilityRoom,
				UtilityType: "dien",
				HouseCode:   "679qt",
				Entries:     []RoomUtilityEntry{{RoomName: "p201", NewIndex: 661, HasNewIndex: true}},
			},
		},
		{
			name: "room target without house code",
			text: "#dien P201 661",
			want: ParsedCommand{
				Type:        CommandUtilityRoom,
				UtilityType: "dien",
				Entries:     []RoomUtilityEntry{{RoomName: "p201", NewIndex: 661, HasNewIndex: true}},
			},
		},
		{
			name: "room target with multi word room name",
			text: "#nuoc 679qt Phòng 201 123",
			want: ParsedCommand{
				Type:        CommandUtilityRoom,
				UtilityType: "nuoc",
				HouseCode:   "679qt",
				Entries:     []RoomUtilityEntry{{RoomName: "phong 201", NewIndex: 123, HasNewIndex: true}},
			},
		},
		{
			name: "batch header without readings",
			text: "#dien 679qt",
			want: ParsedCommand{Type: CommandUtilityBatch, UtilityType: "dien", HouseCode: "679qt"},
		},
		{
			name: "update tenant with house and room",
			text: "#update-tenant 679qt P201 0912345678",
			want: ParsedCommand{Type: CommandUpdateTenantPhone, HouseCode: "679qt", RoomName: "p201", Value: "0912345678"},
		},
		{
			name: "update tenant without house code",
			text: "#update-tenant P201 0912345678",
			want: ParsedCommand{Type: CommandUpdateTenantPhone, RoomName: "p201", Value: "0912345678"},
		},
		{
			name: "update tenant in room chat with spaced phone",
			text: "#update-tenant 0912 345 678",
			want: ParsedCommand{Type: CommandUpdateTenantPhone, Value: "0912345678"},
		},
		{
			name: "update tenant normalizes international phone",
			text: "#update_tenant P201 +84912345678",
			want: ParsedCommand{Type: CommandUpdateTenantPhone, RoomName: "p201", Value: "0912345678"},
		},
		{
			name: "update tenant with invalid phone keeps value empty",
			text: "#update-tenant P201 12345",
			want: ParsedCommand{Type: CommandUpdateTenantPhone},
		},
		{
			name: "update room with explicit group id",
			text: "#update-room 679qt P201 1234567890",
			want: ParsedCommand{Type: CommandUpdateRoomGroup, HouseCode: "679qt", RoomName: "p201", Value: "1234567890"},
		},
		{
			name: "update room without group id targets current group",
			text: "#update-room 679qt P201",
			want: ParsedCommand{Type: CommandUpdateRoomGroup, HouseCode: "679qt", RoomName: "p201"},
		},
		{
			name: "update room with only room name",
			text: "#updateroom P201",
			want: ParsedCommand{Type: CommandUpdateRoomGroup, RoomName: "p201"},
		},
		{name: "help", text: "#help", want: ParsedCommand{Type: CommandHelp}},
		{name: "help vietnamese", text: "#Trợ giúp", want: ParsedCommand{Type: CommandHelp}},
		{name: "help with mention prefix", text: "@Bot #help", want: ParsedCommand{Type: CommandHelp}},
		{name: "confirm with mention prefix", text: "@Bot #ok", want: ParsedCommand{Type: CommandConfirm}},
		{name: "confirm", text: "#ok", want: ParsedCommand{Type: CommandConfirm}},
		{name: "cancel", text: "#huy", want: ParsedCommand{Type: CommandCancel}},
		{name: "period", text: "#6", want: ParsedCommand{Type: CommandPeriodSelect, PeriodMonth: 6}},
		{name: "plain confirm text", text: "ok", want: ParsedCommand{Type: CommandUnknown}},
		{name: "plain cancel text", text: "huy", want: ParsedCommand{Type: CommandUnknown}},
		{name: "plain month text", text: "6", want: ParsedCommand{Type: CommandUnknown}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseCommand(tt.text)
			assert.Equal(t, tt.want.Type, got.Type)
			assert.Equal(t, tt.want.UtilityType, got.UtilityType)
			assert.Equal(t, tt.want.HouseCode, got.HouseCode)
			assert.Equal(t, tt.want.PeriodMonth, got.PeriodMonth)
			assert.Equal(t, tt.want.RoomName, got.RoomName)
			assert.Equal(t, tt.want.Value, got.Value)
			assert.Equal(t, tt.want.Entries, got.Entries)
		})
	}
}

func TestNormalizeRoomName(t *testing.T) {
	assert.Equal(t, "101", NormalizeRoomName("P101"))
	assert.Equal(t, "101", NormalizeRoomName("Phòng 101"))
	assert.Equal(t, "101", NormalizeRoomName("phong101"))
	assert.Equal(t, "201", NormalizeRoomName("p201"))
	assert.Equal(t, "a101", NormalizeRoomName("A101"))
	assert.Equal(t, "a101", NormalizeRoomName("Phòng A-101"))
}

func TestMatchRoom(t *testing.T) {
	rooms := []model.Room{
		{ID: "r1", Name: "Phòng 101"},
		{ID: "r2", Name: "P201"},
	}

	room, err := MatchRoom("P101", rooms)
	require.NoError(t, err)
	assert.Equal(t, "r1", room.ID)

	_, err = MatchRoom("P999", rooms)
	assert.ErrorIs(t, err, model.ErrRoomNotFound)

	_, err = MatchRoom("101", append(rooms, model.Room{ID: "r3", Name: "101"}))
	assert.True(t, errors.Is(err, ErrAmbiguousRoomName))

	letteredRooms := []model.Room{
		{ID: "a101", Name: "A101"},
		{ID: "b101", Name: "B101"},
	}
	room, err = MatchRoom("A101", letteredRooms)
	require.NoError(t, err)
	assert.Equal(t, "a101", room.ID)

	room, err = MatchRoom("B101", letteredRooms)
	require.NoError(t, err)
	assert.Equal(t, "b101", room.ID)

	_, err = MatchRoom("101", letteredRooms)
	assert.ErrorIs(t, err, model.ErrRoomNotFound)
}
