package pkg_grpc_limiter

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type GrpcRateLimiter struct {
	Mtx sync.RWMutex
	Requests map[string]map[string][]time.Time // ip / function/method name / timestamp
	MaxRequests uint
	Duration time.Duration
}

// @brief create new internal grpc limiter
//
// @param maxReq uint - max requeest number
//
// @param duration time.Duration - wind time duration limiter
//
// @return *NewGrpcRateLimiter
func NewGrpcRateLimiter(maxReq uint, duration time.Duration) *GrpcRateLimiter {
	return &GrpcRateLimiter{
		Requests: make(map[string]map[string][]time.Time),
		MaxRequests: maxReq,
		Duration: duration,
	}
}

func (lmtr *GrpcRateLimiter) CheckRequestLimit(ip, method string) bool {
	lmtr.Mtx.Lock()
	defer lmtr.Mtx.Unlock()

	now := time.Now()

	// // #1st attempt
	// // crash:
	// // map somewhow make crash
	// if _, ok := lmtr.Requests[ip]; ok {
	// 	lmtr.Requests[ip] = make(map[string][]time.Time)
	// }

	// #2nd attempt
	// init request to be check
	// could happen
	if lmtr.Requests == nil {
		lmtr.Requests = make(map[string]map[string][]time.Time)
	}

	// init ip need to be check
	if lmtr.Requests[ip] == nil {
		lmtr.Requests[ip] = make(map[string][]time.Time)
	}

	// init method to be check
	if lmtr.Requests[ip][method] == nil {
		lmtr.Requests[ip][method] = []time.Time{}
	}

	// cleanup for the old request from the current ip and method/function name
	validRequests := []time.Time{}
	for _, t := range lmtr.Requests[ip][method] {
		if now.Sub(t) <= lmtr.Duration {
			validRequests = append(validRequests, t)
		}
	}

	if len(validRequests) >= int(lmtr.MaxRequests) {
		return false
	}

	validRequests = append(validRequests, now)
	lmtr.Requests[ip][method] = validRequests

	return true
}

// @param lmtr *GrpcRateLimiter
//
// @param d time.Duration
func CleanupOldRequest(lmtr *GrpcRateLimiter, d time.Duration) {
	ticker := time.NewTicker(d)
	defer ticker.Stop()

	for range ticker.C {
		lmtr.Mtx.Lock()
		now := time.Now()

		for ip, methods := range lmtr.Requests {
			for method, timestamps := range methods {
				validRequests := []time.Time{}
				for _, t := range timestamps {
					if now.Sub(t) <= lmtr.Duration {
						validRequests = append(validRequests, t)
					}
				}

				if len(validRequests) == 0 {
					delete(methods, method)
				} else {
					lmtr.Requests[ip][method] = validRequests
				}
			}

			if len(methods) == 0 {
				delete(lmtr.Requests, ip)
			}
		}

		lmtr.Mtx.Unlock()
	}
}

func (rl *GrpcRateLimiter) GetRequestCount(ip, method string) int {
	rl.Mtx.RLock()
	defer rl.Mtx.RUnlock()

	if methods, ok := rl.Requests[ip]; ok {
		if timestamps, ok := methods[method]; ok {
			// count request in duration
			now := time.Now()
			count := 0
			for _, t := range timestamps {
				if now.Sub(t) <= rl.Duration {
					count++
				}
			}
			return count
		}
	}
	return 0
}

// --------------------------------------------------------- //

type GrpcMiddleware struct {
	Limiter *GrpcRateLimiter
}

func NewGrpcMiddleware(limiter *GrpcRateLimiter) *GrpcMiddleware {
	return &GrpcMiddleware{Limiter: limiter}
}

// @brief get client ip from middleware
//
// @note empty string need to be handled correctly, otherwise it's panic
//
// @return string - "" is error/unknown
func (m *GrpcMiddleware) ClientIP(ctx context.Context) string {
	// try to get from x-real-ip and x-forwarded-for
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if realIp := md.Get("x-real-ip"); len(realIp) > 0 {
			return realIp[0] // first index only
		}

		if forwarded := md.Get("x-forwarded-for"); len(forwarded) > 0 {
			ips := strings.Split(forwarded[0], ",") // first index only

			if len(ips) > 0 {
				return strings.TrimSpace(ips[0])
			}
		}
	}

	// peer info
	if pr, ok := peer.FromContext(ctx); ok {
		if addr, ok := pr.Addr.(*net.TCPAddr); ok {
			return addr.IP.String()
		}

		// non tcp address
		addrStr := pr.Addr.String()

		// parse as "ip:port"
		if host, _, err := net.SplitHostPort(addrStr); err == nil {
			return host
		}

		return addrStr
	}

	return ""
}

// @brief in-case of fire, helper for reset ip param
func (lmtr *GrpcRateLimiter) ResetIP(ip string) {
	lmtr.Mtx.Lock()
	defer lmtr.Mtx.Unlock()
	delete(lmtr.Requests, ip)
}

func (m *GrpcMiddleware) Limit() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ip := m.ClientIP(ctx)

		if len(ip) <= 0 {
			return nil, status.Error(
				codes.FailedPrecondition,
				"Precondition Failed; IP Address Required")
		}

		method := info.FullMethod

		if !m.Limiter.CheckRequestLimit(ip, method) {
			current := m.Limiter.GetRequestCount(ip, method)
			return nil, status.Errorf(
				codes.ResourceExhausted,
				"Rate limit exceeded for %s. Current: %d/%d requests per %v",
				method, current, m.Limiter.MaxRequests, m.Limiter.Duration,
			)
		}

		header := metadata.Pairs(
			"x-ratelimit-limit", fmt.Sprintf("%d", m.Limiter.MaxRequests),
			"x-ratelimit-duration", m.Limiter.Duration.String(),
			"x-ratelimit-ip", ip,
			"x-ratelimit-method", method,
		)
		grpc.SendHeader(ctx, header)

		return handler(ctx, req)
	}
}
