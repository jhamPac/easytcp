package main

import (
	"bufio"
	"log"
	"net"
	"strings"
)

// Server represents Server type
type Server struct {
	Addr string
}

// ListenAndServe executes the server
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

func handle(conn net.Conn) error {
	defer func() {
		log.Printf("closing connection from %v", conn.RemoteAddr())
	}()
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	scanner := bufio.NewScanner(r)
	for {
		scanned := scanner.Scan()
		if !scanned {
			if err := scanner.Err(); err != nil {
				log.Printf("%v(%v)", err, conn.RemoteAddr())
				return err
			}
			break
		}
		w.WriteString(strings.ToUpper(scanner.Text()) + "\n")
		w.Flush()
	}
	return nil
}
