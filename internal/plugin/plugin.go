package plugin

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/handler/httpx"
	"github.com/mihb123/quanly-phongtro/internal/model"
)

const APIVersion = 1

const (
	PermInvoicesRead  = "invoices:read"
	PermInvoicesWrite = "invoices:write"
)

type Manifest struct {
	API            int            `json:"api,omitempty"`
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Version        string         `json:"version"`
	Description    string         `json:"description"`
	Author         string         `json:"author"`
	Homepage       string         `json:"homepage,omitempty"`
	DefaultEnabled bool           `json:"default_enabled"`
	Permissions    []string       `json:"permissions,omitempty"`
	Settings       []SettingField `json:"settings,omitempty"`
}

type SettingType string

const (
	SettingToggle   SettingType = "toggle"
	SettingText     SettingType = "text"
	SettingTextarea SettingType = "textarea"
	SettingPassword SettingType = "password"
	SettingNumber   SettingType = "number"
	SettingSelect   SettingType = "select"
)

type SettingOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type SettingField struct {
	Key         string          `json:"key"`
	Type        SettingType     `json:"type"`
	Label       string          `json:"label"`
	Description string          `json:"description,omitempty"`
	Group       string          `json:"group,omitempty"`
	Placeholder string          `json:"placeholder,omitempty"`
	Default     any             `json:"default,omitempty"`
	Options     []SettingOption `json:"options,omitempty"`
	Min         *float64        `json:"min,omitempty"`
	Max         *float64        `json:"max,omitempty"`
}

type Plugin interface {
	Manifest() Manifest
	Init(host Host) error
	Routes(r chi.Router)
}

type Host interface {
	Invoices() InvoiceAPI
	Config() Config
}

type InvoiceAPI interface {
	ListOutstanding(ctx context.Context, managerID string, periods []string) ([]model.OutstandingInvoice, error)
	ListPaid(ctx context.Context, managerID string, periods []string) ([]model.OutstandingInvoice, error)
	RecordedPayment(ctx context.Context, source, accountRef, transactionRef string) (invoiceID string, recorded bool, err error)
	MarkPaid(ctx context.Context, managerID string, payment model.PluginPayment) error
}

type Config interface {
	Get(ctx context.Context, managerID string) (Values, error)
	Set(ctx context.Context, managerID string, values map[string]any) error
}

type Values map[string]any

func (v Values) String(key string) string {
	s, _ := v[key].(string)
	return s
}

func (v Values) Bool(key string) bool {
	b, _ := v[key].(bool)
	return b
}

func (v Values) Number(key string) float64 {
	n, _ := v[key].(float64)
	return n
}

var (
	registryMu sync.Mutex
	registry   = map[string]Plugin{}
	idPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}$`)
)

func Register(p Plugin) {
	registryMu.Lock()
	defer registryMu.Unlock()
	id := p.Manifest().ID
	if !idPattern.MatchString(id) {
		panic(fmt.Sprintf("plugin: invalid id %q", id))
	}
	if _, exists := registry[id]; exists {
		panic(fmt.Sprintf("plugin: duplicate id %q", id))
	}
	registry[id] = p
}

func Registered() []Plugin {
	registryMu.Lock()
	defer registryMu.Unlock()
	plugins := make([]Plugin, 0, len(registry))
	for _, p := range registry {
		plugins = append(plugins, p)
	}
	sort.Slice(plugins, func(i, j int) bool { return plugins[i].Manifest().ID < plugins[j].Manifest().ID })
	return plugins
}

func ManagerID(w http.ResponseWriter, r *http.Request) (string, bool) {
	return httpx.GetManagerID(r, w)
}

func WriteJSON(w http.ResponseWriter, status int, data any, message string) {
	httpx.WriteJSON(w, status, data, message)
}

func WriteError(w http.ResponseWriter, status int, message string) {
	httpx.WriteError(w, status, message)
}
