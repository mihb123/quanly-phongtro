package service

import (
	"errors"
	"strconv"
	"strings"
	"unicode"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"golang.org/x/text/unicode/norm"
)

type CommandType int

const (
	CommandUnknown CommandType = iota
	CommandUtilitySingle
	CommandUtilityBatch
	CommandConfirm
	CommandCancel
	CommandPeriodSelect
)

type ParsedCommand struct {
	Type        CommandType
	UtilityType string
	HouseCode   string
	Entries     []RoomUtilityEntry
	PeriodMonth int
}

type RoomUtilityEntry struct {
	RoomName    string
	NewIndex    int
	HasNewIndex bool
}

var ErrAmbiguousRoomName = errors.New("ambiguous room name")

// ParseCommand parses a raw chat message into a structured invoice command.
func ParseCommand(text string) *ParsedCommand {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return &ParsedCommand{Type: CommandUnknown}
	}

	normalizedText := normalizeVietnamese(strings.ToLower(trimmed))
	compactText := strings.Join(strings.Fields(normalizedText), "")
	if strings.HasPrefix(compactText, "#") {
		replyToken := strings.TrimPrefix(compactText, "#")
		switch replyToken {
		case "ok":
			return &ParsedCommand{Type: CommandConfirm}
		case "huy":
			return &ParsedCommand{Type: CommandCancel}
		}

		if month, err := strconv.Atoi(replyToken); err == nil && month >= 1 && month <= 12 {
			return &ParsedCommand{Type: CommandPeriodSelect, PeriodMonth: month}
		}
	}

	lines := compactNonEmptyLines(trimmed)
	if len(lines) == 0 {
		return &ParsedCommand{Type: CommandUnknown}
	}

	firstLine := normalizeVietnamese(strings.ToLower(lines[0]))
	fields := strings.Fields(firstLine)
	if len(fields) == 0 {
		return &ParsedCommand{Type: CommandUnknown}
	}

	utilityType := utilityTypeFromToken(fields[0])
	if utilityType == "" {
		return &ParsedCommand{Type: CommandUnknown}
	}

	if len(lines) > 1 {
		houseCode := ""
		if len(fields) > 1 {
			houseCode = fields[1]
		}
		return &ParsedCommand{
			Type:        CommandUtilityBatch,
			UtilityType: utilityType,
			HouseCode:   houseCode,
			Entries:     parseBatchEntries(lines[1:]),
		}
	}

	entry := RoomUtilityEntry{HasNewIndex: false}
	if len(fields) > 1 {
		newIndex, err := strconv.Atoi(fields[1])
		if err != nil {
			return &ParsedCommand{Type: CommandUtilityBatch, UtilityType: utilityType, HouseCode: fields[1]}
		}
		entry.NewIndex = newIndex
		entry.HasNewIndex = true
	}

	return &ParsedCommand{
		Type:        CommandUtilitySingle,
		UtilityType: utilityType,
		Entries:     []RoomUtilityEntry{entry},
	}
}

// NormalizeRoomName strips Vietnamese diacritics, separators, and common room prefixes for matching.
func NormalizeRoomName(name string) string {
	cleaned := normalizeVietnamese(strings.ToLower(strings.TrimSpace(name)))
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "")
	cleaned = replacer.Replace(cleaned)
	cleaned = strings.TrimPrefix(cleaned, "phong")
	cleaned = strings.TrimPrefix(cleaned, "p")
	return cleaned
}

// MatchRoom finds the best-matching room from a list by normalized name comparison.
func MatchRoom(input string, rooms []model.Room) (*model.Room, error) {
	normalizedInput := NormalizeRoomName(input)
	if normalizedInput == "" {
		return nil, model.ErrRoomNotFound
	}

	var matched *model.Room
	for i := range rooms {
		if NormalizeRoomName(rooms[i].Name) != normalizedInput {
			continue
		}
		if matched != nil {
			return nil, ErrAmbiguousRoomName
		}
		room := rooms[i]
		matched = &room
	}

	if matched == nil {
		return nil, model.ErrRoomNotFound
	}
	return matched, nil
}

// normalizeVietnamese removes accents while preserving ASCII command syntax.
func normalizeVietnamese(value string) string {
	value = strings.ReplaceAll(value, "đ", "d")
	value = strings.ReplaceAll(value, "Đ", "D")

	decomposed := norm.NFD.String(value)
	var builder strings.Builder
	builder.Grow(len(decomposed))
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		builder.WriteRune(r)
	}
	return norm.NFC.String(builder.String())
}

// utilityTypeFromToken returns the supported utility keyword for a command token.
func utilityTypeFromToken(token string) string {
	switch strings.TrimSpace(token) {
	case "#dien":
		return "dien"
	case "#nuoc":
		return "nuoc"
	default:
		return ""
	}
}

// compactNonEmptyLines returns trimmed non-empty message lines.
func compactNonEmptyLines(text string) []string {
	rawLines := strings.Split(text, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// parseBatchEntries parses manager multi-room utility readings from message lines.
func parseBatchEntries(lines []string) []RoomUtilityEntry {
	entries := make([]RoomUtilityEntry, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		newIndex, err := strconv.Atoi(fields[len(fields)-1])
		if err != nil {
			continue
		}
		entries = append(entries, RoomUtilityEntry{
			RoomName:    strings.Join(fields[:len(fields)-1], " "),
			NewIndex:    newIndex,
			HasNewIndex: true,
		})
	}
	return entries
}
