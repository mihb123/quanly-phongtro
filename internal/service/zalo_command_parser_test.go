package service

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
