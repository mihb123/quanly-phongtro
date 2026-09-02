package zalo

import (
	"errors"
	"regexp"
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
	// CommandUtilityRoom is a one-line reading that names its room, e.g. "#dien 679qt P201 661".
	CommandUtilityRoom
	CommandHelp
	// CommandUpdateTenantPhone and CommandUpdateRoomGroup are manager-only edits sent from chat.
	CommandUpdateTenantPhone
	CommandUpdateRoomGroup
)

type ParsedCommand struct {
	Type        CommandType
	UtilityType string
	HouseCode   string
	Entries     []RoomUtilityEntry
	PeriodMonth int
	// RoomName and Value carry the target and the new value of a manager update command.
	RoomName string
	Value    string
}

type RoomUtilityEntry struct {
	RoomName    string
	NewIndex    int
	HasNewIndex bool
}

var ErrAmbiguousRoomName = errors.New("ambiguous room name")

var (
	vietnamesePhonePattern = regexp.MustCompile(`^(?:\+84|84|0)[0-9]{9}$`)
	zaloGroupChatIDPattern = regexp.MustCompile(`^[0-9]{6,32}$`)
)

// ParseCommand parses a raw chat message into a structured invoice command.
func ParseCommand(text string) *ParsedCommand {
	trimmed := strings.TrimSpace(stripMentionPrefix(text))
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
		case "help", "trogiup":
			return &ParsedCommand{Type: CommandHelp}
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

	if command := parseManagerUpdateCommand(fields); command != nil {
		return command
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

	return parseSingleLineCommand(utilityType, fields[1:])
}

// parseSingleLineCommand reads the arguments of a one-line utility command. The reading is always
// the last token, so anything before it names the target: "<room>" alone when the sender manages a
// single house, or "<house code> <room>" when the house must be spelled out. Without a trailing
// reading the message is the header of a multi-room batch ("#dien 679qt").
func parseSingleLineCommand(utilityType string, args []string) *ParsedCommand {
	if len(args) == 0 {
		return &ParsedCommand{
			Type:        CommandUtilitySingle,
			UtilityType: utilityType,
			Entries:     []RoomUtilityEntry{{}},
		}
	}

	newIndex, err := strconv.Atoi(args[len(args)-1])
	if err != nil {
		return &ParsedCommand{Type: CommandUtilityBatch, UtilityType: utilityType, HouseCode: args[0]}
	}

	entry := RoomUtilityEntry{NewIndex: newIndex, HasNewIndex: true}
	target := args[:len(args)-1]
	houseCode := ""
	switch len(target) {
	case 0:
		return &ParsedCommand{
			Type:        CommandUtilitySingle,
			UtilityType: utilityType,
			Entries:     []RoomUtilityEntry{entry},
		}
	case 1:
		entry.RoomName = target[0]
	default:
		houseCode = target[0]
		entry.RoomName = strings.Join(target[1:], " ")
	}

	return &ParsedCommand{
		Type:        CommandUtilityRoom,
		UtilityType: utilityType,
		HouseCode:   houseCode,
		Entries:     []RoomUtilityEntry{entry},
	}
}

// parseManagerUpdateCommand parses the manager edit commands, or returns nil for anything else.
// Both take an optional room target followed by the new value: "#update-tenant 679qt P201 0912345678".
func parseManagerUpdateCommand(fields []string) *ParsedCommand {
	commandType := managerUpdateTypeFromToken(fields[0])
	if commandType == CommandUnknown {
		return nil
	}

	value, target := splitManagerUpdateValue(commandType, fields[1:])
	command := &ParsedCommand{Type: commandType, Value: value}
	switch len(target) {
	case 0:
	case 1:
		command.RoomName = target[0]
	default:
		command.HouseCode = target[0]
		command.RoomName = strings.Join(target[1:], " ")
	}
	return command
}

// managerUpdateTypeFromToken maps a command token to its manager update command type.
func managerUpdateTypeFromToken(token string) CommandType {
	switch strings.TrimSpace(token) {
	case "#update-tenant", "#update_tenant", "#updatetenant":
		return CommandUpdateTenantPhone
	case "#update-room", "#update_room", "#updateroom":
		return CommandUpdateRoomGroup
	default:
		return CommandUnknown
	}
}

// splitManagerUpdateValue separates the new value from the room target. A phone number typed with
// spaces is joined back together; a missing group chat ID is left empty so the handler can fall back
// to the current group chat.
func splitManagerUpdateValue(commandType CommandType, args []string) (string, []string) {
	if len(args) == 0 {
		return "", nil
	}
	last := args[len(args)-1]

	if commandType == CommandUpdateTenantPhone {
		if phone, ok := NormalizePhone(last); ok {
			return phone, args[:len(args)-1]
		}
		if phone, ok := NormalizePhone(strings.Join(args, "")); ok {
			return phone, nil
		}
		// Without a readable phone the command cannot run, so the target is left out as well.
		return "", nil
	}

	if IsZaloGroupChatID(last) {
		return last, args[:len(args)-1]
	}
	return "", args
}

// NormalizePhone returns the local 0-prefixed form of a Vietnamese mobile number, or false when the
// text is not one.
func NormalizePhone(text string) (string, bool) {
	compact := strings.Map(func(r rune) rune {
		if r == ' ' || r == '.' || r == '-' {
			return -1
		}
		return r
	}, text)
	if !vietnamesePhonePattern.MatchString(compact) {
		return "", false
	}
	switch {
	case strings.HasPrefix(compact, "+84"):
		return "0" + compact[3:], true
	case strings.HasPrefix(compact, "84"):
		return "0" + compact[2:], true
	default:
		return compact, true
	}
}

// IsZaloGroupChatID reports whether a token is a Zalo group chat ID as the bot prints it. Group IDs
// are long digit strings, which keeps them apart from the short room names in the same position.
func IsZaloGroupChatID(value string) bool {
	return zaloGroupChatIDPattern.MatchString(value)
}

// stripMentionPrefix removes any leading text before the first '#' command marker. In group
// chats the Zalo Bot API prepends the bot's @mention to the message text (e.g. "@Bot #dien 50"),
// which would otherwise shift the command token and make the message parse as unknown.
func stripMentionPrefix(text string) string {
	idx := strings.Index(text, "#")
	if idx <= 0 {
		return text
	}
	return text[idx:]
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
