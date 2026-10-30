package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

type Store interface {
	ListPluginSettings(ctx context.Context, managerID string) ([]model.PluginSetting, error)
	GetPluginSetting(ctx context.Context, managerID, pluginID string) (*model.PluginSetting, error)
	SetPluginEnabled(ctx context.Context, managerID, pluginID string, enabled bool) error
	SavePluginData(ctx context.Context, managerID, pluginID string, encryptedData *string) error
	ListOutstandingInvoices(ctx context.Context, managerID string) ([]model.OutstandingInvoice, error)
	PaymentRecorded(ctx context.Context, source, accountRef, transactionRef string) (bool, error)
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
	routers map[string]chi.Router
	order   []string
}

func NewManager(store Store, encryptionKey []byte, plugins []Plugin) *Manager {
	m := &Manager{store: store, key: encryptionKey, plugins: map[string]Plugin{}, routers: map[string]chi.Router{}}
	for _, p := range plugins {
		manifest := p.Manifest()
		if err := p.Init(&host{store: store, key: encryptionKey, pluginID: manifest.ID}); err != nil {
			log.Printf("plugin %s: init failed, skipped: %v", manifest.ID, err)
			continue
		}
		router := chi.NewRouter()
		p.Routes(router)
		m.plugins[manifest.ID] = p
		m.routers[manifest.ID] = router
		m.order = append(m.order, manifest.ID)
	}
	return m
}

func (m *Manager) Routes(r chi.Router) {
	r.Get("/", m.handleList)
	r.Put("/{pluginID}/enabled", m.handleSetEnabled)
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
}

func (h *host) Invoices() InvoiceAPI { return invoiceAPI{store: h.store} }

func (h *host) Storage() Storage { return h }

func (h *host) Load(ctx context.Context, managerID string, out any) (bool, error) {
	setting, err := h.store.GetPluginSetting(ctx, managerID, h.pluginID)
	if err != nil || setting == nil || setting.EncryptedData == nil || *setting.EncryptedData == "" {
		return false, err
	}
	plain, err := security.Decrypt(*setting.EncryptedData, h.key)
	if err != nil {
		return false, fmt.Errorf("decrypt plugin data: %w", err)
	}
	return true, json.Unmarshal([]byte(plain), out)
}

func (h *host) Save(ctx context.Context, managerID string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	encrypted, err := security.Encrypt(string(raw), h.key)
	if err != nil {
		return fmt.Errorf("encrypt plugin data: %w", err)
	}
	return h.store.SavePluginData(ctx, managerID, h.pluginID, &encrypted)
}

func (h *host) Clear(ctx context.Context, managerID string) error {
	return h.store.SavePluginData(ctx, managerID, h.pluginID, nil)
}

type invoiceAPI struct {
	store Store
}

func (a invoiceAPI) ListOutstanding(ctx context.Context, managerID string) ([]model.OutstandingInvoice, error) {
	return a.store.ListOutstandingInvoices(ctx, managerID)
}

func (a invoiceAPI) PaymentRecorded(ctx context.Context, source, accountRef, transactionRef string) (bool, error) {
	return a.store.PaymentRecorded(ctx, source, accountRef, transactionRef)
}

func (a invoiceAPI) MarkPaid(ctx context.Context, managerID string, payment model.PluginPayment) error {
	if payment.Source == "" || payment.InvoiceID == "" || payment.TransactionReference == "" {
		return errors.New("plugin payment needs source, invoice and transaction reference")
	}
	return a.store.SettleInvoice(ctx, managerID, payment)
}
