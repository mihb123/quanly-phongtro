package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	securityChannel     = "security"
	securityLevelName   = "WARNING"
	defaultSecurityFile = "logs/security.log"
	defaultSecurityDays = 30
)

type SecurityConfig struct {
	FilePath string
	MaxDays  int
}

type SecurityEvent struct {
	Time       time.Time
	Event      string
	Message    string
	Rules      []string
	ClientIP   string
	UserAgents []string
	Routes     []string
	Counts     map[string]int
	FirstSeen  time.Time
}

type securityRecord struct {
	Message   string          `json:"message"`
	Context   securityContext `json:"context"`
	LevelName string          `json:"level_name"`
	Channel   string          `json:"channel"`
	Datetime  string          `json:"datetime"`
}

type securityContext struct {
	Event      string         `json:"event"`
	Rules      []string       `json:"rules"`
	ClientIP   string         `json:"client_ip"`
	UserAgents []string       `json:"user_agents"`
	Routes     []string       `json:"routes"`
	Counts     map[string]int `json:"counts"`
	FirstSeen  string         `json:"first_seen"`
	Timestamp  string         `json:"timestamp"`
}

type SecurityLogger struct {
	mu   sync.Mutex
	file *rotatingFile
}

func NewSecurityLogger(cfg SecurityConfig) (*SecurityLogger, error) {
	path := strings.TrimSpace(cfg.FilePath)
	if path == "" {
		path = defaultSecurityFile
	}
	maxDays := cfg.MaxDays
	if maxDays <= 0 {
		maxDays = defaultSecurityDays
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve security log path %q: %w", path, err)
	}

	return &SecurityLogger{file: &rotatingFile{basePath: absPath, maxDays: maxDays}}, nil
}

func (l *SecurityLogger) Log(event SecurityEvent) {
	if l == nil {
		return
	}

	if event.Time.IsZero() {
		event.Time = time.Now()
	}

	record := securityRecord{
		Message:   event.Message,
		LevelName: securityLevelName,
		Channel:   securityChannel,
		Datetime:  event.Time.Format(time.RFC3339Nano),
		Context: securityContext{
			Event:      event.Event,
			Rules:      event.Rules,
			ClientIP:   event.ClientIP,
			UserAgents: event.UserAgents,
			Routes:     event.Routes,
			Counts:     event.Counts,
			FirstSeen:  event.FirstSeen.Format("2006-01-02 15:04:05"),
			Timestamp:  event.Time.Format("2006-01-02 15:04:05"),
		},
	}

	payload, err := json.Marshal(record)
	if err != nil {
		log.Printf("security log marshal: %v", err)
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if _, err := l.file.write(append(payload, '\n'), event.Time); err != nil {
		log.Printf("security log write: %v", err)
	}
}

func (l *SecurityLogger) Close() error {
	if l == nil {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.close()
}
