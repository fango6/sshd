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

	// injects uuid in every connection's context
	connContextOption := sshd.WithConnContext(func(conn net.Conn) context.Context {
		var ctx = context.WithValue(context.Background(), "uuid", uuid.NewString())

		if ppConn, ok := conn.(*proxyproto.Conn); ok && ppConn != nil {
			ctx = context.WithValue(ctx, "vpceID", ppConn.GetVpceID())
		}
		return ctx
	})

	sshd.PublicKeyAuth(func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		fingerprint := ssh.FingerprintSHA256(key)
		log.Printf("remote %s@%s, fingerprint %s", conn.User(), conn.RemoteAddr(), fingerprint)

		return &ssh.Permissions{
			Extensions: map[string]string{
				"fingerprint": fingerprint,
			},
		}, nil
	})

	if err := sshd.ListenAndServe(":56789", sshd.GetDefaultSshServerConfig, mux,
		sshd.WithProxyProtocol(true), connContextOption); err != nil {
		log.Println("serve error:", err)
	}
	log.Println("sshd exited")
}
