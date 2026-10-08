package zalo

import (
	"strings"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/pkg/zalobot"
)

// TestIsInvoiceCommandText verifies command detection for known and unknown messages.
func TestIsInvoiceCommandText(t *testing.T) {
	if !isInvoiceCommandText("#dien 100") {
		t.Error("expected #dien command to be recognized")
	}
	if isInvoiceCommandText("hello world") {
		t.Error("expected plain text to be unrecognized")
	}
}

// TestOppositeUtility verifies the paired utility lookup.
func TestOppositeUtility(t *testing.T) {
	if got := oppositeUtility("dien"); got != "nuoc" {
		t.Errorf("expected nuoc, got %s", got)
	}
	if got := oppositeUtility("nuoc"); got != "dien" {
		t.Errorf("expected dien, got %s", got)
	}
}

// TestAddMonthToPeriod covers valid advancement and invalid period parsing.
func TestAddMonthToPeriod(t *testing.T) {
	next, err := addMonthToPeriod("2026-01")
	if err != nil || next != "2026-02" {
		t.Errorf("expected 2026-02, got %s err %v", next, err)
	}
	next, err = addMonthToPeriod("2026-12")
	if err != nil || next != "2027-01" {
		t.Errorf("expected 2027-01, got %s err %v", next, err)
	}
	if _, err := addMonthToPeriod("bad"); err == nil {
		t.Error("expected error for invalid period")
	}
}

// TestOverwriteConfirmMessage verifies the overwrite prompt includes both values.
func TestOverwriteConfirmMessage(t *testing.T) {
	msg := overwriteConfirmMessage("2026-05", "dien", 100, 200)
	if msg == "" {
		t.Fatal("expected non-empty message")
	}
	for _, want := range []string{"điện", "05/2026", "100", "200"} {
		if !contains(msg, want) {
			t.Errorf("message %q missing %q", msg, want)
		}
	}
}

// TestOnlyPeriodFromPendingOptions covers single option maps and non-single cases.
func TestOnlyPeriodFromPendingOptions(t *testing.T) {
	if got := onlyPeriodFromPendingOptions(map[string]any{"period_options": map[string]any{"5": "2026-05"}}); got != "2026-05" {
		t.Errorf("expected 2026-05 from any-map, got %s", got)
	}
	if got := onlyPeriodFromPendingOptions(map[string]any{"period_options": map[string]string{"6": "2026-06"}}); got != "2026-06" {
		t.Errorf("expected 2026-06 from string-map, got %s", got)
	}
	if got := onlyPeriodFromPendingOptions(map[string]any{"period_options": map[string]any{"5": "2026-05", "6": "2026-06"}}); got != "" {
		t.Errorf("expected empty for multi option, got %s", got)
	}
	if got := onlyPeriodFromPendingOptions(map[string]any{}); got != "" {
		t.Errorf("expected empty for missing options, got %s", got)
	}
	// non-string value should be ignored
	if got := onlyPeriodFromPendingOptions(map[string]any{"period_options": map[string]any{"5": 123}}); got != "" {
		t.Errorf("expected empty for non-string value, got %s", got)
	}
}

// TestPeriodFromPendingOptions covers any-map, string-map, and missing lookups.
func TestPeriodFromPendingOptions(t *testing.T) {
	if got := periodFromPendingOptions(map[string]any{"period_options": map[string]any{"5": "2026-05"}}, 5); got != "2026-05" {
		t.Errorf("expected 2026-05, got %s", got)
	}
	if got := periodFromPendingOptions(map[string]any{"period_options": map[string]string{"6": "2026-06"}}, 6); got != "2026-06" {
		t.Errorf("expected 2026-06, got %s", got)
	}
	if got := periodFromPendingOptions(map[string]any{"period_options": map[string]string{"6": "2026-06"}}, 7); got != "" {
		t.Errorf("expected empty for missing month, got %s", got)
	}
	if got := periodFromPendingOptions(map[string]any{}, 5); got != "" {
		t.Errorf("expected empty for missing options, got %s", got)
	}
}

// TestIntFromPending covers each supported numeric encoding and the default.
func TestIntFromPending(t *testing.T) {
	cases := map[string]struct {
		data map[string]any
		want int
	}{
		"int":     {map[string]any{"v": int(5)}, 5},
		"int64":   {map[string]any{"v": int64(6)}, 6},
		"float64": {map[string]any{"v": float64(7)}, 7},
		"jsonNum": {map[string]any{"v": jsonNumber("8")}, 8},
		"badJson": {map[string]any{"v": jsonNumber("nan")}, 0},
		"default": {map[string]any{"v": "str"}, 0},
		"missing": {map[string]any{}, 0},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := intFromPending(tc.data, "v"); got != tc.want {
				t.Errorf("expected %d, got %d", tc.want, got)
			}
		})
	}
}

