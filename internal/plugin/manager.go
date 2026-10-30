package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

type Store interface {
	ListPluginSettings(ctx context.Context, managerID string) ([]model.PluginSetting, error)
	GetPluginSetting(ctx context.Context, managerID, pluginID string) (*model.PluginSetting, error)
	SetPluginEnabled(ctx context.Context, managerID, pluginID string, enabled bool) error
	SavePluginData(ctx context.Context, managerID, pluginID string, encryptedData *string) error
	ListOutstandingInvoices(ctx context.Context, managerID string, periods []string) ([]model.OutstandingInvoice, error)
	ListPaidInvoices(ctx context.Context, managerID string, periods []string) ([]model.OutstandingInvoice, error)
	RecordedPayment(ctx context.Context, source, accountRef, transactionRef string) (string, bool, error)
	SettleInvoice(ctx context.Context, managerID string, payment model.PluginPayment) error
}

type PluginInfo struct {
	Manifest
	Enabled bool `json:"enabled"`
}

type Manager struct {
	store   Store
	key     []byte
	plugins map[string]Plugin
	hosts   map[string]*host
	routers map[string]chi.Router
	order   []string
}

func NewManager(store Store, encryptionKey []byte, plugins []Plugin) *Manager {
	m := &Manager{store: store, key: encryptionKey, plugins: map[string]Plugin{}, hosts: map[string]*host{}, routers: map[string]chi.Router{}}
	for _, p := range plugins {
		manifest := p.Manifest()
		h := &host{store: store, key: encryptionKey, pluginID: manifest.ID, fields: manifest.Settings}
		if err := p.Init(h); err != nil {
			log.Printf("plugin %s: init failed, skipped: %v", manifest.ID, err)
			continue
		}
		router := chi.NewRouter()
		p.Routes(router)
		m.plugins[manifest.ID] = p
		m.hosts[manifest.ID] = h
		m.routers[manifest.ID] = router
		m.order = append(m.order, manifest.ID)
	}
	return m
}

func (m *Manager) Routes(r chi.Router) {
	r.Get("/", m.handleList)
	r.Put("/{pluginID}/enabled", m.handleSetEnabled)
	r.Get("/{pluginID}/config", m.handleGetConfig)
	r.Put("/{pluginID}/config", m.handleSetConfig)
	r.HandleFunc("/{pluginID}/*", m.handlePluginRequest)
}

func (m *Manager) List(ctx context.Context, managerID string) ([]PluginInfo, error) {
	settings, err := m.store.ListPluginSettings(ctx, managerID)
	if err != nil {
		return nil, err
	}
	enabled := map[string]bool{}
	for _, s := range settings {
		if s.Enabled != nil {
			enabled[s.PluginID] = *s.Enabled
		}
	}
	infos := make([]PluginInfo, 0, len(m.order))
	for _, id := range m.order {
		manifest := m.plugins[id].Manifest()
		state, ok := enabled[id]
		if !ok {
			state = manifest.DefaultEnabled
		}
		infos = append(infos, PluginInfo{Manifest: manifest, Enabled: state})
	}
	return infos, nil
}

func (m *Manager) isEnabled(ctx context.Context, managerID, pluginID string) (bool, error) {
	setting, err := m.store.GetPluginSetting(ctx, managerID, pluginID)
	if err != nil {
		return false, err
	}
	if setting == nil || setting.Enabled == nil {
		return m.plugins[pluginID].Manifest().DefaultEnabled, nil
	}
	return *setting.Enabled, nil
}

func (m *Manager) handleList(w http.ResponseWriter, r *http.Request) {
	managerID, ok := ManagerID(w, r)
	if !ok {
		return
	}
	infos, err := m.List(r.Context(), managerID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Không tải được danh sách plugin")
		return
	}
	WriteJSON(w, http.StatusOK, infos, "")
}

