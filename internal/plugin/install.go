package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/tetratelabs/wazero"
)

type installer struct {
	dir     string
	admins  map[string]bool
	runtime wazero.Runtime
	mu      sync.Mutex
}

func (m *Manager) EnableInstall(ctx context.Context, dir string, adminEmails []string) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create plugin dir: %w", err)
	}
	rt, err := newWasmRuntime(ctx)
	if err != nil {
		return fmt.Errorf("start wasm runtime: %w", err)
	}
	admins := map[string]bool{}
	for _, email := range adminEmails {
		if email = strings.ToLower(strings.TrimSpace(email)); email != "" {
			admins[email] = true
		}
	}
	m.installer = &installer{dir: dir, admins: admins, runtime: rt}

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read plugin dir: %w", err)
	}
	for _, d := range dirEntries {
		name := d.Name()
		if !d.IsDir() {
			continue
		}
		if strings.HasPrefix(name, ".") {
			if strings.HasPrefix(name, ".install-") || strings.HasPrefix(name, ".old-") {
				_ = os.RemoveAll(filepath.Join(dir, name))
			}
			continue
		}
		if err := m.loadInstalled(ctx, filepath.Join(dir, name)); err != nil {
			log.Printf("plugin %s: load failed, skipped: %v", name, err)
		}
	}
	return nil
}

func (m *Manager) Close(ctx context.Context) {
	if m.installer != nil {
		_ = m.installer.runtime.Close(ctx)
	}
}

func (m *Manager) loadInstalled(ctx context.Context, dir string) error {
	raw, err := os.ReadFile(filepath.Join(dir, manifestFile))
	if err != nil {
		return err
	}
	manifest, err := ParseManifest(raw)
	if err != nil {
		return err
	}
	if manifest.ID != filepath.Base(dir) {
		return fmt.Errorf("manifest id %q does not match folder", manifest.ID)
	}
	if e, exists := m.get(manifest.ID); exists && e.installed == nil {
		return fmt.Errorf("id %q is used by a built-in plugin", manifest.ID)
	}
	p := m.newInstalled(manifest, dir)
	p.hasWeb = fileExists(filepath.Join(dir, webScriptFile))
	p.hasStyle = p.hasWeb && fileExists(filepath.Join(dir, webStyleFile))
	code, err := os.ReadFile(filepath.Join(dir, backendFile))
	switch {
	case err == nil:
		if p.compiled, err = compileBackend(ctx, m.installer.runtime, code); err != nil {
			return err
		}
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}
	return m.add(p)
}

func (m *Manager) newInstalled(manifest Manifest, dir string) *installedPlugin {
	return &installedPlugin{manifest: manifest, dir: dir, runtime: m.installer.runtime, slots: make(chan struct{}, wasmConcurrency)}
}

func (m *Manager) install(ctx context.Context, pkg *Package) (*entry, error) {
	inst := m.installer
	inst.mu.Lock()
	defer inst.mu.Unlock()

	id := pkg.Manifest.ID
	if e, exists := m.get(id); exists && e.installed == nil {
		return nil, fail(http.StatusConflict, fmt.Sprintf("ID %q trùng với tiện ích có sẵn của hệ thống", id))
	}
	p := m.newInstalled(pkg.Manifest, filepath.Join(inst.dir, id))
	p.hasWeb = pkg.HasWeb()
	p.hasStyle = pkg.Files[webStyleFile] != nil
	if pkg.HasBackend() {
		compiled, err := compileBackend(ctx, inst.runtime, pkg.Files[backendFile])
		if err != nil {
			return nil, fail(http.StatusBadRequest, err.Error())
		}
		p.compiled = compiled
	}
	if err := writePackage(inst.dir, id, pkg.Files); err != nil {
		p.close()
		return nil, err
	}
	if err := m.add(p); err != nil {
		p.close()
		return nil, err
	}
	e, _ := m.get(id)
	return e, nil
}

func writePackage(root, id string, files map[string][]byte) error {
	tmp, err := os.MkdirTemp(root, ".install-"+id+"-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	for name, content := range files {
		target := filepath.Join(tmp, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(target, content, 0o640); err != nil {
			return err
		}
	}
	final := filepath.Join(root, id)
	old := filepath.Join(root, ".old-"+filepath.Base(tmp))
	hadOld := fileExists(final)
	if hadOld {
		if err := os.Rename(final, old); err != nil {
			return err
		}
	}
	if err := os.Rename(tmp, final); err != nil {
		if hadOld {
			_ = os.Rename(old, final)
		}
		return err
	}
	return os.RemoveAll(old)
}

func (m *Manager) uninstall(id string) error {
	inst := m.installer
	inst.mu.Lock()
	defer inst.mu.Unlock()

	e, exists := m.get(id)
	if !exists {
		return fail(http.StatusNotFound, model.ErrPluginNotFound.Error())
	}
	if e.installed == nil {
		return fail(http.StatusBadRequest, "Tiện ích có sẵn của hệ thống, không gỡ được")
	}
	m.remove(id)
	go e.installed.close()
	trash := filepath.Join(inst.dir, ".old-"+id)
	_ = os.RemoveAll(trash)
	if err := os.Rename(e.installed.dir, trash); err != nil {
		return err
	}
	return os.RemoveAll(trash)
}

func (m *Manager) canInstall(r *http.Request) bool {
	if m.installer == nil || len(m.installer.admins) == 0 {
		return false
	}
	claims, ok := security.ClaimsFromContext(r.Context())
	return ok && claims != nil && m.installer.admins[strings.ToLower(claims.Email)]
}

func (m *Manager) handleInstallInfo(w http.ResponseWriter, r *http.Request) {
	if _, ok := ManagerID(w, r); !ok {
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"allowed":     m.canInstall(r),
		"api_version": APIVersion,
		"max_size":    MaxPackageSize,
		"permissions": permissions,
	}, "")
}

