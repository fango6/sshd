package sshd

import (
	"log"
)

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

func WithChannelHandler(chanType string, handler ChannelHandler) Option {
	return func(srv *Server) {
		if srv.ChannelHandlers == nil {
			srv.ChannelHandlers = make(map[string]ChannelHandler)
		}
		srv.ChannelHandlers[chanType] = handler
	}
}

func WithErrLogger(logger *log.Logger) Option {
	return func(srv *Server) {
		srv.ErrLogger = logger
	}
}
