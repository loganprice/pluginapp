package shared

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// HealthCheck represents the health check configuration
type HealthCheck struct {
	Interval    time.Duration
	MaxRetries  int
	RetryDelay  time.Duration
	OnUnhealthy func(error)
}

// DefaultHealthCheck returns the default health check configuration
func DefaultHealthCheck() HealthCheck {
	return HealthCheck{
		Interval:   time.Second * 30,
		MaxRetries: 3,
		RetryDelay: time.Second * 5,
	}
}

// StartHealthServer starts the gRPC health checking server
func StartHealthServer(server *grpc.Server) *health.Server {
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)
	return healthServer
}

// MonitorPluginHealth monitors the health of a plugin connection
func MonitorPluginHealth(ctx context.Context, client *GRPCClient, config HealthCheck) {
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	healthClient := healthpb.NewHealthClient(client.conn)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var lastErr error
			for retry := 0; retry < config.MaxRetries; retry++ {
				checkCtx, cancel := context.WithTimeout(ctx, time.Second*5)
				resp, err := healthClient.Check(checkCtx, &healthpb.HealthCheckRequest{})
				cancel()

				if err == nil && resp.Status == healthpb.HealthCheckResponse_SERVING {
					lastErr = nil
					break
				}

				lastErr = fmt.Errorf("health check failed: %v", err)
				time.Sleep(config.RetryDelay)
			}

			if lastErr != nil && config.OnUnhealthy != nil {
				config.OnUnhealthy(lastErr)
			}
		}
	}
}

// EnableHealthCheck enables health checking for a plugin client
func (c *GRPCClient) EnableHealthCheck(ctx context.Context, config HealthCheck) {
	go MonitorPluginHealth(ctx, c, config)
}