func (m *Manager) handleInstall(w http.ResponseWriter, r *http.Request) {
	managerID, ok := ManagerID(w, r)
	if !ok {
		return
	}
	if !m.canInstall(r) {
		WriteError(w, http.StatusForbidden, "Chỉ quản trị hệ thống được cài tiện ích")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxPackageSize+1<<20)
	file, _, err := r.FormFile("file")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			WriteError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("Gói tiện ích tối đa %d MB", MaxPackageSize>>20))
			return
		}
		WriteError(w, http.StatusBadRequest, "Chưa chọn tệp tiện ích")
		return
	}
	defer file.Close()
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	data, err := io.ReadAll(io.LimitReader(file, MaxPackageSize+1))
	if err != nil || len(data) > MaxPackageSize {
		WriteError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("Gói tiện ích tối đa %d MB", MaxPackageSize>>20))
		return
	}
	pkg, err := ReadPackage(data)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	e, err := m.install(r.Context(), pkg)
	if err != nil {
		var se *statusError
		if errors.As(err, &se) {
			WriteError(w, se.status, se.message)
			return
		}
		log.Printf("plugin %s: install failed: %v", pkg.Manifest.ID, err)
		WriteError(w, http.StatusInternalServerError, "Không cài được tiện ích")
		return
	}
	claims, _ := security.ClaimsFromContext(r.Context())
	log.Printf("plugin %s %s installed by %s", pkg.Manifest.ID, pkg.Manifest.Version, claims.Email)
	enabled, err := m.isEnabled(r.Context(), managerID, e)
	if err != nil {
		enabled = pkg.Manifest.DefaultEnabled
	}
	WriteJSON(w, http.StatusOK, e.info(enabled), "Đã cài "+pkg.Manifest.Name)
}

func (m *Manager) handleUninstall(w http.ResponseWriter, r *http.Request) {
	if _, ok := ManagerID(w, r); !ok {
		return
	}
	if !m.canInstall(r) {
		WriteError(w, http.StatusForbidden, "Chỉ quản trị hệ thống được gỡ tiện ích")
		return
	}
	id := chi.URLParam(r, "pluginID")
	if err := m.uninstall(id); err != nil {
		var se *statusError
		if errors.As(err, &se) {
			WriteError(w, se.status, se.message)
			return
		}
		log.Printf("plugin %s: uninstall failed: %v", id, err)
		WriteError(w, http.StatusInternalServerError, "Không gỡ được tiện ích")
		return
	}
	claims, _ := security.ClaimsFromContext(r.Context())
	log.Printf("plugin %s uninstalled by %s", id, claims.Email)
	WriteJSON(w, http.StatusOK, nil, "Đã gỡ tiện ích")
}

func (m *Manager) handleAsset(w http.ResponseWriter, r *http.Request) {
	managerID, ok := ManagerID(w, r)
	if !ok {
		return
	}
	e, exists := m.get(chi.URLParam(r, "pluginID"))
	if !exists || e.installed == nil {
		WriteError(w, http.StatusNotFound, model.ErrPluginNotFound.Error())
		return
	}
	var name, contentType string
	switch chi.URLParam(r, "file") {
	case "index.js":
		name, contentType = webScriptFile, "text/javascript; charset=utf-8"
	case "index.css":
		name, contentType = webStyleFile, "text/css; charset=utf-8"
	}
	if name == "" || (name == webScriptFile && !e.installed.hasWeb) || (name == webStyleFile && !e.installed.hasStyle) {
		WriteError(w, http.StatusNotFound, "Không tìm thấy tệp giao diện")
		return
	}
	enabled, err := m.isEnabled(r.Context(), managerID, e)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Không kiểm tra được trạng thái plugin")
		return
	}
	if !enabled {
		WriteError(w, http.StatusForbidden, model.ErrPluginDisabled.Error())
		return
	}
	f, err := os.Open(filepath.Join(e.installed.dir, filepath.FromSlash(name)))
	if err != nil {
		WriteError(w, http.StatusNotFound, "Không tìm thấy tệp giao diện")
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Không đọc được tệp giao diện")
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, name, stat.ModTime(), f)
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}
