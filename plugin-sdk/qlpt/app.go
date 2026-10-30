package qlpt

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
)

type HandlerFunc func(c *Context) error

type route struct {
	method   string
	segments []string
	handler  HandlerFunc
}

type App struct {
	routes []route
}

func New() *App { return &App{} }

func (a *App) Get(pattern string, h HandlerFunc)    { a.Handle("GET", pattern, h) }
func (a *App) Post(pattern string, h HandlerFunc)   { a.Handle("POST", pattern, h) }
func (a *App) Put(pattern string, h HandlerFunc)    { a.Handle("PUT", pattern, h) }
func (a *App) Patch(pattern string, h HandlerFunc)  { a.Handle("PATCH", pattern, h) }
func (a *App) Delete(pattern string, h HandlerFunc) { a.Handle("DELETE", pattern, h) }

func (a *App) Handle(method, pattern string, h HandlerFunc) {
	a.routes = append(a.routes, route{method: strings.ToUpper(method), segments: splitPath(pattern), handler: h})
}

func (a *App) Run() {
	var req Request
	if err := call("request", nil, &req); err != nil {
		fmt.Fprintln(os.Stderr, "qlpt: không đọc được request:", err)
		os.Exit(1)
	}
	if req.Query == nil {
		req.Query = url.Values{}
	}
	c := &Context{Request: &req, params: map[string]string{}}
	if err := a.dispatch(c); err != nil {
		var e *Error
		if !errors.As(err, &e) {
			fmt.Fprintf(os.Stderr, "%s %s: %v\n", req.Method, req.Path, err)
			e = &Error{Status: 500, Message: "Tiện ích gặp lỗi"}
		}
		if c.responded {
			fmt.Fprintf(os.Stderr, "%s %s: lỗi sau khi đã trả response: %v\n", req.Method, req.Path, err)
			return
		}
		_ = c.respond(response{Status: e.Status, Message: e.Message})
		return
	}
	if !c.responded {
		_ = c.respond(response{Status: 200})
	}
}

func (a *App) dispatch(c *Context) error {
	path := splitPath(c.Request.Path)
	pathMatched := false
	for _, rt := range a.routes {
		params, ok := match(rt.segments, path)
		if !ok {
			continue
		}
		pathMatched = true
		if rt.method != c.Request.Method {
			continue
		}
		c.params = params
		return rt.handler(c)
	}
	if pathMatched {
		return &Error{Status: 405, Message: "Phương thức không được hỗ trợ"}
	}
	return &Error{Status: 404, Message: "Không tìm thấy"}
}

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

func match(pattern, path []string) (map[string]string, bool) {
	if len(pattern) != len(path) {
		return nil, false
	}
	params := map[string]string{}
	for i, seg := range pattern {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			value, err := url.PathUnescape(path[i])
			if err != nil || value == "" {
				return nil, false
			}
			params[seg[1:len(seg)-1]] = value
			continue
		}
		if seg != path[i] {
			return nil, false
		}
	}
	return params, true
}
