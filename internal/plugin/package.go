package plugin

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	MaxPackageSize    = 32 << 20
	maxPackageFiles   = 64
	maxUnpackedSize   = 64 << 20
	manifestFile      = "manifest.json"
	backendFile       = "backend/plugin.wasm"
	webScriptFile     = "web/index.js"
	webStyleFile      = "web/index.css"
	maxSettingsFields = 50
)

var (
	packageFiles = []string{manifestFile, backendFile, webScriptFile, webStyleFile, "README.md", "CHANGELOG.md", "LICENSE", "LICENSE.md", "LICENSE.txt"}
	permissions  = []string{PermInvoicesRead, PermInvoicesWrite}

	versionPattern    = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-[0-9A-Za-z.-]+)?$`)
	settingKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
)

type Package struct {
	Manifest Manifest
	Files    map[string][]byte
}

func (p *Package) HasBackend() bool { return p.Files[backendFile] != nil }

func (p *Package) HasWeb() bool { return p.Files[webScriptFile] != nil }

func ReadPackage(data []byte) (*Package, error) {
	files, err := readArchive(data, 0)
	if err != nil {
		return nil, err
	}
	files = stripWrapperDir(files)
	for name := range files {
		if !slices.Contains(packageFiles, name) {
			return nil, fmt.Errorf("Gói chứa tệp không được phép: %s (chỉ chấp nhận %s)", name, strings.Join(packageFiles, ", "))
		}
	}
	raw, ok := files[manifestFile]
	if !ok {
		return nil, errors.New("Gói thiếu manifest.json ở thư mục gốc")
	}
	manifest, err := ParseManifest(raw)
	if err != nil {
		return nil, err
	}
	pkg := &Package{Manifest: manifest, Files: files}
	if !pkg.HasBackend() && !pkg.HasWeb() {
		return nil, fmt.Errorf("Gói phải có %s hoặc %s", backendFile, webScriptFile)
	}
	if files[webStyleFile] != nil && !pkg.HasWeb() {
		return nil, fmt.Errorf("Gói có %s nhưng thiếu %s", webStyleFile, webScriptFile)
	}
	return pkg, nil
}

func ParseManifest(raw []byte) (Manifest, error) {
	var manifest Manifest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("manifest.json không hợp lệ: %v", err)
	}
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, fmt.Errorf("manifest.json: %w", err)
	}
	return manifest, nil
}

func validateManifest(m Manifest) error {
	switch {
	case m.API == 0:
		return errors.New(`thiếu "api" (phiên bản API plugin, hiện là 1)`)
	case m.API > APIVersion:
		return fmt.Errorf("plugin cần API %d, hệ thống mới hỗ trợ tới API %d", m.API, APIVersion)
	case !idPattern.MatchString(m.ID):
		return fmt.Errorf(`"id" %q không hợp lệ (chữ thường, số, gạch ngang; 2-63 ký tự)`, m.ID)
	case strings.TrimSpace(m.Name) == "" || utf8.RuneCountInString(m.Name) > 80:
		return errors.New(`"name" bắt buộc, tối đa 80 ký tự`)
	case !versionPattern.MatchString(m.Version):
		return fmt.Errorf(`"version" %q phải theo semver, ví dụ 1.0.0`, m.Version)
	case utf8.RuneCountInString(m.Description) > 500:
		return errors.New(`"description" tối đa 500 ký tự`)
	case utf8.RuneCountInString(m.Author) > 120:
		return errors.New(`"author" tối đa 120 ký tự`)
	}
	if m.Homepage != "" {
		u, err := url.Parse(m.Homepage)
		if err != nil || u.Scheme != "https" || u.Host == "" || len(m.Homepage) > 300 {
			return errors.New(`"homepage" phải là URL https`)
		}
	}
	for i, p := range m.Permissions {
		if !slices.Contains(permissions, p) {
			return fmt.Errorf("quyền %q không tồn tại (hỗ trợ: %s)", p, strings.Join(permissions, ", "))
		}
		if slices.Contains(m.Permissions[:i], p) {
			return fmt.Errorf("quyền %q bị lặp", p)
		}
	}
	return validateSettingFields(m.Settings)
}

func validateSettingFields(fields []SettingField) error {
	if len(fields) > maxSettingsFields {
		return fmt.Errorf("tối đa %d cài đặt", maxSettingsFields)
	}
	seen := map[string]bool{}
	for _, f := range fields {
		if !settingKeyPattern.MatchString(f.Key) {
			return fmt.Errorf("cài đặt %q: key chỉ gồm chữ thường, số, gạch dưới và bắt đầu bằng chữ", f.Key)
		}
		if seen[f.Key] {
			return fmt.Errorf("cài đặt %q bị lặp", f.Key)
		}
		seen[f.Key] = true
		if strings.TrimSpace(f.Label) == "" {
			return fmt.Errorf("cài đặt %q thiếu label", f.Key)
		}
		switch f.Type {
		case SettingToggle, SettingText, SettingTextarea, SettingNumber:
		case SettingPassword:
			if f.Default != nil {
				return fmt.Errorf("cài đặt %q: password không được có default", f.Key)
			}
		case SettingSelect:
			if len(f.Options) == 0 {
				return fmt.Errorf("cài đặt %q: select cần options", f.Key)
			}
		default:
			return fmt.Errorf("cài đặt %q: type %q không hỗ trợ", f.Key, f.Type)
		}
		if f.Default != nil {
			if _, err := validateSetting(f, f.Default); err != nil {
				return fmt.Errorf("cài đặt %q: default sai kiểu", f.Key)
			}
		}
	}
	return nil
}

func readArchive(data []byte, depth int) (map[string][]byte, error) {
	switch {
	case bytes.HasPrefix(data, []byte("PK\x03\x04")):
		return readZip(data)
	case bytes.HasPrefix(data, []byte{0x1f, 0x8b}) && depth == 0:
		gz, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("Không giải nén được .gz: %v", err)
		}
		inner, err := io.ReadAll(io.LimitReader(gz, maxUnpackedSize+1))
		if err != nil {
			return nil, fmt.Errorf("Không giải nén được .gz: %v", err)
		}
		if len(inner) > maxUnpackedSize {
			return nil, errors.New("Gói sau khi giải nén quá lớn")
		}
		return readArchive(inner, depth+1)
	case len(data) > 262 && string(data[257:262]) == "ustar":
		return readTar(data)
	}
	return nil, errors.New("Chỉ hỗ trợ gói .zip, .tar.gz (.tgz) hoặc .tar")
}

type unpacker struct {
	files map[string][]byte
	total int64
}

func (u *unpacker) add(name string, r io.Reader) error {
	clean, skip, err := cleanEntryPath(name)
	if err != nil || skip {
		return err
	}
	if _, exists := u.files[clean]; exists {
		return fmt.Errorf("Tệp %s bị lặp trong gói", clean)
	}
	if len(u.files) >= maxPackageFiles {
		return fmt.Errorf("Gói có quá nhiều tệp (tối đa %d)", maxPackageFiles)
	}
	content, err := io.ReadAll(io.LimitReader(r, maxUnpackedSize-u.total+1))
	if err != nil {
		return fmt.Errorf("Không đọc được %s: %v", clean, err)
	}
	u.total += int64(len(content))
	if u.total > maxUnpackedSize {
		return errors.New("Gói sau khi giải nén quá lớn")
	}
	u.files[clean] = content
	return nil
}

func readZip(data []byte) (map[string][]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("Không đọc được tệp .zip: %v", err)
	}
	u := &unpacker{files: map[string][]byte{}}
	for _, f := range zr.File {
		mode := f.Mode()
		if mode.IsDir() {
			continue
		}
		if !mode.IsRegular() {
			return nil, fmt.Errorf("Gói chứa liên kết/tệp đặc biệt: %s", f.Name)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("Không đọc được %s: %v", f.Name, err)
		}
		err = u.add(f.Name, rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
	}
	return u.files, nil
}

func readTar(data []byte) (map[string][]byte, error) {
	tr := tar.NewReader(bytes.NewReader(data))
	u := &unpacker{files: map[string][]byte{}}
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return u.files, nil
		}
		if err != nil {
			return nil, fmt.Errorf("Không đọc được tệp .tar: %v", err)
		}
		switch h.Typeflag {
		case tar.TypeDir, tar.TypeXGlobalHeader:
			continue
		case tar.TypeReg:
			if err := u.add(h.Name, tr); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("Gói chứa liên kết/tệp đặc biệt: %s", h.Name)
		}
	}
}

func cleanEntryPath(name string) (clean string, skip bool, err error) {
	name = strings.ReplaceAll(name, `\`, "/")
	clean = path.Clean(strings.TrimPrefix(name, "./"))
	if strings.HasPrefix(name, "/") || clean == ".." || strings.HasPrefix(clean, "../") || !fs.ValidPath(clean) {
		return "", false, fmt.Errorf("Đường dẫn không hợp lệ trong gói: %s", name)
	}
	base := path.Base(clean)
	if strings.HasPrefix(clean, "__MACOSX/") || base == ".DS_Store" || base == "Thumbs.db" || strings.HasPrefix(base, "._") {
		return "", true, nil
	}
	return clean, false, nil
}

func stripWrapperDir(files map[string][]byte) map[string][]byte {
	if _, ok := files[manifestFile]; ok {
		return files
	}
	prefix := ""
	for name := range files {
		dir, _, found := strings.Cut(name, "/")
		if !found || (prefix != "" && dir != prefix) {
			return files
		}
		prefix = dir
	}
	if _, ok := files[prefix+"/"+manifestFile]; !ok {
		return files
	}
	stripped := make(map[string][]byte, len(files))
	for name, content := range files {
		stripped[strings.TrimPrefix(name, prefix+"/")] = content
	}
	return stripped
}
