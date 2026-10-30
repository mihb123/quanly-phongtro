package qlpt

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

type Request struct {
	Method    string              `json:"method"`
	Path      string              `json:"path"`
	Query     url.Values          `json:"query"`
	Header    map[string][]string `json:"header"`
	Body      []byte              `json:"body"`
	ManagerID string              `json:"manager_id"`
}

type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string { return e.Message }

func NewError(status int, message string) *Error { return &Error{Status: status, Message: message} }

func BadRequest(message string) *Error { return NewError(400, message) }

func NotFound(message string) *Error { return NewError(404, message) }

func Conflict(message string) *Error { return NewError(409, message) }

type Context struct {
	Request   *Request
	params    map[string]string
	responded bool
}

func (c *Context) Param(name string) string { return c.params[name] }

func (c *Context) Query(name string) string { return strings.TrimSpace(c.Request.Query.Get(name)) }

func (c *Context) Header(name string) string {
	for key, values := range c.Request.Header {
		if strings.EqualFold(key, name) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func (c *Context) ManagerID() string { return c.Request.ManagerID }

func (c *Context) Bind(v any) error {
	if len(c.Request.Body) == 0 {
		return BadRequest("Thiếu dữ liệu gửi lên")
	}
	if err := json.Unmarshal(c.Request.Body, v); err != nil {
		return BadRequest("Dữ liệu gửi lên không hợp lệ")
	}
	return nil
}

type response struct {
	Status      int    `json:"status"`
	Data        any    `json:"data,omitempty"`
	Message     string `json:"message,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Filename    string `json:"filename,omitempty"`
	Body        []byte `json:"body,omitempty"`
}

func (c *Context) respond(r response) error {
	if c.responded {
		return errors.New("qlpt: response đã được gửi")
	}
	c.responded = true
	return call("respond", r, nil)
}

func (c *Context) JSON(status int, data any) error {
	return c.respond(response{Status: status, Data: data})
}

func (c *Context) Message(status int, message string, data any) error {
	return c.respond(response{Status: status, Data: data, Message: message})
}

func (c *Context) File(contentType, filename string, body []byte) error {
	if body == nil {
		body = []byte{}
	}
	return c.respond(response{Status: 200, ContentType: contentType, Filename: filename, Body: body})
}