func (m *Manager) handleSetEnabled(w http.ResponseWriter, r *http.Request) {
	managerID, ok := ManagerID(w, r)
	if !ok {
		return
	}
	pluginID := chi.URLParam(r, "pluginID")
	if _, exists := m.plugins[pluginID]; !exists {
		WriteError(w, http.StatusNotFound, model.ErrPluginNotFound.Error())
		return
	}
	var body struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&body); err != nil || body.Enabled == nil {
		WriteError(w, http.StatusBadRequest, "Thiếu trạng thái bật/tắt")
		return
	}
	if err := m.store.SetPluginEnabled(r.Context(), managerID, pluginID, *body.Enabled); err != nil {
		WriteError(w, http.StatusInternalServerError, "Không lưu được trạng thái plugin")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]bool{"enabled": *body.Enabled}, "")
}

type configResponse struct {
	Values  map[string]any  `json:"values"`
	Secrets map[string]bool `json:"secrets"`
}

func (m *Manager) configHost(w http.ResponseWriter, r *http.Request) (*host, string, bool) {
	managerID, ok := ManagerID(w, r)
	if !ok {
		return nil, "", false
	}
	h, exists := m.hosts[chi.URLParam(r, "pluginID")]
	if !exists {
		WriteError(w, http.StatusNotFound, model.ErrPluginNotFound.Error())
		return nil, "", false
	}
	return h, managerID, true
}

func (m *Manager) writeConfig(w http.ResponseWriter, r *http.Request, h *host, managerID string) {
	values, err := h.Get(r.Context(), managerID)
	if err != nil {
		log.Printf("plugin %s: load config: %v", h.pluginID, err)
		WriteError(w, http.StatusInternalServerError, "Không đọc được cài đặt plugin")
		return
	}
	resp := configResponse{Values: map[string]any{}, Secrets: map[string]bool{}}
	for _, f := range h.fields {
		if f.Type == SettingPassword {
			resp.Secrets[f.Key] = values.String(f.Key) != ""
			continue
		}
		if v, ok := values[f.Key]; ok {
			resp.Values[f.Key] = v
		}
	}
	WriteJSON(w, http.StatusOK, resp, "")
}

func (m *Manager) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	h, managerID, ok := m.configHost(w, r)
	if !ok {
		return
	}
	m.writeConfig(w, r, h, managerID)
}

