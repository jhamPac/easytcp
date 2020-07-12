package main

import (
	"log"
	"net"
)

// Server represents Server type
type Server struct {
	Addr string
}

func (srv Server) ListenAndServe() error {
	addr := srv.Addr
	if addr == "" {
		addr = ":9000"
	}
	log.Printf("starting server on %v\n", addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("error accepting connection %v", err)
			continue
		}
		log.Printf("accepted connection from %v", conn.RemoteAddr())
		handle(conn)
	}
}
