// Package lint chứa test tĩnh (AST) bảo vệ quy ước "log once, wrap everywhere".
package lint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// roots: các tầng phải wrap lỗi bằng apperr.
var roots = []string{"../repository", "../service"}

// skipDirs: thư mục không phải code nghiệp vụ (helper test, logger cũ sắp xoá).
var skipDirs = map[string]bool{}

// pending: package CHƯA migrate (tạm cho qua). Đã migrate xong toàn bộ: giữ rỗng.
// TestPendingIsShrinking nhắc gỡ package ra khỏi đây khi nó đã sạch.
var pending = map[string]bool{}

// TestErrorsAreWrapped: repository/service không được trả lỗi "trần".
// Trần = return <ident err> | return <ErrXxx sentinel> | return fmt.Errorf(...) / errors.New(...).
func TestErrorsAreWrapped(t *testing.T) {
	for _, dir := range packageDirs(t) {
		if pending[dir] {
			continue
		}
		for _, v := range violations(t, dir) {
			t.Error(v)
		}
	}
}

// TestPendingIsShrinking: package trong `pending` mà đã sạch thì phải được gỡ khỏi danh sách.
func TestPendingIsShrinking(t *testing.T) {
	dirs := map[string]bool{}
	for _, d := range packageDirs(t) {
		dirs[d] = true
	}
	for dir := range pending {
		if !dirs[dir] {
			t.Errorf("pending entry %s does not exist any more", dir)
			continue
		}
		if len(violations(t, dir)) == 0 {
			t.Errorf("%s is clean now: remove it from `pending`", dir)
		}
	}
}

func packageDirs(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				return nil
			}
			path = filepath.ToSlash(path)
			if skipDirs[path] {
				return filepath.SkipDir
			}
			if hasGoFiles(path) {
				out = append(out, path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	sort.Strings(out)
	return out
}

func hasGoFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
			return true
		}
	}
	return false
}

func violations(t *testing.T, dir string) []string {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			ast.Inspect(f, func(n ast.Node) bool {
				ret, ok := n.(*ast.ReturnStmt)
				if !ok || len(ret.Results) == 0 {
					return true
				}
				if reason := bare(ret.Results[len(ret.Results)-1]); reason != "" {
					out = append(out, fset.Position(ret.Pos()).String()+": "+reason)
				}
				return true
			})
		}
	}
	return out
}

func bare(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		if x.Name == "err" {
			return "bare `err` must be wrapped with apperr.Repo/Service/Wrap"
		}
		if strings.HasPrefix(x.Name, "Err") {
			return "sentinel " + x.Name + " must be wrapped with apperr"
		}
	case *ast.SelectorExpr:
		if strings.HasPrefix(x.Sel.Name, "Err") {
			return "sentinel " + x.Sel.Name + " must be wrapped with apperr"
		}
	case *ast.CallExpr:
		if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok {
				if (id.Name == "fmt" && sel.Sel.Name == "Errorf") || (id.Name == "errors" && sel.Sel.Name == "New") {
					return id.Name + "." + sel.Sel.Name + " must be wrapped with apperr"
				}
			}
		}
	}
	return ""
}
