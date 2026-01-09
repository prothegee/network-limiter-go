package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"time"

	gen "github.com/network-limiter-go/pkg"
	config "github.com/network-limiter-go/pkg/config"
	grpc_limiter "github.com/network-limiter-go/pkg/grpc"
	pb "github.com/network-limiter-go/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// --------------------------------------------------------- //

type Server struct {
	pb.UnimplementedLocationServer
}

func (s *Server) SendLocationAndSave(ctx context.Context,
									 req *pb.LocationReq) (*pb.LocationResp, error) {
	ok := false
	message := "n/a"

	reqMsg := strings.ToLower(req.GetMessage())
	switch reqMsg {
		case "bad": {
			message = fmt.Sprintf("\"%s\" grpc; long: %f; lat: %f", reqMsg, req.Long, req.Lat)
		}
		case "good": {
			ok = true
			message = fmt.Sprintf("\"%s\" grpc; long: %f; lat: %f", reqMsg, req.Long, req.Lat)
		}
		default: {
			ok = true
			message = fmt.Sprintf("\"%s\" grpc; long: %f; lat: %f", reqMsg, req.Long, req.Lat)
		}
	}

	// assume intense rw/io from 1 to 6 seconds
	rn := rand.New(rand.NewSource(time.Now().UnixNano()))
	rnNum := gen.RandomNumberSign(rn, 1, 6)
	time.Sleep(time.Duration(rnNum) * time.Second)

	return &pb.LocationResp{
		Ok: ok,
		Message: message,
	}, nil
}

// --------------------------------------------------------- //

func main() {
	cfg, err := config.ConfigServerGrpcLoad("../../config.grpc.json")
	if err != nil {
		log.Fatalf("can't load config: %v\n", err)
		return
	}

	maxReqInterval := time.Duration(cfg.Limiter.MaxRequestInterval) * time.Second
	cleanupInterval := time.Duration(cfg.Limiter.CleanupOldRequestInterval) * time.Second

	limiter := grpc_limiter.NewGrpcRateLimiter(
		uint(cfg.Limiter.MaxRequestPerIp), maxReqInterval)
	middleware := grpc_limiter.NewGrpcMiddleware(limiter)

	server := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.Limit()),
	)

	pb.RegisterLocationServer(server, &Server{})

	reflection.Register(server)

	go grpc_limiter.CleanupOldRequest(limiter, cleanupInterval)
	
	listAddr, err := net.Listen("tcp",
		fmt.Sprintf("%s:%d", cfg.Listener.Address, cfg.Listener.Port)); if err != nil {
			log.Fatalf("error: %v\n", err)
		}

	log.Printf("INFO: run grpc server on %s\n",
		fmt.Sprintf("%s:%d", cfg.Listener.Address, cfg.Listener.Port))
	log.Fatal(server.Serve(listAddr))
}
