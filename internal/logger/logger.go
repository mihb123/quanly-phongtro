package logger

import (
	"log"
	"net/http"
)

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
