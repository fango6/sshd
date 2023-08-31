package sshd

import "net"

func Serve(ln net.Listener, fn GetSshServerConfig, handler Handler, options ...Option) error {
	return NewServer(fn, handler, options...).Serve(ln)
}

func ListenAndServe(addr string, fn GetSshServerConfig, handler Handler, options ...Option) error {
	srv := NewServer(fn, handler, options...)
	return srv.ListenAndServe(addr)
}
