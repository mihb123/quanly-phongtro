package plugin

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

const (
	hostModule        = "qlpt"
	wasmMemoryPages   = 4096
	wasmTimeout       = 30 * time.Second
	wasmConcurrency   = 4
	maxRequestBody    = 25 << 20
	maxCallPayload    = 32 << 20
	maxStorageValue   = 256 << 10
	maxPluginOutput   = 8 << 10
	storageKeyPrefix  = "$data:"
	maxStorageKeySize = 100
)

var (
	errPluginClosed  = errors.New("plugin closed")
	errPluginTimeout = errors.New("plugin timed out")
	forwardedHeaders = []string{"Content-Type", "Accept", "Accept-Language"}
	allowedImports   = []string{wasi_snapshot_preview1.ModuleName, hostModule}
)

func newWasmRuntime(ctx context.Context) (wazero.Runtime, error) {
	config := wazero.NewRuntimeConfig().WithCloseOnContextDone(true).WithMemoryLimitPages(wasmMemoryPages)
	if dir, err := os.UserCacheDir(); err == nil {
		if cache, err := wazero.NewCompilationCacheWithDir(filepath.Join(dir, "quanly-phongtro", "wasm")); err == nil {
			config = config.WithCompilationCache(cache)
		}
	}
	rt := wazero.NewRuntimeWithConfig(ctx, config)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		return nil, err
	}
	_, err := rt.NewHostModuleBuilder(hostModule).
		NewFunctionBuilder().WithFunc(hostCall).Export("call").
		NewFunctionBuilder().WithFunc(hostResult).Export("result").
		Instantiate(ctx)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func compileBackend(ctx context.Context, rt wazero.Runtime, code []byte) (wazero.CompiledModule, error) {
	compiled, err := rt.CompileModule(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("%s không phải WebAssembly hợp lệ: %v", backendFile, err)
	}
	for _, fn := range compiled.ImportedFunctions() {
		module, name, _ := fn.Import()
		if !slices.Contains(allowedImports, module) {
			_ = compiled.Close(ctx)
			return nil, fmt.Errorf("%s import %s.%s không được hỗ trợ (chỉ wasi_snapshot_preview1 và %s)", backendFile, module, name, hostModule)
		}
	}
	if _, ok := compiled.ExportedFunctions()["_start"]; !ok {
		_ = compiled.Close(ctx)
		return nil, fmt.Errorf("%s thiếu hàm _start (build bằng GOOS=wasip1 GOARCH=wasm go build)", backendFile)
	}
	return compiled, nil
}

type installedPlugin struct {
	manifest Manifest
	dir      string
	hasWeb   bool
	hasStyle bool
	runtime  wazero.Runtime
	slots    chan struct{}

	mu       sync.RWMutex
	compiled wazero.CompiledModule
	closed   bool
	host     Host
}

func (p *installedPlugin) Manifest() Manifest { return p.manifest }

func (p *installedPlugin) Init(host Host) error {
	p.host = host
	return nil
}

func (p *installedPlugin) Routes(r chi.Router) {
	r.HandleFunc("/*", p.serve)
}

func (p *installedPlugin) close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	if p.compiled != nil {
		_ = p.compiled.Close(context.Background())
	}
}

func (p *installedPlugin) hasPermission(permission string) bool {
	return slices.Contains(p.manifest.Permissions, permission)
}

type wasmRequest struct {
	Method    string              `json:"method"`
	Path      string              `json:"path"`
	Query     url.Values          `json:"query"`
	Header    map[string][]string `json:"header"`
	Body      []byte              `json:"body"`
	ManagerID string              `json:"manager_id"`
}

type wasmResponse struct {
	Status      int             `json:"status"`
	Data        json.RawMessage `json:"data,omitempty"`
	Message     string          `json:"message,omitempty"`
	ContentType string          `json:"content_type,omitempty"`
	Filename    string          `json:"filename,omitempty"`
	Body        []byte          `json:"body,omitempty"`
}

