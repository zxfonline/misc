package web

import (
	"net"
	"net/http/fcgi"

	"github.com/zxfonline/misc/log"
)

func (s *Server) listenAndServeFCGI(addr string) error {
	var l net.Listener
	var err error

	//if the path begins with a "/", assume it's a unix address
	if addr[0] == '/' {
		l, err = net.Listen("unix", addr)
	} else {
		l, err = net.Listen("tcp", addr)
	}

	//save the listener so it can be closed
	s.l = l

	if err != nil {
		log.Errorf("FCGI listen err:%v", err)
		return err
	}
	return fcgi.Serve(s.l, s)
}
