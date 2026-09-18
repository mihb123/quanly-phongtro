package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	slowAPIChannel     = "slow_api"
	slowAPIMessage     = "Slow API detected"
	slowAPILevelName   = "WARNING"
	defaultSlowAPIFile = "logs/slow_api.log"
	defaultSlowAPIDays = 7
)

// SlowAPIConfig cấu hình bộ ghi log request chậm.
type SlowAPIConfig struct {
	Threshold time.Duration // Ngưỡng coi là chậm; <= 0 nghĩa là tắt tính năng.
	FilePath  string        // Đường dẫn file gốc, mặc định logs/slow_api.log.
	MaxDays   int           // Số ngày log được giữ lại, mặc định 7.
}

// SlowAPIEntry là dữ liệu một request chậm do middleware thu thập.
type SlowAPIEntry struct {
	Time         time.Time
	Method       string
	URI          string
	Path         string
	Route        string
	QueryParams  string
	StatusCode   int
	Duration     time.Duration
	ClientIP     string
	UserAgent    string
	RequestSize  int64
	ResponseSize int
}

// slowAPIRecord là cấu trúc JSON ghi xuống file, giữ cùng shape với log của amacsport
// để dùng chung công cụ phân tích.
type slowAPIRecord struct {
	Message   string         `json:"message"`
	Context   slowAPIContext `json:"context"`
	LevelName string         `json:"level_name"`
	Channel   string         `json:"channel"`
	Datetime  string         `json:"datetime"`
}

type slowAPIContext struct {
	Method       string  `json:"method"`
	URI          string  `json:"uri"`
	Path         string  `json:"path"`
	Route        string  `json:"route"`
	QueryParams  string  `json:"query_params"`
	DurationMs   float64 `json:"duration_ms"`
	StatusCode   int     `json:"status_code"`
	ClientIP     string  `json:"client_ip"`
	UserAgent    string  `json:"user_agent"`
	RequestSize  int64   `json:"request_size"`
	ResponseSize int     `json:"response_size"`
	// Số liệu heap là của cả tiến trình tại thời điểm request kết thúc (Go không đo được
	// bộ nhớ theo từng request), chỉ dùng để nhận biết áp lực bộ nhớ khi API chậm.
	HeapAllocBytes uint64 `json:"heap_alloc_bytes"`
	HeapSysBytes   uint64 `json:"heap_sys_bytes"`
	Goroutines     int    `json:"goroutines"`
	Timestamp      string `json:"timestamp"`
}

// SlowAPILogger ghi các request vượt ngưỡng xuống file JSON theo ngày.
type SlowAPILogger struct {
	threshold time.Duration
	mu        sync.Mutex
	file      *rotatingFile
}

// NewSlowAPILogger tạo logger request chậm. Trả về nil khi Threshold <= 0 (tắt tính năng);
// file log được mở lười ở lần ghi đầu tiên nên hàm này không chạm tới đĩa.
func NewSlowAPILogger(cfg SlowAPIConfig) (*SlowAPILogger, error) {
	if cfg.Threshold <= 0 {
		return nil, nil
	}

	path := strings.TrimSpace(cfg.FilePath)
	if path == "" {
		path = defaultSlowAPIFile
	}
	maxDays := cfg.MaxDays
	if maxDays <= 0 {
		maxDays = defaultSlowAPIDays
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve slow api log path %q: %w", path, err)
	}

	return &SlowAPILogger{
		threshold: cfg.Threshold,
		file:      &rotatingFile{basePath: absPath, maxDays: maxDays},
	}, nil
}

// Threshold trả về ngưỡng thời gian bị coi là chậm.
func (l *SlowAPILogger) Threshold() time.Duration {
	if l == nil {
		return 0
	}
	return l.threshold
}

// IsSlow cho biết một request có vượt ngưỡng hay không.
func (l *SlowAPILogger) IsSlow(duration time.Duration) bool {
	return l != nil && l.threshold > 0 && duration >= l.threshold
}

