package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/example/grpc-plugin-app/pkg/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type HealthCheck struct {
	Interval    time.Duration
	MaxRetries  int
	RetryDelay  time.Duration
	OnUnhealthy func(error)
}

// MonitorPluginHealth monitors the health of a plugin connection
func MonitorPluginHealth(ctx context.Context, client *grpc.Client, config HealthCheck) {
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	healthClient := healthpb.NewHealthClient(client.Conn)

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
