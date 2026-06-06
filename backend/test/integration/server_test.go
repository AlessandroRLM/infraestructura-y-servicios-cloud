package integration_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	healthv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/health/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/health/v1/healthv1connect"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/health"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/db"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/logging"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/server"
)

// IT-1: Server boots and accepts TCP connections.
func TestServerBoot(t *testing.T) {
	addr := strings.TrimPrefix(baseURL, "http://")
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("server is not reachable at %s: %v", addr, err)
	}
	if err := conn.Close(); err != nil {
		t.Logf("conn.Close: %v", err)
	}
}

// IT-2: Health.Ping round-trip via a real Connect client.
func TestHealthPingRoundTrip(t *testing.T) {
	client := healthv1connect.NewHealthServiceClient(
		http.DefaultClient,
		baseURL,
	)

	resp, err := client.Ping(context.Background(), connect.NewRequest(&healthv1.PingRequest{}))
	if err != nil {
		t.Fatalf("Ping RPC failed: %v", err)
	}
	if resp.Msg.Status == "" {
		t.Error("Ping response status is empty, want non-empty")
	}
}

// IT-3: GET /healthz returns 200 unconditionally.
func TestLivenessAlways200(t *testing.T) {
	resp, err := http.Get(baseURL + "/healthz") //nolint:noctx
	if err != nil {
		t.Fatalf("GET /healthz failed: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Logf("resp.Body.Close: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("/healthz status = %d, want 200", resp.StatusCode)
	}
}

// IT-4: GET /readyz returns 200 when both Postgres and Redis are healthy.
func TestReadyzBothDepsUp(t *testing.T) {
	resp, err := http.Get(baseURL + "/readyz") //nolint:noctx
	if err != nil {
		t.Fatalf("GET /readyz failed: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Logf("resp.Body.Close: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("/readyz status = %d, want 200 (both deps up)", resp.StatusCode)
	}
}

// IT-5: GET /readyz returns 503 when a dependency is down.
// We build a dedicated server instance wired with a dead-address Pinger so we
// avoid stopping the shared containers that the rest of the suite depends on.
func TestReadyzDepDown(t *testing.T) {
	deadPinger := &deadAddrPinger{err: fmt.Errorf("redis: connection refused")}

	realPool, ok := dbPool.(db.Pinger)
	if !ok {
		t.Fatal("dbPool does not implement db.Pinger")
	}

	log := logging.New(slog.LevelError)
	depDownSrv := server.New(log, realPool, deadPinger, health.Register)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to bind: %v", err)
	}
	depDownURL := "http://" + listener.Addr().String()

	go func() {
		if err := depDownSrv.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Logf("depDownSrv error: %v", err)
		}
	}()

	waitForServer(listener.Addr().String(), 3*time.Second)

	resp, err := http.Get(depDownURL + "/readyz") //nolint:noctx
	if err != nil {
		t.Fatalf("GET /readyz failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	if err := resp.Body.Close(); err != nil {
		t.Logf("resp.Body.Close: %v", err)
	}

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("/readyz status = %d, want 503 (redis dep down), body: %s", resp.StatusCode, body)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := depDownSrv.Shutdown(shutdownCtx); err != nil {
		t.Logf("depDownSrv.Shutdown: %v", err)
	}
}

// deadAddrPinger always returns the configured error on Ping.
type deadAddrPinger struct {
	err error
}

func (d *deadAddrPinger) Ping(_ context.Context) error {
	return d.err
}
