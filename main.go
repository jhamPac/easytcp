package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

func main() {
	s := Server{
		Addr:         "localhost:9000",
		IdleTimeout:  10 * time.Second,
		MaxReadbytes: 1 << (10 * 2),
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	// handle ^C interrupt signals gracefully
	go func() {
		select {
		case sig := <-c:
			fmt.Println("An interrupt signal was detected:", sig)
			s.Shutdown()
		}
	}()

	if err := s.ListenAndServe(); err != nil {
		log.Fatal("oops")
	}
}

// Server represents Server type
type Server struct {
	Addr         string
	IdleTimeout  time.Duration
	MaxReadbytes int64

	listener   net.Listener
	conns      map[*ConnWrapper]struct{}
	mu         sync.Mutex
	inShutDown bool
}

// ConnWrapper wraps around the net.Conn returned from net.Listen
type ConnWrapper struct {
	net.Conn

	IdleTimeout   time.Duration
	MaxReadBuffer int64
}

func (c *ConnWrapper) Write(p []byte) (int, error) {
	c.updateDeadLine()
	return c.Conn.Write(p)
}

func (c *ConnWrapper) Read(b []byte) (int, error) {
	c.updateDeadLine()
	r := io.LimitReader(c.Conn, c.MaxReadBuffer)
	return r.Read(b)
}

// Close closes the connection
func (c *ConnWrapper) Close() (err error) {
	err = c.Conn.Close()
	return err
}

func (c *ConnWrapper) updateDeadLine() {
	iTime := time.Now().Add(c.IdleTimeout)
	c.Conn.SetDeadline(iTime)
}

// ListenAndServe executes the server
func (srv *Server) ListenAndServe() error {
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
	srv.listener = listener
	for {
		if srv.inShutDown {
			break
		}
		c, err := listener.Accept()
		if err != nil {
			log.Printf("error accepting connection %v", err)
			continue
		}

		conn := &ConnWrapper{
			Conn:          c,
			IdleTimeout:   srv.IdleTimeout,
			MaxReadBuffer: srv.MaxReadbytes,
		}

		log.Printf("accepted connection from %v", conn.RemoteAddr())
		srv.trackConn(conn)
		conn.SetDeadline(time.Now().Add(conn.IdleTimeout))
		go srv.handle(conn)
	}
	return nil
}

func (srv *Server) trackConn(c *ConnWrapper) {
	defer srv.mu.Unlock()
	srv.mu.Lock()
	if srv.conns == nil {
		srv.conns = make(map[*ConnWrapper]struct{})
	}
	srv.conns[c] = struct{}{}
}

func (srv *Server) handle(conn *ConnWrapper) error {
	defer func() {
		log.Printf("closing connection from %v", conn.RemoteAddr())
		conn.Close()
		srv.deleteConn(conn)
	}()
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	scanner := bufio.NewScanner(r)

	sc := make(chan bool)
	deadline := time.After(conn.IdleTimeout)
	for {
		go func(s chan bool) {
			s <- scanner.Scan()
		}(sc)

		select {
		case <-deadline:
			return nil
		case scanned := <-sc:
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
	}
}

func (srv *Server) deleteConn(conn *ConnWrapper) {
	defer srv.mu.Unlock()
	srv.mu.Lock()
	delete(srv.conns, conn)
}

// Shutdown initiates the shutdown process
func (srv *Server) Shutdown() {
	srv.inShutDown = true
	log.Println("shutting down...")
	srv.listener.Close()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			log.Printf("waiting on %v connections", len(srv.conns))
		}
		if len(srv.conns) == 0 {
			return
		}
	}
}