// TestEntriesFromPending covers well-formed entries and malformed inputs.
func TestEntriesFromPending(t *testing.T) {
	if got := entriesFromPending(map[string]any{}); got != nil {
		t.Errorf("expected nil for missing entries, got %v", got)
	}
	if got := entriesFromPending(map[string]any{"entries": "bad"}); got != nil {
		t.Errorf("expected nil for non-slice entries, got %v", got)
	}
	data := map[string]any{
		"entries": []any{
			"not a map",
			map[string]any{
				"room_name":     "P101",
				"new_index":     float64(250),
				"has_new_index": true,
			},
		},
	}
	entries := entriesFromPending(data)
	if len(entries) != 1 {
		t.Fatalf("expected 1 valid entry, got %d", len(entries))
	}
	if entries[0].RoomName != "P101" || entries[0].NewIndex != 250 || !entries[0].HasNewIndex {
		t.Errorf("unexpected entry: %+v", entries[0])
	}
}

// TestPeriodSelectionMessage covers single and multiple option prompts.
func TestPeriodSelectionMessage(t *testing.T) {
	single := periodSelectionMessage("P101", map[string]string{"5": "2026-05"})
	if !contains(single, "05/2026") || !contains(single, "#ok") {
		t.Errorf("unexpected single-option message: %q", single)
	}
	multi := periodSelectionMessage("P101", map[string]string{"5": "2026-05", "6": "2026-06"})
	if !contains(multi, "#5") || !contains(multi, "#6") {
		t.Errorf("unexpected multi-option message: %q", multi)
	}
}

// TestDisplayPeriod covers valid and invalid period strings.
func TestDisplayPeriod(t *testing.T) {
	if got := displayPeriod("2026-05"); got != "05/2026" {
		t.Errorf("expected 05/2026, got %s", got)
	}
	if got := displayPeriod("weird"); got != "weird" {
		t.Errorf("expected passthrough for invalid, got %s", got)
	}
}

// TestAwaitUtilityPromptMessage covers messages with and without a saved period.
func TestAwaitUtilityPromptMessage(t *testing.T) {
	noPeriod := awaitUtilityPromptMessage("dien", "")
	if !contains(noPeriod, "#dien") || contains(noPeriod, "tháng") {
		t.Errorf("unexpected no-period message: %q", noPeriod)
	}
	withPeriod := awaitUtilityPromptMessage("nuoc", "2026-05")
	if !contains(withPeriod, "05/2026") || !contains(withPeriod, "#nuoc") {
		t.Errorf("unexpected period message: %q", withPeriod)
	}
}

// TestCommandChatID covers reply, chat, and sender precedence.
func TestCommandChatID(t *testing.T) {
	if got := commandChatID(zalobot.Update{ReplyChatID: "reply", ChatID: "chat", SenderID: "sender"}); got != "reply" {
		t.Errorf("expected reply, got %s", got)
	}
	if got := commandChatID(zalobot.Update{ChatID: "chat", SenderID: "sender"}); got != "chat" {
		t.Errorf("expected chat, got %s", got)
	}
	if got := commandChatID(zalobot.Update{SenderID: "sender"}); got != "sender" {
		t.Errorf("expected sender, got %s", got)
	}
}

// TestExistingUtilityValues covers electricity and water index accessors.
func TestExistingUtilityValues(t *testing.T) {
	inv := &model.Invoice{
		NewElectricityIndex: 150,
		OldElectricityIndex: 100,
		NewWaterIndex:       60,
		OldWaterIndex:       40,
	}
	if existingUtilityValue(inv, "dien") != 150 {
		t.Error("expected new electricity 150")
	}
	if existingUtilityValue(inv, "nuoc") != 60 {
		t.Error("expected new water 60")
	}
	if existingOldUtilityValue(inv, "dien") != 100 {
		t.Error("expected old electricity 100")
	}
	if existingOldUtilityValue(inv, "nuoc") != 40 {
		t.Error("expected old water 40")
	}
}

// TestUtilityLabelFromMissing verifies the entered label is the opposite of the missing one.
func TestUtilityLabelFromMissing(t *testing.T) {
	if got := utilityLabelFromMissing(utilityUpdateResult{MissingUtility: "dien"}); got != "nước" {
		t.Errorf("expected nước, got %s", got)
	}
	if got := utilityLabelFromMissing(utilityUpdateResult{MissingUtility: "nuoc"}); got != "điện" {
		t.Errorf("expected điện, got %s", got)
	}
}

// contains is a small substring helper for message assertions.
func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
