package sshd

import (
	"log"
	"time"
)

// Option 创建 Server 的可选类型
type Option func(srv *Server)

func WithConnCallback(fn ConnCallback) Option {
	return func(srv *Server) {
		srv.ConnCallback = fn
	}
}

func WithConnContext(fn ConnContext) Option {
	return func(srv *Server) {
		srv.ConnContext = fn
	}
}

func WithGetSshServerConfig(fn GetSshServerConfig) Option {
	return func(srv *Server) {
		srv.GetSshServerConfig = fn
	}
}

func WithErrLogger(logger *log.Logger) Option {
	return func(srv *Server) {
		srv.ErrLogger = logger
	}
}

func WithReadTimeout(duration time.Duration) Option {
	return func(srv *Server) {
		srv.ReadTimeout = duration
	}
}

func WithWriteTimeout(duration time.Duration) Option {
	return func(srv *Server) {
		srv.WriteTimeout = duration
	}
}

func WithIdleTimeout(duration time.Duration) Option {
	return func(srv *Server) {
		srv.IdleTimeout = duration
	}
}
