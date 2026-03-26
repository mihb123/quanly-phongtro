package logger

import (
	"log"
	"net/http"
	"time"
)

func Info(r *http.Request, status int, duration time.Duration) {
	if r == nil {
		log.Printf("[%-3d] -", status)
		return
	}

	log.Printf(
		"[%d] %-7s %s | %s",
		status,
		r.Method,
		r.URL.Path,
		r.UserAgent(),
	)
}

func Error(r *http.Request, status int, message string, err error) {
	if r == nil {
		if err != nil {
			log.Printf("error status=%d msg=%q err=%v", status, message, err)
			return
		}
		log.Printf("error status=%d msg=%q", status, message)
		return
	}

	if err != nil {
		log.Printf(
			"error status=%d method=%s path=%s remote=%s msg=%q err=%v",
			status, r.Method, r.URL.Path, r.RemoteAddr, message, err,
		)
		return
	}

	log.Printf(
		"error status=%d method=%s path=%s remote=%s msg=%q",
		status, r.Method, r.URL.Path, r.RemoteAddr, message,
	)
}
