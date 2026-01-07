package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	internal_http_limiter "github.com/network-limiter-go/internal/http"
)



func TestIntegration_rateLimit(t *testing.T) {
	rateLimiter := internal_http_limiter.NewHttpRateLimiter(3, 30*time.Second)
	middleware := &internal_http_limiter.HttpMiddleware{Limiter: rateLimiter}

	handler := middleware.Limit(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("only GET method is allowed")
		}

		precondFailed := true
		xRealIp := r.Header.Get("X-Real-IP")
		xForwardedFor := r.Header.Get("X-Forwarded-For")

		if precondFailed && len(xRealIp) >= 7 {
			precondFailed = false
		}
		if precondFailed && len(xForwardedFor) >= 7 {
			precondFailed = false
		}
		if precondFailed {
			http.Error(w, "Precond Failed", http.StatusPreconditionFailed)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	client := &http.Client{Timeout: 12 * time.Second}

	t.Run("TEST: without required headers", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL, nil)
		req.Header.Set("X-Real-IP", "")
		req.Header.Set("X-Forwarded-For", "")

		resp, err := client.Do(req); if err != nil {
			t.Fatalf("request check fail\n")
		}
		if resp.StatusCode != http.StatusPreconditionFailed {
			t.Fatalf("expected %d, but got %d\n",
				http.StatusPreconditionFailed, resp.StatusCode)
		}
		defer resp.Body.Close()
	})

	// make 10 request but fail after 3 attempt as in limiter with x-real-ip header
	t.Run("TEST: with X-Real-IP", func(t *testing.T) {
		for i := 1; i <= 10; i++ {
			req, _ := http.NewRequest("GET", ts.URL, nil)
			req.Header.Set("X-Real-IP", "192.168.1.100")

			resp, err := client.Do(req); if err != nil {
				t.Fatalf("request #%d failed: %v\n", i, err)
			}
			defer resp.Body.Close()

			expected := http.StatusOK

			if i > 3 {
				expected = http.StatusTooManyRequests
			}

			if resp.StatusCode != expected {
				t.Errorf("request #%d: got status %d, want %d\n", i, resp.StatusCode, expected)
			}

			time.Sleep(10 * time.Millisecond)
		}
	})

	// make 10 request but fail after 3 attempt as in limiter with x-forwarded-for header
	t.Run("TEST: with X-Forwarded-For", func(t *testing.T) {
		for i := 1; i <= 10; i++ {
			req, _ := http.NewRequest("GET", ts.URL, nil)
			req.Header.Set("X-Forwarded-For", "192.168.200.200")

			resp, err := client.Do(req); if err != nil {
				t.Fatalf("request #%d failed: %v\n", i, err)
			}
			defer resp.Body.Close()

			expected := http.StatusOK

			if i > 3 {
				expected = http.StatusTooManyRequests
			}

			if resp.StatusCode != expected {
				t.Errorf("request #%d: got status %d, want %d\n", i, resp.StatusCode, expected)
			}

			time.Sleep(10 * time.Millisecond)
		}
	})
}

