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
	var remoteConn net.Conn
	var err error

	for i := int32(0); i <= conf.RetryAttempts; i++ {
		remoteConn, err = sshConn.Dial("tcp", conf.RemoteAddr)
		if err != nil {
			tflog.Warn(ctx, "failed to dial remote connection, retrying", map[string]interface{}{"err": err})
			time.Sleep(conf.RetryDelay)
			continue
		}
	}
	if err != nil {
		tflog.Error(ctx, "failed to dial remote connection", map[string]interface{}{"retry_attempts": conf.RetryAttempts, "err": err})
		return
	}
	defer remoteConn.Close()

	var wait chan struct{}
	go func() {
		if _, err := io.Copy(remoteConn, localConn); err != nil {
			tflog.Error(ctx, "failed to copy data from remote to local", map[string]interface{}{"err": err})
		}
		wait <- struct{}{}
	}()

	if _, err := io.Copy(localConn, remoteConn); err != nil {
		tflog.Error(ctx, "failed to copy data from local to remote", map[string]interface{}{"err": err})
	}

	<-wait

	defer localConn.Close()
}
