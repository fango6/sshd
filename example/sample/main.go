package main

import (
	"context"
	"log"
	"net"

	"github.com/fango6/sshd"
	"golang.org/x/crypto/ssh"
)

var noAuthSSConf = &ssh.ServerConfig{
	NoClientAuth: true,
}

func main() {
	srv := sshd.NewServer(
		getSshServerConfig,
		nil,
		sshd.WithChannelHandler("session", func(ctx context.Context, tcpConn net.Conn, sshConn *ssh.ServerConn, newChannel ssh.NewChannel) {
			log.Printf("recv %s connection\n", tcpConn.RemoteAddr())
			if err := sshConn.Close(); err != nil {
				_ = err
			}
		}),
	)

	if err := srv.ListenAndServe(":56789"); err != nil {
		log.Println("serve error:", err)
	}
}

func getSshServerConfig(ctx context.Context) *ssh.ServerConfig {
	return noAuthSSConf
}