func (p *installedPlugin) serve(w http.ResponseWriter, r *http.Request) {
	managerID, ok := ManagerID(w, r)
	if !ok {
		return
	}
	if p.compiled == nil {
		WriteError(w, http.StatusNotFound, "Tiện ích không có API")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBody))
	if err != nil {
		WriteError(w, http.StatusRequestEntityTooLarge, "Dữ liệu gửi lên quá lớn")
		return
	}
	header := map[string][]string{}
	for _, name := range forwardedHeaders {
		if values := r.Header.Values(name); len(values) > 0 {
			header[name] = values
		}
	}
	resp, err := p.invoke(r.Context(), managerID, wasmRequest{
		Method:    r.Method,
		Path:      "/" + chi.URLParam(r, "*"),
		Query:     r.URL.Query(),
		Header:    header,
		Body:      body,
		ManagerID: managerID,
	})
	switch {
	case errors.Is(err, errPluginClosed):
		WriteError(w, http.StatusServiceUnavailable, "Tiện ích vừa được cập nhật, vui lòng thử lại")
		return
	case errors.Is(err, errPluginTimeout):
		WriteError(w, http.StatusGatewayTimeout, "Tiện ích xử lý quá lâu")
		return
	case err != nil:
		log.Printf("plugin %s: %v", p.manifest.ID, err)
		WriteError(w, http.StatusInternalServerError, "Tiện ích gặp lỗi")
		return
	}
	writeWasmResponse(w, resp)
}

func writeWasmResponse(w http.ResponseWriter, resp *wasmResponse) {
	status := resp.Status
	if status < 200 || status > 599 {
		status = http.StatusOK
	}
	if resp.Body == nil && resp.ContentType == "" {
		var data any
		if len(resp.Data) > 0 {
			data = resp.Data
		}
		WriteJSON(w, status, data, resp.Message)
		return
	}
	contentType := resp.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	filename := resp.Filename
	if filename == "" {
		filename = "download"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox")
	w.WriteHeader(status)
	_, _ = w.Write(resp.Body)
}

type invocationKey struct{}

type invocation struct {
	plugin    *installedPlugin
	managerID string
	request   []byte
	response  *wasmResponse
	result    []byte
}

func (p *installedPlugin) invoke(ctx context.Context, managerID string, req wasmRequest) (*wasmResponse, error) {
	select {
	case p.slots <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-p.slots }()

	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return nil, errPluginClosed
	}

	raw, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	call := &invocation{plugin: p, managerID: managerID, request: raw}
	ctx, cancel := context.WithTimeout(ctx, wasmTimeout)
	defer cancel()
	ctx = context.WithValue(ctx, invocationKey{}, call)

	output := &cappedBuffer{limit: maxPluginOutput}
	config := wazero.NewModuleConfig().
		WithName("").
		WithArgs(p.manifest.ID).
		WithStdout(output).
		WithStderr(output).
		WithSysWalltime().
		WithSysNanotime().
		WithRandSource(rand.Reader)
	mod, err := p.runtime.InstantiateModule(ctx, p.compiled, config)
	if mod != nil {
		_ = mod.Close(context.Background())
	}
	if out := strings.TrimSpace(output.String()); out != "" {
		log.Printf("plugin %s output: %s", p.manifest.ID, out)
	}
	var exitErr *sys.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 0 {
		err = nil
	}
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, errPluginTimeout
		}
		return nil, fmt.Errorf("run: %w", err)
	}
	if call.response == nil {
		return nil, errors.New("plugin exited without a response")
	}
	return call.response, nil
}

