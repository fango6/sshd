package main

import (
	"context"
	"log"
	"net"

	"github.com/fango6/proxyproto"
	"github.com/fango6/sshd"
	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

func main() {
	mux := sshd.NewServeMux()
	mux.HandleFunc("session", func(ctx context.Context, conn *ssh.ServerConn, newChannel ssh.NewChannel) error {
		log.Printf("recv %s connection\n", conn.RemoteAddr())
		return conn.Close()
	})

	// injects uuid in everyone connection
	connContextOption := sshd.WithConnContext(func(conn net.Conn) context.Context {
		var ctx = context.WithValue(context.Background(), "uuid", uuid.NewString())

		if ppConn, ok := conn.(*proxyproto.Conn); ok && ppConn != nil {
			ctx = context.WithValue(ctx, "vpceID", ppConn.GetVpceID())
		}
		return ctx
	})
	// clients are allowed to connect without authenticating.
	sshd.DefaultSshServerConfig.NoClientAuth = true
	if err := sshd.ListenAndServe(":56789", sshd.GetDefaultSshServerConfig, mux,
		sshd.WithProxyProtocol(true), connContextOption); err != nil {
		log.Println("serve error:", err)
	}
	log.Println("sshd exited")
}