// Log ghi một request chậm. Lỗi ghi file không làm hỏng request, chỉ báo ra stdout.
func (l *SlowAPILogger) Log(entry SlowAPIEntry) {
	if l == nil {
		return
	}

	if entry.Time.IsZero() {
		entry.Time = time.Now()
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	record := slowAPIRecord{
		Message:   slowAPIMessage,
		LevelName: slowAPILevelName,
		Channel:   slowAPIChannel,
		Datetime:  entry.Time.Format(time.RFC3339Nano),
		Context: slowAPIContext{
			Method:         entry.Method,
			URI:            entry.URI,
			Path:           entry.Path,
			Route:          entry.Route,
			QueryParams:    entry.QueryParams,
			DurationMs:     durationMs(entry.Duration),
			StatusCode:     entry.StatusCode,
			ClientIP:       entry.ClientIP,
			UserAgent:      entry.UserAgent,
			RequestSize:    entry.RequestSize,
			ResponseSize:   entry.ResponseSize,
			HeapAllocBytes: mem.HeapAlloc,
			HeapSysBytes:   mem.HeapSys,
			Goroutines:     runtime.NumGoroutine(),
			Timestamp:      entry.Time.Format("2006-01-02 15:04:05"),
		},
	}

	payload, err := json.Marshal(record)
	if err != nil {
		log.Printf("slow api log marshal: %v", err)
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if _, err := l.file.write(append(payload, '\n'), entry.Time); err != nil {
		log.Printf("slow api log write: %v", err)
	}
}

// Close đóng file log đang mở.
func (l *SlowAPILogger) Close() error {
	if l == nil {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.close()
}

// durationMs đổi duration sang mili giây, làm tròn 2 chữ số thập phân.
func durationMs(d time.Duration) float64 {
	ms := float64(d) / float64(time.Millisecond)
	return float64(int64(ms*100+0.5)) / 100
}

// rotatingFile ghi log vào file theo ngày (slow_api-2006-01-02.log) và tự xóa
// các file cũ hơn maxDays mỗi khi sang ngày mới.
type rotatingFile struct {
	basePath string
	maxDays  int
	day      string
	file     *os.File
}

func (f *rotatingFile) write(payload []byte, now time.Time) (int, error) {
	if err := f.rotate(now); err != nil {
		return 0, err
	}
	return f.file.Write(payload)
}

func (f *rotatingFile) rotate(now time.Time) error {
	day := now.Format("2006-01-02")
	if f.file != nil && f.day == day {
		return nil
	}

	if f.file != nil {
		_ = f.file.Close()
		f.file = nil
	}

	dir := filepath.Dir(f.basePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create slow api log dir: %w", err)
	}

	file, err := os.OpenFile(f.dailyPath(day), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open slow api log file: %w", err)
	}

	f.file = file
	f.day = day
	f.purge()
	return nil
}

// dailyPath chèn ngày vào trước phần mở rộng: logs/slow_api.log -> logs/slow_api-2026-09-18.log
func (f *rotatingFile) dailyPath(day string) string {
	ext := filepath.Ext(f.basePath)
	return strings.TrimSuffix(f.basePath, ext) + "-" + day + ext
}

// purge xóa các file log vượt quá số ngày được giữ; tên file chứa ngày nên sort
// theo tên cũng chính là sort theo thời gian.
func (f *rotatingFile) purge() {
	ext := filepath.Ext(f.basePath)
	matches, err := filepath.Glob(strings.TrimSuffix(f.basePath, ext) + "-*" + ext)
	if err != nil || len(matches) <= f.maxDays {
		return
	}

	sort.Strings(matches)
	for _, path := range matches[:len(matches)-f.maxDays] {
		_ = os.Remove(path)
	}
}

func (f *rotatingFile) close() error {
	if f == nil || f.file == nil {
		return nil
	}

	err := f.file.Close()
	f.file = nil
	f.day = ""
	return err
}
