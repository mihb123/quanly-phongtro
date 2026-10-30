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

type Manifest struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Version        string `json:"version"`
	Description    string `json:"description"`
	Author         string `json:"author"`
	DefaultEnabled bool   `json:"default_enabled"`
}

type Plugin interface {
	Manifest() Manifest
	Init(host Host) error
	Routes(r chi.Router)
}

type Host interface {
	Invoices() InvoiceAPI
	Storage() Storage
}

type InvoiceAPI interface {
	ListOutstanding(ctx context.Context, managerID string) ([]model.OutstandingInvoice, error)
	PaymentRecorded(ctx context.Context, source, accountRef, transactionRef string) (bool, error)
	MarkPaid(ctx context.Context, managerID string, payment model.PluginPayment) error
}

type Storage interface {
	Load(ctx context.Context, managerID string, out any) (bool, error)
	Save(ctx context.Context, managerID string, value any) error
	Clear(ctx context.Context, managerID string) error
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
