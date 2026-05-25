package logger

import (
	"log"
	"net/http"

	"github.com/fatih/color"
)

// Pre-built colored label printers for each level.
var (
	debugLabel = color.New(color.FgCyan, color.Bold).SprintFunc()
	infoLabel  = color.New(color.FgGreen, color.Bold).SprintFunc()
	warnLabel  = color.New(color.FgYellow, color.Bold).SprintFunc()
	errorLabel = color.New(color.FgRed, color.Bold).SprintFunc()
)

// logf is the shared internal writer.
func logf(label string, r *http.Request, status int, message string, err error) {
	if r == nil {
		if err != nil {
			log.Printf("%s status=%d msg=%q err=%v", label, status, message, err)
			return
		}
		log.Printf("%s status=%d msg=%q", label, status, message)
		return
	}

	if err != nil {
		log.Printf(
			"%s status=%d method=%s path=%s remote=%s msg=%q err=%v",
			label, status, r.Method, r.URL.Path, r.RemoteAddr, message, err,
		)
		return
	}

	log.Printf(
		"%s status=%d method=%s path=%s remote=%s msg=%q",
		label, status, r.Method, r.URL.Path, r.RemoteAddr, message,
	)
}

// Debug logs a DEBUG-level message (cyan).
func Debug(r *http.Request, status int, message string, err error) {
	logf(debugLabel("[DEBUG]"), r, status, message, err)
}

// Info logs an INFO-level message (green).
func Info(r *http.Request, status int, message string, err error) {
	logf(infoLabel("[INFO] "), r, status, message, err)
}

// Warn logs a WARN-level message (yellow).
func Warn(r *http.Request, status int, message string, err error) {
	logf(warnLabel("[WARN] "), r, status, message, err)
}

// Error logs an ERROR-level message (red).
func Error(r *http.Request, status int, message string, err error) {
	logf(errorLabel("[ERROR]"), r, status, message, err)
}