func (m *Manager) handleSetConfig(w http.ResponseWriter, r *http.Request) {
	h, managerID, ok := m.configHost(w, r)
	if !ok {
		return
	}
	var body struct {
		Values map[string]any `json:"values"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&body); err != nil || len(body.Values) == 0 {
		WriteError(w, http.StatusBadRequest, "Dữ liệu cài đặt không hợp lệ")
		return
	}
	updates := make(map[string]any, len(body.Values))
	for key, value := range body.Values {
		i := slices.IndexFunc(h.fields, func(f SettingField) bool { return f.Key == key })
		if i < 0 {
			WriteError(w, http.StatusBadRequest, fmt.Sprintf("Cài đặt %q không tồn tại", key))
			return
		}
		clean, err := validateSetting(h.fields[i], value)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		updates[key] = clean
	}
	if err := h.Set(r.Context(), managerID, updates); err != nil {
		log.Printf("plugin %s: save config: %v", h.pluginID, err)
		WriteError(w, http.StatusInternalServerError, "Không lưu được cài đặt plugin")
		return
	}
	m.writeConfig(w, r, h, managerID)
}

func validateSetting(f SettingField, value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	invalid := fmt.Errorf("%s: giá trị không hợp lệ", f.Label)
	switch f.Type {
	case SettingToggle:
		if b, ok := value.(bool); ok {
			return b, nil
		}
	case SettingText, SettingTextarea, SettingPassword:
		s, ok := value.(string)
		if !ok || utf8.RuneCountInString(s) > 4000 {
			return nil, invalid
		}
		if f.Type == SettingText {
			s = strings.TrimSpace(s)
		}
		if s == "" {
			return nil, nil
		}
		return s, nil
	case SettingNumber:
		n, ok := value.(float64)
		if !ok || math.IsNaN(n) || math.IsInf(n, 0) {
			return nil, invalid
		}
		if f.Min != nil && n < *f.Min {
			return nil, fmt.Errorf("%s: tối thiểu %v", f.Label, *f.Min)
		}
		if f.Max != nil && n > *f.Max {
			return nil, fmt.Errorf("%s: tối đa %v", f.Label, *f.Max)
		}
		return n, nil
	case SettingSelect:
		s, _ := value.(string)
		if slices.ContainsFunc(f.Options, func(o SettingOption) bool { return o.Value == s }) {
			return s, nil
		}
	}
	return nil, invalid
}

func (m *Manager) handlePluginRequest(w http.ResponseWriter, r *http.Request) {
	managerID, ok := ManagerID(w, r)
	if !ok {
		return
	}
	pluginID := chi.URLParam(r, "pluginID")
	router, exists := m.routers[pluginID]
	if !exists {
		WriteError(w, http.StatusNotFound, model.ErrPluginNotFound.Error())
		return
	}
	enabled, err := m.isEnabled(r.Context(), managerID, pluginID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Không kiểm tra được trạng thái plugin")
		return
	}
	if !enabled {
		WriteError(w, http.StatusForbidden, model.ErrPluginDisabled.Error())
		return
	}
	rctx := chi.RouteContext(r.Context())
	rctx.RoutePath = "/" + chi.URLParam(r, "*")
	router.ServeHTTP(w, r)
}

type host struct {
	store    Store
	key      []byte
	pluginID string
	fields   []SettingField
}

func (h *host) Invoices() InvoiceAPI { return invoiceAPI{store: h.store} }

func (h *host) Config() Config { return h }

func (h *host) load(ctx context.Context, managerID string) (map[string]any, error) {
	stored := map[string]any{}
	setting, err := h.store.GetPluginSetting(ctx, managerID, h.pluginID)
	if err != nil || setting == nil || setting.EncryptedData == nil || *setting.EncryptedData == "" {
		return stored, err
	}
	plain, err := security.Decrypt(*setting.EncryptedData, h.key)
	if err != nil {
		return nil, fmt.Errorf("decrypt plugin config: %w", err)
	}
	if err := json.Unmarshal([]byte(plain), &stored); err != nil {
		return nil, fmt.Errorf("decode plugin config: %w", err)
	}
	return stored, nil
}

func (h *host) Get(ctx context.Context, managerID string) (Values, error) {
	stored, err := h.load(ctx, managerID)
	if err != nil {
		return nil, err
	}
	values := Values{}
	for _, f := range h.fields {
		if f.Default == nil {
			continue
		}
		raw, _ := json.Marshal(f.Default)
		var v any
		if json.Unmarshal(raw, &v) == nil {
			values[f.Key] = v
		}
	}
	for k, v := range stored {
		values[k] = v
	}
	return values, nil
}

func (h *host) Set(ctx context.Context, managerID string, values map[string]any) error {
	stored, err := h.load(ctx, managerID)
	if err != nil {
		return err
	}
	for k, v := range values {
		if v == nil {
			delete(stored, k)
		} else {
			stored[k] = v
		}
	}
	if len(stored) == 0 {
		return h.store.SavePluginData(ctx, managerID, h.pluginID, nil)
	}
	raw, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	encrypted, err := security.Encrypt(string(raw), h.key)
	if err != nil {
		return fmt.Errorf("encrypt plugin config: %w", err)
	}
	return h.store.SavePluginData(ctx, managerID, h.pluginID, &encrypted)
}

type invoiceAPI struct {
	store Store
}

func (a invoiceAPI) ListOutstanding(ctx context.Context, managerID string, periods []string) ([]model.OutstandingInvoice, error) {
	return a.store.ListOutstandingInvoices(ctx, managerID, periods)
}

func (a invoiceAPI) ListPaid(ctx context.Context, managerID string, periods []string) ([]model.OutstandingInvoice, error) {
	return a.store.ListPaidInvoices(ctx, managerID, periods)
}

func (a invoiceAPI) RecordedPayment(ctx context.Context, source, accountRef, transactionRef string) (string, bool, error) {
	return a.store.RecordedPayment(ctx, source, accountRef, transactionRef)
}

func (a invoiceAPI) MarkPaid(ctx context.Context, managerID string, payment model.PluginPayment) error {
	if payment.Source == "" || payment.InvoiceID == "" || payment.TransactionReference == "" {
		return errors.New("plugin payment needs source, invoice and transaction reference")
	}
	return a.store.SettleInvoice(ctx, managerID, payment)
}
