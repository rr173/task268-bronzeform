package httpapi

import (
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

// statusRecorder 捕获响应状态码与字节数，供访问日志使用。
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// WithMiddleware 包装 handler：panic 恢复 + 访问日志。
func WithMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		defer func() {
			if p := recover(); p != nil {
				log.Printf("panic recovered: %v\n%s", p, debug.Stack())
				writeJSON(rec, http.StatusInternalServerError,
					map[string]any{"error": "internal server error"})
			}
			log.Printf("%s %s -> %d (%d bytes, %s)",
				r.Method, r.URL.Path, rec.status, rec.bytes, time.Since(start))
		}()
		next.ServeHTTP(rec, r)
	})
}
