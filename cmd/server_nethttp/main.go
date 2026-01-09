package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	gen "github.com/network-limiter-go/pkg"
	config "github.com/network-limiter-go/pkg/config"
	http_limiter "github.com/network-limiter-go/pkg/http"
)

// --------------------------------------------------------- //

type responseHome struct {
	Duration int `json:"duration"`
	XRealIp string `json:"x_real_ip"`
	XForwardedFor string `json:"x_forwarded_for"`

}

// --------------------------------------------------------- //

func handlerHome(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	precondFailed := true
	xRealIp := r.Header.Get("X-Real-IP")
	xForwardedFor := r.Header.Get("X-Forwarded-For")

	if precondFailed && len(xRealIp) >= 12 {
		precondFailed = false
	}
	if precondFailed && len(xForwardedFor) >= 0 {
		precondFailed = false
	}
	if precondFailed {
		http.Error(w, "Precondition Failed", http.StatusPreconditionFailed)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// heavy io sim.
	rn := rand.New(rand.NewSource(time.Now().UnixNano()))
	rnNum := gen.RandomNumberSign(rn, 0,  9)
	time.Sleep(time.Duration(rnNum) * time.Second)

	resp := responseHome{
		Duration: rnNum,
		XRealIp: xRealIp,
		XForwardedFor: xForwardedFor,
	}

	err := json.NewEncoder(w).Encode(resp); if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// --------------------------------------------------------- //

func main() {
	cfg, err := config.ConfigServerHttpLoad("../../config.http.json")
	if err != nil {
		log.Fatalf("can't load config: %v\n", err)
		return
	}
	listAddr := fmt.Sprintf("%s:%d", cfg.Listener.Address, cfg.Listener.Port)

	limiter := http_limiter.NewHttpRateLimiter(
		uint(cfg.Limiter.MaxRequestPerIp),
		time.Duration(cfg.Limiter.MaxRequestInterval)*time.Second)
	middleware := &http_limiter.HttpMiddleware{Limiter: limiter}

	mux := http.NewServeMux()

	mux.HandleFunc("/", middleware.Limit(handlerHome))

	go http_limiter.CleanupOldRequest(limiter,
		time.Duration(cfg.Limiter.CleanupOldRequestInterval)*time.Second)

	server := &http.Server{
		Addr: listAddr,
		Handler: mux,
		IdleTimeout: time.Duration(cfg.Server.IdleTimeout) * time.Second,
		ReadTimeout: time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	log.Printf("INFO: run httpserver on %s\n", listAddr)
	log.Fatal(server.ListenAndServe())
}
