package portforward

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/crypto/ssh"
)

const (
	defaultListenHost = "0.0.0.0"
)

type Config struct {
	LocalPort     *int32
	RemoteAddr    string
	RetryDelay    time.Duration
	RetryAttempts int32
}

// halfCloser is implemented by connections that can signal the end of their
// outbound stream without tearing the whole connection down. Both *net.TCPConn
// and the ssh.Channel-backed connection returned by ssh.Client.Dial satisfy it.
type halfCloser interface {
	CloseWrite() error
}

func New(ctx context.Context, conn *ssh.Client, conf *Config) (net.Listener, error) {
	var listenAddr string
	if conf.LocalPort != nil {
		listenAddr = fmt.Sprintf("%s:%d", defaultListenHost, *conf.LocalPort)
	} else {
		listenAddr = fmt.Sprintf("%s:0", defaultListenHost)
	}

	localListener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("net.Listen failed: %v", err)
	}

	go func() {
		for {
			// Accept a connection
			localConn, err := localListener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				tflog.Error(ctx, "failed to accept connection", map[string]interface{}{"err": err})
				return
			}

			go handleConnection(ctx, conn, localConn, conf)
		}
	}()

	return localListener, nil
}

func handleConnection(ctx context.Context, sshConn *ssh.Client, localConn net.Conn, conf *Config) {
	// Always hand the local peer a closed socket when this handler returns,
	// including on the dial-failure path below, so it observes the end of the
	// connection instead of waiting on a tunnel that no longer exists.
	defer closeConn(ctx, localConn, "local")

	remoteConn, err := dialRemote(ctx, sshConn, conf)
	if err != nil {
		tflog.Error(ctx, "failed to dial remote connection", map[string]interface{}{
			"retry_attempts": conf.RetryAttempts,
			"err":            err,
		})

		return
	}
	defer closeConn(ctx, remoteConn, "remote")

	// Shuttle bytes in both directions and wait for both to finish. The channel
	// is buffered so that neither copy can block forever on the send, which is
	// what makes it safe to return once both have reported in.
	done := make(chan struct{}, 2)
	go copyStream(ctx, done, remoteConn, localConn, "local", "remote")
	go copyStream(ctx, done, localConn, remoteConn, "remote", "local")

	<-done
	<-done
}

// dialRemote opens the remote end of the tunnel, retrying up to
// conf.RetryAttempts additional times before giving up. It returns as soon as a
// dial succeeds so no extra remote connections are opened and then abandoned.
func dialRemote(ctx context.Context, sshConn *ssh.Client, conf *Config) (net.Conn, error) {
	var lastErr error

	for i := int32(0); i <= conf.RetryAttempts; i++ {
		if i > 0 {
			time.Sleep(conf.RetryDelay)
		}

		remoteConn, err := sshConn.Dial("tcp", conf.RemoteAddr)
		if err == nil {
			return remoteConn, nil
		}

		lastErr = err
		tflog.Warn(ctx, "failed to dial remote connection, retrying", map[string]interface{}{
			"attempt":            i + 1,
			"remaining_attempts": conf.RetryAttempts - i,
			"err":                err,
		})
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no dial attempt was made, retry_attempts is %d", conf.RetryAttempts)
	}

	return nil, lastErr
}

// copyStream copies src into dst until the source is exhausted, then signals the
// end of the stream to dst and reports completion on done. It half-closes dst
// where possible so the opposite direction can still drain rather than being
// truncated mid-response.
func copyStream(ctx context.Context, done chan<- struct{}, dst, src net.Conn, from, to string) {
	defer func() { done <- struct{}{} }()

	if _, err := io.Copy(dst, src); err != nil && !errors.Is(err, net.ErrClosed) {
		tflog.Error(ctx, "failed to copy data", map[string]interface{}{
			"from": from,
			"to":   to,
			"err":  err,
		})
	}

	if hc, ok := dst.(halfCloser); ok {
		if err := hc.CloseWrite(); err != nil && !errors.Is(err, net.ErrClosed) {
			tflog.Debug(ctx, "failed to half-close connection", map[string]interface{}{
				"to":  to,
				"err": err,
			})
		}

		return
	}

	closeConn(ctx, dst, to)
}

// closeConn closes conn, ignoring the benign error returned when it has already
// been closed by the opposite copy direction or by a deferred cleanup.
func closeConn(ctx context.Context, conn net.Conn, side string) {
	if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		tflog.Debug(ctx, "failed to close connection", map[string]interface{}{
			"side": side,
			"err":  err,
		})
	}
}
