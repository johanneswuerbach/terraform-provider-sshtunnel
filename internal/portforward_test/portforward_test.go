package portforward_test

import (
	"context"
	"errors"
	"io"
	"math"
	"net"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/johanneswuerbach/terraform-provider-sshtunnel/internal/portforward"
)

// testTimeout bounds every blocking assertion so a regression fails fast instead
// of hanging until the CI job's own timeout fires.
const testTimeout = 5 * time.Second

func TestPortForwardIntegration(t *testing.T) {
	srv := setupTestServer(t, testServerOpts{})

	conn := dialTunnel(t, srv, &portforward.Config{
		RemoteAddr: srv.remoteAddr,
	})

	readGreeting(t, conn)
}

func TestPortForwardRetry(t *testing.T) {
	srv := setupTestServer(t, testServerOpts{failedAttempts: 1})

	conn := dialTunnel(t, srv, &portforward.Config{
		RemoteAddr:    srv.remoteAddr,
		RetryDelay:    10 * time.Millisecond, // Short delay for tests.
		RetryAttempts: 3,
	})

	readGreeting(t, conn)

	// The first dial failed and the retry succeeded, so the loop has to stop
	// there rather than spending its remaining attempts opening further remote
	// connections that are then abandoned.
	if got, want := srv.DialAttempts(), int64(2); got != want {
		t.Errorf("got %d remote dial attempts, want %d", got, want)
	}
}

// TestPortForwardRetryExhausted checks that giving up on the remote still closes
// the local side, so the local peer fails instead of waiting on a tunnel that
// was never established.
func TestPortForwardRetryExhausted(t *testing.T) {
	srv := setupTestServer(t, testServerOpts{failedAttempts: math.MaxInt64})

	conn := dialTunnel(t, srv, &portforward.Config{
		RemoteAddr:    srv.remoteAddr,
		RetryDelay:    10 * time.Millisecond,
		RetryAttempts: 1,
	})

	expectEOF(t, conn)

	if got, want := srv.DialAttempts(), int64(2); got != want {
		t.Errorf("got %d remote dial attempts, want %d", got, want)
	}
}

// TestPortForwardClosesConnections guards against the connection handler leaking
// goroutines and sockets. The local peer has to observe the end of the stream
// once the remote closes, and once both ends are done nothing may still be
// running inside the handler.
func TestPortForwardClosesConnections(t *testing.T) {
	// Connections from earlier tests in this package may still be winding down.
	if stacks := waitForHandlerGoroutines(testTimeout); len(stacks) > 0 {
		t.Fatalf("%d connection-handling goroutines were already running when the test started:\n\n%s",
			len(stacks), strings.Join(stacks, "\n\n"))
	}

	srv := setupTestServer(t, testServerOpts{})

	conn := dialTunnel(t, srv, &portforward.Config{
		RemoteAddr: srv.remoteAddr,
	})

	readGreeting(t, conn)

	// The remote closed after greeting, so the tunnel has to pass that on
	// instead of leaving the local peer waiting for bytes nobody will send.
	expectEOF(t, conn)

	// Closing the local side ends the other copy direction, after which the
	// handler must return and release both sockets.
	if err := conn.Close(); err != nil {
		t.Fatalf("Failed to close connection: %v", err)
	}

	if stacks := waitForHandlerGoroutines(testTimeout); len(stacks) > 0 {
		t.Fatalf("%d connection-handling goroutines still running %s after the connection closed:\n\n%s",
			len(stacks), testTimeout, strings.Join(stacks, "\n\n"))
	}
}

// dialTunnel starts a port forward against srv and returns a connection to it.
func dialTunnel(t *testing.T, srv *testServer, config *portforward.Config) net.Conn {
	t.Helper()

	listener, err := portforward.New(context.Background(), srv.client, config)
	if err != nil {
		t.Fatalf("Failed to create port forward: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect to forwarded port: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	if err := conn.SetReadDeadline(time.Now().Add(testTimeout)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}

	return conn
}

// readGreeting reads the test server's greeting through the tunnel.
func readGreeting(t *testing.T, conn net.Conn) {
	t.Helper()

	buf := make([]byte, len(testServerGreeting))
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("Failed to read from connection: %v", err)
	}

	if got := string(buf); got != testServerGreeting {
		t.Errorf("got %q, want %q", got, testServerGreeting)
	}
}

// expectEOF asserts that the tunnel has propagated the end of the stream to the
// local peer, rather than leaving the read to hit its deadline.
func expectEOF(t *testing.T, conn net.Conn) {
	t.Helper()

	buf := make([]byte, 1)
	if _, err := conn.Read(buf); !errors.Is(err, io.EOF) {
		t.Fatalf("got %v reading from the tunnel, want io.EOF", err)
	}
}

// waitForHandlerGoroutines waits for the package's connection-handling
// goroutines to finish, returning the stacks of any that are still running when
// it gives up.
func waitForHandlerGoroutines(timeout time.Duration) []string {
	deadline := time.Now().Add(timeout)

	for {
		stacks := handlerGoroutines()
		if len(stacks) == 0 || time.Now().After(deadline) {
			return stacks
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// handlerGoroutines returns the stacks of the goroutines currently inside the
// portforward package's per-connection handling.
func handlerGoroutines() []string {
	buf := make([]byte, 1<<16)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			buf = buf[:n]

			break
		}

		buf = make([]byte, 2*len(buf))
	}

	var stacks []string

	for _, stack := range strings.Split(string(buf), "\n\n") {
		if strings.Contains(stack, "portforward.handleConnection") ||
			strings.Contains(stack, "portforward.copyStream") {
			stacks = append(stacks, stack)
		}
	}

	return stacks
}
