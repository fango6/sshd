package sshd

import (
	"log"
)

// Option 创建 Server 的可选类型
type Option func(srv *Server)

func WithProxyProtocol(enable bool) Option {
	return func(srv *Server) {
		srv.ProxyProtocol = enable
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

func WithHandler(h Handler) Option {
	return func(srv *Server) {
		srv.Handler = h
	}
}

func WithHandlerFunc(h HandlerFunc) Option {
	return func(srv *Server) {
		srv.Handler = HandlerFunc(h)
	}
}

func WithErrLogger(logger *log.Logger) Option {
	return func(srv *Server) {
		srv.ErrLogger = logger
	}
}
