package pkg_http_limiter

import (
	"net/http"
	"sync"
	"time"
)

type HttpRateLimiter struct {
	Mtx sync.RWMutex
	Requests map[string][]time.Time
	MaxRequests uint
	Duration time.Duration
}

// @brief create new internal http limiter
//
// @param maxReq uint - max requeest number
//
// @param duration time.Duration - wind time duration limiter
//
// @return *NewHttpRateLimiter
func NewHttpRateLimiter(maxReq uint, duration time.Duration) *HttpRateLimiter {
	return &HttpRateLimiter{
		Requests: make(map[string][]time.Time),
		MaxRequests: maxReq,
		Duration: duration,
	}
}

func (lmtr *HttpRateLimiter) CheckRequestLimit(ip string) bool {
	lmtr.Mtx.Lock()
	defer lmtr.Mtx.Unlock()

	now := time.Now()

	// cleanup for the old request from the current ip
	validRequests := []time.Time{}
	for _, t := range lmtr.Requests[ip] {
		if now.Sub(t) <= lmtr.Duration {
			validRequests = append(validRequests, t)
		}
	}

	if len(validRequests) >= int(lmtr.MaxRequests) {
		return false
	}

	validRequests = append(validRequests, now)
	lmtr.Requests[ip] = validRequests

	return true
}

// @param lmtr *HttpRateLimiter
//
// @param d time.Duration
func CleanupOldRequest(lmtr *HttpRateLimiter, d time.Duration) {
	ticker := time.NewTicker(d)
	defer ticker.Stop()

	for range ticker.C {
		lmtr.Mtx.Lock()
		for ip, requests := range lmtr.Requests {
			validRequests := []time.Time{}
			now := time.Now()

			for _, t := range requests {
				if now.Sub(t) <= lmtr.Duration {
					validRequests = append(validRequests, t)
				}
			}

			if len(validRequests) == 0 {
				delete(lmtr.Requests, ip)
			} else {
				lmtr.Requests[ip] = validRequests
			}
		}
		lmtr.Mtx.Unlock()
	}
}

// --------------------------------------------------------- //

type HttpMiddleware struct {
	Limiter *HttpRateLimiter
}

// @brief in-case of fire, helper for reset ip param
func (lmtr *HttpRateLimiter) ResetIP(ip string) {
	lmtr.Mtx.Lock()
	defer lmtr.Mtx.Unlock()
	delete(lmtr.Requests, ip)
}

func (m *HttpMiddleware) Limit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		if xForwardedFor := r.Header.Get("X-Forwarded-For"); xForwardedFor != "" {
			ip = xForwardedFor
		} else if xRealIp := r.Header.Get("X-Real-IP"); xRealIp != "" {
			ip = xRealIp
		}

		if !m.Limiter.CheckRequestLimit(ip) {
			http.Error(w, "request limi exceeded", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}
