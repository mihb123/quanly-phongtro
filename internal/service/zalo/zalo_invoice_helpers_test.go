package zalo

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"strings"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/model"
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
	if got := commandChatID(webhookMessageContext{replyChatID: "reply", chatID: "chat", senderID: "sender"}); got != "reply" {
		t.Errorf("expected reply, got %s", got)
	}
	if got := commandChatID(webhookMessageContext{chatID: "chat", senderID: "sender"}); got != "chat" {
		t.Errorf("expected chat, got %s", got)
	}
	if got := commandChatID(webhookMessageContext{senderID: "sender"}); got != "sender" {
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

// TestIsAllowedImageContentType covers accepted, rejected, and malformed content types.
func TestIsAllowedImageContentType(t *testing.T) {
	for _, ct := range []string{"image/jpeg", "image/png; charset=utf-8", "IMAGE/GIF", "image/webp"} {
		if !isAllowedImageContentType(ct) {
			t.Errorf("expected %q allowed", ct)
		}
	}
	for _, ct := range []string{"text/plain", "application/json", "", "!!!bad"} {
		if isAllowedImageContentType(ct) {
			t.Errorf("expected %q rejected", ct)
		}
	}
}

// TestDefaultResolveURLHost covers IP literal parsing.
func TestDefaultResolveURLHost(t *testing.T) {
	addrs, err := defaultResolveURLHost(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(addrs) != 1 || addrs[0].String() != "8.8.8.8" {
		t.Errorf("unexpected addresses: %v", addrs)
	}

	addrs, err = defaultResolveURLHost(context.Background(), "localhost")
	if err != nil {
		t.Fatalf("unexpected localhost resolve error: %v", err)
	}
	if len(addrs) == 0 {
		t.Error("expected localhost to resolve at least one address")
	}

	if _, err := defaultResolveURLHost(context.Background(), "invalid host name"); err == nil {
		t.Error("expected resolver error for malformed host")
	}
}

// TestValidateZaloTransactionImageURL_Errors covers parse, scheme, host, and resolver failures.
func TestValidateZaloTransactionImageURL_Errors(t *testing.T) {
	ctx := context.Background()

	if err := validateZaloTransactionImageURL(ctx, "https://%zz"); err == nil {
		t.Error("expected parse error")
	}
	if err := validateZaloTransactionImageURL(ctx, "http://host/x"); err == nil {
		t.Error("expected scheme error")
	}
	if err := validateZaloTransactionImageURL(ctx, "https:///path"); err == nil {
		t.Error("expected empty host error")
	}

	previous := resolveZaloImageHost
	resolveZaloImageHost = func(context.Context, string) ([]netip.Addr, error) {
		return nil, errors.New("resolve failed")
	}
	defer func() { resolveZaloImageHost = previous }()
	if err := validateZaloTransactionImageURL(ctx, "https://zalo.test/x.jpg"); err == nil {
		t.Error("expected resolver error")
	}
}

// TestValidateZaloTransactionImageURL_PublicAllowed accepts a public routable target.
func TestValidateZaloTransactionImageURL_PublicAllowed(t *testing.T) {
	previous := resolveZaloImageHost
	resolveZaloImageHost = func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
	}
	defer func() { resolveZaloImageHost = previous }()
	if err := validateZaloTransactionImageURL(context.Background(), "https://zalo.test/x.jpg"); err != nil {
		t.Errorf("expected public target allowed, got %v", err)
	}
}

// TestDefaultZaloImageHTTPClient exercises the redirect guard branches.
func TestDefaultZaloImageHTTPClient(t *testing.T) {
	client := defaultZaloImageHTTPClient()
	if client.Timeout != zaloTransactionImageDownloadTimeout {
		t.Errorf("unexpected timeout: %v", client.Timeout)
	}

	mustReq := func(rawURL string) *http.Request {
		req, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			t.Fatalf("build request: %v", err)
		}
		return req
	}

	// Too many redirects.
	via := []*http.Request{mustReq("https://a.test/1"), mustReq("https://a.test/2"), mustReq("https://a.test/3")}
	if err := client.CheckRedirect(mustReq("https://a.test/4"), via); err == nil {
		t.Error("expected too-many-redirects error")
	}

	// Redirect to a different host.
	via = []*http.Request{mustReq("https://a.test/1")}
	if err := client.CheckRedirect(mustReq("https://b.test/2"), via); err == nil {
		t.Error("expected cross-host redirect error")
	}

	// Same-host redirect that passes host validation via stubbed resolver.
	previous := resolveZaloImageHost
	resolveZaloImageHost = func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
	}
	defer func() { resolveZaloImageHost = previous }()
	via = []*http.Request{mustReq("https://a.test/1")}
	if err := client.CheckRedirect(mustReq("https://a.test/2"), via); err != nil {
		t.Errorf("expected allowed same-host redirect, got %v", err)
	}
}

// contains is a small substring helper for message assertions.
func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
