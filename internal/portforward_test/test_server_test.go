package portforward_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net"
	"testing"

	"golang.org/x/crypto/ssh"
)

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
	failedAttempts int
}

func setupTestServer(t *testing.T, opts testServerOpts) (net.Listener, *ssh.Client, string) {
	t.Helper()

	// Start a test TCP server that will be our "remote" target
	tcpServer, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to start TCP server: %v", err)
	}

	tcpServerAddr := tcpServer.Addr().String()
	attempts := 0

	// Handle connections to our test TCP server
	go func() {
		for {
			conn, err := tcpServer.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				if attempts < opts.failedAttempts {
					attempts++
					conn.Close()
					return
				}
				_, err := io.WriteString(conn, "Hello from TCP server!")
				if err != nil {
					t.Log("Failed to write to connection", "err", err)
				}
			}(conn)
		}
	}()

	// Start test SSH server
	serverConfig := &ssh.ServerConfig{
		NoClientAuth: true,
	}

	signer := generateTestSSHKey(t)
	serverConfig.AddHostKey(signer)

	sshListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to start SSH server: %v", err)
	}

	// Accept SSH connections
	go func() {
		for {
			conn, err := sshListener.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				sshConn, chans, reqs, err := ssh.NewServerConn(conn, serverConfig)
				if err != nil {
					return
				}
				defer sshConn.Close()

				go ssh.DiscardRequests(reqs)

				for newChannel := range chans {
					if newChannel.ChannelType() != "direct-tcpip" {
						if err := newChannel.Reject(ssh.UnknownChannelType, "unknown channel type"); err != nil {
							t.Log("Failed to reject channel", "err", err)
						}
						continue
					}

					channel, requests, err := newChannel.Accept()
					if err != nil {
						return
					}
					go ssh.DiscardRequests(requests)

					// Connect to local TCP server
					targetConn, err := net.Dial("tcp", tcpServerAddr)
					if err != nil {
						channel.Close()
						continue
					}

					// Bind bidirectional communication
					go func() {
						defer channel.Close()
						defer targetConn.Close()
						if _, err := io.Copy(channel, targetConn); err != nil {
							t.Log("Failed to copy data from remote to local", "err", err)
						}
					}()
					go func() {
						defer channel.Close()
						defer targetConn.Close()
						if _, err := io.Copy(targetConn, channel); err != nil {
							t.Log("Failed to copy data from local to remote", "err", err)
						}
					}()
				}
			}(conn)
		}
	}()

	// Create SSH client
	sshClient, err := ssh.Dial("tcp", sshListener.Addr().String(), &ssh.ClientConfig{
		User:            "test",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		t.Fatalf("Failed to dial SSH server: %v", err)
	}

	return tcpServer, sshClient, tcpServerAddr
}
