package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}$`)

type manifest struct {
	API     int    `json:"api"`
	ID      string `json:"id"`
	Version string `json:"version"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Cách dùng (chạy trong thư mục plugin-sdk): go run ./cmd/pack <thư-mục-plugin>")
		fmt.Fprintln(os.Stderr, "Ví dụ: go run ./cmd/pack examples/debt-reminder")
		os.Exit(2)
	}
	if err := pack(filepath.Clean(os.Args[1])); err != nil {
		fmt.Fprintln(os.Stderr, "Lỗi:", err)
		os.Exit(1)
	}
}

func pack(dir string) error {
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return err
	}
	var m manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return fmt.Errorf("manifest.json không hợp lệ: %w", err)
	}
	if m.API != 1 || !idPattern.MatchString(m.ID) || m.Version == "" {
		return fmt.Errorf(`manifest.json cần "api": 1, "id" hợp lệ và "version"`)
	}

	build := filepath.Join(dir, "build")
	if err := os.RemoveAll(build); err != nil {
		return err
	}
	files := [][2]string{{"manifest.json", filepath.Join(dir, "manifest.json")}}

	if isDir(filepath.Join(dir, "backend")) {
		wasm := filepath.Join(build, "backend", "plugin.wasm")
		pkg := "./" + filepath.ToSlash(filepath.Join(dir, "backend"))
		cmd := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", wasm, pkg)
		cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
		if err := run(cmd); err != nil {
			return fmt.Errorf("build backend: %w", err)
		}
		files = append(files, [2]string{"backend/plugin.wasm", wasm})
	}

	if isFile(filepath.Join(dir, "web", "index.tsx")) {
		cmd := exec.Command("npx", "vite", "build", "--config", "web/vite.config.ts", "--logLevel", "warn")
		cmd.Env = append(os.Environ(), "PLUGIN_DIR="+dir)
		if err := run(cmd); err != nil {
			return fmt.Errorf("build web: %w", err)
		}
		files = append(files, [2]string{"web/index.js", filepath.Join(build, "web", "index.js")})
		if css := filepath.Join(build, "web", "index.css"); isFile(css) {
			files = append(files, [2]string{"web/index.css", css})
		}
	}

	for _, name := range []string{"README.md", "CHANGELOG.md", "LICENSE", "LICENSE.md", "LICENSE.txt"} {
		if isFile(filepath.Join(dir, name)) {
			files = append(files, [2]string{name, filepath.Join(dir, name)})
		}
	}
	if len(files) == 1 {
		return fmt.Errorf("%s không có backend/ hay web/index.tsx", dir)
	}

	out := filepath.Join("dist", fmt.Sprintf("%s-%s.zip", m.ID, m.Version))
	if err := writeZip(out, files); err != nil {
		return err
	}
	stat, err := os.Stat(out)
	if err != nil {
		return err
	}
	fmt.Printf("Đã tạo %s (%.1f MB)\n", out, float64(stat.Size())/(1<<20))
	for _, f := range files {
		fmt.Println("  " + f[0])
	}
	return nil
}

func writeZip(out string, files [][2]string) error {
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for _, file := range files {
		w, err := zw.Create(file[0])
		if err != nil {
			return err
		}
		src, err := os.Open(file[1])
		if err != nil {
			return err
		}
		_, err = io.Copy(w, src)
		src.Close()
		if err != nil {
			return err
		}
	}
	return zw.Close()
}

func run(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func isDir(name string) bool {
	stat, err := os.Stat(name)
	return err == nil && stat.IsDir()
}

func isFile(name string) bool {
	stat, err := os.Stat(name)
	return err == nil && stat.Mode().IsRegular()
}
