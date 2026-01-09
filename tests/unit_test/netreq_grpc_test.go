package unit_test

import (
	"context"
	"testing"
	"time"

	grpc_limiter "github.com/network-limiter-go/pkg/grpc"
	pb "github.com/network-limiter-go/protobuf"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestIntegration_GrpcRateLimit(t *testing.T) {
	rateLimiter := grpc_limiter.NewGrpcRateLimiter(3, 30*time.Second)
	middleware := &grpc_limiter.GrpcMiddleware{Limiter: rateLimiter}

	handler := func(ctx context.Context, req any) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)

		if !ok {
			return nil, status.Error(codes.InvalidArgument, "metadata required")
		}

		precondFailed := true
		xRealIp := ""
		xForwardedFor := ""

		// check first index for each required header
		if xRealIpHeaders := md.Get("x-real-ip"); len(xRealIpHeaders) > 0 {
			xRealIp = xRealIpHeaders[0]
		}
		if xForwardedForHeaders := md.Get("x-forwarded-for"); len(xForwardedForHeaders) > 0 {
			xForwardedFor = xForwardedForHeaders[0]
		}

		if precondFailed && len(xRealIp) >= 7 {
			precondFailed = false
		}
		if precondFailed && len(xForwardedFor) >= 7 {
			precondFailed = false
		}
		if precondFailed {
			return nil, status.Error(codes.FailedPrecondition, "Precond Failed")
		}

		return map[string]any{
			"message": "ok",
			"headers": md,
		}, nil
	}

	interceptor := middleware.Limit()

	t.Run("TEST: without required headers", func(t *testing.T) {
		ctx := context.Background()

		_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{
			FullMethod: pb.LOCATION_SEND_LOCATION_AND_SAVE,
		}, handler)

		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("expected %v, but got %v (err: %v)\n",
				codes.FailedPrecondition, status.Code(err), err)
		}
	})

	// make 10 request but fail after 3 attempts as in limiter with x-real-ip header
	t.Run("TEST: with x-real-ip", func(t *testing.T) {
		for i := 1; i <= 10; i++ {
			ctx := metadata.NewIncomingContext(context.Background(),
				metadata.Pairs("x-real-ip", "192.168.1.100"))

			_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{
				FullMethod: pb.LOCATION_SEND_LOCATION_AND_SAVE,
			}, handler)

			expected := codes.OK

			if i > 3 {
				expected = codes.ResourceExhausted
			}

			if status.Code(err) != expected {
				t.Errorf("request #%d: got status %v, want %v (err: %v)\n",
					i, status.Code(err), expected, err)
			}

			time.Sleep(10 * time.Millisecond)
		}
	})

	// make 10 request but fail after 3 attempts as in limiter with x-forwarded-for header
	t.Run("TEST: with x-forwarded-for", func(t *testing.T) {
		for i := 1; i <= 10; i++ {
			ctx := metadata.NewIncomingContext(context.Background(),
				metadata.Pairs("x-real-ip", "192.168.2.200"))

			_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{
				FullMethod: pb.LOCATION_SEND_LOCATION_AND_SAVE,
			}, handler)
		
			expected := codes.OK
		
			if i > 3 {
				expected = codes.ResourceExhausted
			}

			if status.Code(err) != expected {
				t.Errorf("request #%d: got status %v, want %v (err: %v)\n",
					i, status.Code(err), expected, err)
			}

			time.Sleep(10 * time.Millisecond)
		}
	})
}
