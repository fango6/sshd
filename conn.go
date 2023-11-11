package sshd

import (
	"net"
	"time"
)

const (
	maxIdleTimeout = time.Minute * 30
)

type Conn struct {
	net.Conn

	readTimeout  time.Duration
	writeTimeout time.Duration
	idleTimeout  time.Duration
}

func (c *Conn) Read(b []byte) (int, error) {
	if c.readTimeout > 0 {
		c.Conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	}
	return c.Conn.Read(b)
}

func (c *Conn) Write(b []byte) (int, error) {
	if c.writeTimeout > 0 {
		c.Conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	}
	return c.Conn.Write(b)
}

func (c *Conn) Close() error {
	if c.idleTimeout > 0 {
		c.Conn.SetDeadline(time.Now().Add(c.idleTimeout))
	} else {
		c.SetDeadline(time.Now().Add(maxIdleTimeout))
	}
	return c.Conn.Close()
}