func hostCall(ctx context.Context, m api.Module, ptr, size uint32) uint32 {
	call, ok := ctx.Value(invocationKey{}).(*invocation)
	if !ok {
		return 0
	}
	if size > maxCallPayload {
		call.result = callError(http.StatusRequestEntityTooLarge, "payload quá lớn")
		return uint32(len(call.result))
	}
	payload, ok := m.Memory().Read(ptr, size)
	if !ok {
		panic("qlpt.call: out of memory range")
	}
	call.result = call.handle(ctx, append([]byte(nil), payload...))
	return uint32(len(call.result))
}

func hostResult(ctx context.Context, m api.Module, ptr uint32) {
	call, ok := ctx.Value(invocationKey{}).(*invocation)
	if !ok {
		return
	}
	if !m.Memory().Write(ptr, call.result) {
		panic("qlpt.result: out of memory range")
	}
	call.result = nil
}

type statusError struct {
	status  int
	message string
}

func (e *statusError) Error() string { return e.message }

func fail(status int, message string) error { return &statusError{status: status, message: message} }

func callError(status int, message string) []byte {
	raw, _ := json.Marshal(map[string]any{"ok": false, "status": status, "error": message})
	return raw
}

func (c *invocation) handle(ctx context.Context, payload []byte) []byte {
	var op struct {
		Op     string          `json:"op"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(payload, &op); err != nil {
		return callError(http.StatusBadRequest, "lời gọi host không hợp lệ")
	}
	data, err := c.dispatch(ctx, op.Op, op.Params)
	if err != nil {
		var failure *statusError
		if errors.As(err, &failure) {
			return callError(failure.status, failure.message)
		}
		log.Printf("plugin %s: host %s: %v", c.plugin.manifest.ID, op.Op, err)
		return callError(http.StatusInternalServerError, "lỗi hệ thống khi xử lý "+op.Op)
	}
	if raw, ok := data.(json.RawMessage); ok {
		return append(append([]byte(`{"ok":true,"data":`), raw...), '}')
	}
	raw, err := json.Marshal(map[string]any{"ok": true, "data": data})
	if err != nil {
		return callError(http.StatusInternalServerError, "không mã hoá được kết quả")
	}
	return raw
}

func decodeParams(raw json.RawMessage, v any) error {
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return fail(http.StatusBadRequest, "tham số không hợp lệ: "+err.Error())
	}
	return nil
}

func (c *invocation) require(permission string) error {
	if !c.plugin.hasPermission(permission) {
		return fail(http.StatusForbidden, fmt.Sprintf("plugin chưa khai báo quyền %q trong manifest", permission))
	}
	return nil
}

func (c *invocation) dispatch(ctx context.Context, op string, params json.RawMessage) (any, error) {
	host := c.plugin.host
	switch op {
	case "request":
		return json.RawMessage(c.request), nil
	case "respond":
		if c.response != nil {
			return nil, fail(http.StatusConflict, "đã trả response")
		}
		var resp wasmResponse
		if err := decodeParams(params, &resp); err != nil {
			return nil, err
		}
		c.response = &resp
		return nil, nil
	case "config.get":
		values, err := host.Config().Get(ctx, c.managerID)
		if err != nil {
			return nil, err
		}
		settings := Values{}
		for _, f := range c.plugin.manifest.Settings {
			if v, ok := values[f.Key]; ok {
				settings[f.Key] = v
			}
		}
		return settings, nil
	case "storage.get":
		var p struct {
			Key string `json:"key"`
		}
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if err := validStorageKey(p.Key); err != nil {
			return nil, err
		}
		values, err := host.Config().Get(ctx, c.managerID)
		if err != nil {
			return nil, err
		}
		return values[storageKeyPrefix+p.Key], nil
	case "storage.set":
		var p struct {
			Key   string          `json:"key"`
			Value json.RawMessage `json:"value"`
		}
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if err := validStorageKey(p.Key); err != nil {
			return nil, err
		}
		if len(p.Value) > maxStorageValue {
			return nil, fail(http.StatusRequestEntityTooLarge, fmt.Sprintf("giá trị storage tối đa %d KB", maxStorageValue>>10))
		}
		var value any
		if len(p.Value) > 0 {
			if err := json.Unmarshal(p.Value, &value); err != nil {
				return nil, fail(http.StatusBadRequest, "giá trị storage không hợp lệ")
			}
		}
		return nil, host.Config().Set(ctx, c.managerID, map[string]any{storageKeyPrefix + p.Key: value})
	case "invoices.list_outstanding", "invoices.list_paid":
		if err := c.require(PermInvoicesRead); err != nil {
			return nil, err
		}
		var p struct {
			Periods []string `json:"periods"`
		}
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		var (
			invoices []model.OutstandingInvoice
			err      error
		)
		if op == "invoices.list_paid" {
			invoices, err = host.Invoices().ListPaid(ctx, c.managerID, p.Periods)
		} else {
			invoices, err = host.Invoices().ListOutstanding(ctx, c.managerID, p.Periods)
		}
		if invoices == nil {
			invoices = []model.OutstandingInvoice{}
		}
		return invoices, err
	case "invoices.recorded_payment":
		if err := c.require(PermInvoicesRead); err != nil {
			return nil, err
		}
		var p struct {
			AccountRef     string `json:"account_ref"`
			TransactionRef string `json:"transaction_ref"`
		}
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		invoiceID, recorded, err := host.Invoices().RecordedPayment(ctx, c.plugin.manifest.ID, p.AccountRef, p.TransactionRef)
		return map[string]any{"invoice_id": invoiceID, "recorded": recorded}, err
	case "invoices.mark_paid":
		if err := c.require(PermInvoicesWrite); err != nil {
			return nil, err
		}
		var p struct {
			InvoiceID      string `json:"invoice_id"`
			Amount         int    `json:"amount"`
			AccountRef     string `json:"account_ref"`
			TransactionRef string `json:"transaction_ref"`
			MatchingMethod string `json:"matching_method"`
			PaymentMethod  string `json:"payment_method"`
			RawPayload     string `json:"raw_payload"`
		}
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.InvoiceID == "" || p.TransactionRef == "" || p.Amount < 0 {
			return nil, fail(http.StatusBadRequest, "cần invoice_id, transaction_ref và amount >= 0")
		}
		if p.PaymentMethod == "" {
			p.PaymentMethod = "PLUGIN"
		}
		if len(p.PaymentMethod) > 50 || len(p.MatchingMethod) > 50 {
			return nil, fail(http.StatusBadRequest, "payment_method/matching_method tối đa 50 ký tự")
		}
		err := host.Invoices().MarkPaid(ctx, c.managerID, model.PluginPayment{
			Source:               c.plugin.manifest.ID,
			InvoiceID:            p.InvoiceID,
			Amount:               p.Amount,
			AccountRef:           p.AccountRef,
			TransactionReference: p.TransactionRef,
			MatchingMethod:       p.MatchingMethod,
			PaymentMethod:        p.PaymentMethod,
			RawPayload:           p.RawPayload,
		})
		if errors.Is(err, model.ErrPluginPaymentRecorded) || errors.Is(err, model.ErrPluginInvoiceNotPayable) {
			return nil, fail(http.StatusConflict, err.Error())
		}
		return nil, err
	}
	return nil, fail(http.StatusNotImplemented, fmt.Sprintf("host không hỗ trợ %q ở API %d", op, APIVersion))
}

func validStorageKey(key string) error {
	if key == "" || len(key) > maxStorageKeySize {
		return fail(http.StatusBadRequest, fmt.Sprintf("key storage bắt buộc, tối đa %d ký tự", maxStorageKeySize))
	}
	return nil
}

type cappedBuffer struct {
	limit int
	buf   []byte
	mu    sync.Mutex
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if room := b.limit - len(b.buf); room > 0 {
		b.buf = append(b.buf, p[:min(room, len(p))]...)
	}
	return len(p), nil
}

func (b *cappedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}
