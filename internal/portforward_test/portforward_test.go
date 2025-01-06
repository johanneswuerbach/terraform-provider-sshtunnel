package portforward_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/johanneswuerbach/terraform-provider-sshtunnel/internal/portforward"
)

func TestPortForwardIntegration(t *testing.T) {
	tcpServer, sshClient, tcpServerAddr := setupTestServer(t, testServerOpts{})
	defer tcpServer.Close()
	defer sshClient.Close()

	// Create port forward
	ctx := context.Background()
	config := &portforward.Config{
		RemoteAddr: tcpServerAddr,
	}

	listener, err := portforward.New(ctx, sshClient, config)
	if err != nil {
		t.Fatalf("Failed to create port forward: %v", err)
	}
	defer listener.Close()

	// Connect to forwarded port
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect to forwarded port: %v", err)
	}
	defer conn.Close()

	// Read response
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read from connection: %v", err)
	}

	response := string(buf[:n])
	expected := "Hello from TCP server!"
	if response != expected {
		t.Errorf("got %q, want %q", response, expected)
	}
}

func TestPortForwardRetry(t *testing.T) {
	tcpServer, sshClient, tcpServerAddr := setupTestServer(t, testServerOpts{failedAttempts: 2})
	defer tcpServer.Close()
	defer sshClient.Close()

	// Create port forward with retry configuration
	ctx := context.Background()
	config := &portforward.Config{
		RemoteAddr:    tcpServerAddr,
		RetryDelay:    10 * time.Millisecond, // Short delay for tests
		RetryAttempts: 2,                     // Should succeed on the 2nd retry attempt
	}

	listener, err := portforward.New(ctx, sshClient, config)
	if err != nil {
		t.Fatalf("Failed to create port forward: %v", err)
	}
	defer listener.Close()

	// Connect to forwarded port
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect to forwarded port: %v", err)
	}
	defer conn.Close()

	// Read response
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read from connection: %v", err)
	}

	response := string(buf[:n])
	expected := "Hello from TCP server!"
	if response != expected {
		t.Errorf("got %q, want %q", response, expected)
	}
}
