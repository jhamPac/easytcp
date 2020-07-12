package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

func main() {
	s := Server{"localhost:9000", time.Duration(10 * time.Second)}
	if err := s.ListenAndServe(); err != nil {
		log.Fatal("oops")
	}
}

// Server represents Server type
type Server struct {
	Addr        string
	IdleTimeout time.Duration
}

// ConnWrapper wraps around the net.Conn returned from net.Listen
type ConnWrapper struct {
	net.Conn
	IdleTimeout time.Duration
}

func (c *ConnWrapper) Write(p []byte) (int, error) {
	c.updateDeadLine()
	return c.Conn.Write(p)
}

func (c *ConnWrapper) Read(b []byte) (int, error) {
	c.updateDeadLine()
	r := io.LimitReader(c.Conn, 1<<(10*3))
	return r.Read(b)
}

func (c *ConnWrapper) updateDeadLine() {
	iTime := time.Now().Add(c.IdleTimeout)
	c.Conn.SetDeadline(iTime)
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
		c, err := listener.Accept()
		conn := &ConnWrapper{
			Conn:        c,
			IdleTimeout: srv.IdleTimeout,
		}

		if err != nil {
			log.Printf("error accepting connection %v", err)
			continue
		}
		log.Printf("accepted connection from %v", conn.RemoteAddr())
		conn.SetDeadline(time.Now().Add(conn.IdleTimeout))
		go handle(conn)
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
