package middleware

import (
	"bytes"
	"log"
	"net/http"
	"strings"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

const maxLoggedErrorBodyBytes = 4096

type errorLoggingResponseWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *errorLoggingResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *errorLoggingResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}

	if w.status >= http.StatusInternalServerError && w.body.Len() < maxLoggedErrorBodyBytes {
		remaining := maxLoggedErrorBodyBytes - w.body.Len()
		if len(data) > remaining {
			data = data[:remaining]
		}

		_, _ = w.body.Write(data)
	}

	return w.ResponseWriter.Write(data)
}

func ErrorLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writer := &errorLoggingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(writer, r)

		if writer.status < http.StatusInternalServerError {
			return
		}

		reqID := chimiddleware.GetReqID(r.Context())
		body := strings.TrimSpace(writer.body.String())
		if body == "" {
			body = "<empty>"
		}

		log.Printf(
			"request failed\n  request_id: %s\n  method: %s\n  path: %s\n  status: %d\n  error: %s",
			reqID,
			r.Method,
			r.URL.RequestURI(),
			writer.status,
			body,
		)
	})
}
