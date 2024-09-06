package main

import (
	"context"
	"log"
	"net"

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
	uuidOption := sshd.WithConnContext(func(conn net.Conn) context.Context {
		return context.WithValue(context.Background(), "uuid", uuid.NewString())
	})
	// clients are allowed to connect without authenticating.
	sshd.DefaultSshServerConfig.NoClientAuth = true
	if err := sshd.ListenAndServe(":56789", mux, uuidOption); err != nil {
		log.Println("serve error:", err)
	}
	log.Println("sshd exited")
}
