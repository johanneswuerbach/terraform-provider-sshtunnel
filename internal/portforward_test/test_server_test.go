package portforward_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// testServerGreeting is what the test TCP server writes before closing.
const testServerGreeting = "Hello from TCP server!"

func generateTestSSHKey(t *testing.T) ssh.Signer {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	signer, err := ssh.ParsePrivateKey(pem.EncodeToMemory(privateKeyPEM))
	if err != nil {
		t.Fatalf("Failed to parse private key: %v", err)
	}

	return signer
}

type testServerOpts struct {
	// failedAttempts is how many direct-tcpip channel requests the SSH server
	// rejects before it starts forwarding. A rejected channel makes the caller's
	// remote dial fail, which is what the retry loop is supposed to handle.
	failedAttempts int64
}

// testServer is an SSH server that forwards direct-tcpip channels to a TCP
// server which greets and immediately closes. Everything it starts is torn down
// via t.Cleanup.
type testServer struct {
	// client is an SSH client connected to the test SSH server.
	client *ssh.Client
	// remoteAddr is the address of the TCP server reachable through the tunnel.
	remoteAddr string

	// dialAttempts counts the direct-tcpip channel requests seen, i.e. how many
	// times the caller has tried to dial the remote address.
	dialAttempts atomic.Int64
}

// DialAttempts reports how many remote dials have been attempted through the
// tunnel so far.
func (s *testServer) DialAttempts() int64 {
	return s.dialAttempts.Load()
}

func setupTestServer(t *testing.T, opts testServerOpts) *testServer {
	t.Helper()

	// Start a test TCP server that will be our "remote" target.
	tcpListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to start TCP server: %v", err)
	}
	t.Cleanup(func() { _ = tcpListener.Close() })

	srv := &testServer{remoteAddr: tcpListener.Addr().String()}

	// Handle connections to our test TCP server.
	go func() {
		for {
			conn, err := tcpListener.Accept()
			if err != nil {
				return
			}

			go func(conn net.Conn) {
				defer conn.Close()
				_, _ = io.WriteString(conn, testServerGreeting)
			}(conn)
		}
	}()

	// Start test SSH server.
	serverConfig := &ssh.ServerConfig{
		NoClientAuth: true,
	}
	serverConfig.AddHostKey(generateTestSSHKey(t))

	sshListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to start SSH server: %v", err)
	}
	t.Cleanup(func() { _ = sshListener.Close() })

	// Accept SSH connections.
	go func() {
		for {
			conn, err := sshListener.Accept()
			if err != nil {
				return
			}

			go srv.serveSSH(conn, serverConfig, opts)
		}
	}()

	// Create SSH client.
	client, err := ssh.Dial("tcp", sshListener.Addr().String(), &ssh.ClientConfig{
		User:            "test",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to dial SSH server: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	srv.client = client

	return srv
}

// serveSSH handles one SSH connection, forwarding direct-tcpip channels to the
// test TCP server after rejecting the first opts.failedAttempts of them.
func (s *testServer) serveSSH(conn net.Conn, config *ssh.ServerConfig, opts testServerOpts) {
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		return
	}
	defer sshConn.Close()

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "direct-tcpip" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")

			continue
		}

		// Simulate a remote that is not reachable yet, so the dial itself fails
		// and the caller has something to retry.
		if s.dialAttempts.Add(1) <= opts.failedAttempts {
			_ = newChannel.Reject(ssh.ConnectionFailed, "simulated dial failure")

			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			return
		}
		go ssh.DiscardRequests(requests)

		// Connect to local TCP server.
		targetConn, err := net.Dial("tcp", s.remoteAddr)
		if err != nil {
			channel.Close()

			continue
		}

		// Bind bidirectional communication.
		go func() {
			defer channel.Close()
			defer targetConn.Close()
			_, _ = io.Copy(channel, targetConn)
		}()
		go func() {
			defer channel.Close()
			defer targetConn.Close()
			_, _ = io.Copy(targetConn, channel)
		}()
	}
}
