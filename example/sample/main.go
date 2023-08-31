package main

import (
	"context"
	"log"

	"github.com/fango6/sshd"
	"golang.org/x/crypto/ssh"
)

func main() {
	mux := sshd.NewServeMux()
	mux.HandleFunc("session", func(ctx context.Context, conn *ssh.ServerConn, newChannel ssh.NewChannel) error {
		log.Printf("recv %s connection\n", conn.RemoteAddr())
		return conn.Close()
	})

	// clients are allowed to connect without authenticating.
	sshd.DefaultSshServerConfig.NoClientAuth = true
	if err := sshd.ListenAndServe(":56789", sshd.GetDefaultSshServerConfig, mux); err != nil {
		log.Println("serve error:", err)
	}
	log.Println("sshd exited")
}
