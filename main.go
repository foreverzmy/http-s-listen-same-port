package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/valyala/fasthttp"
)

// here's a buffered conn for peeking into the connection
type PeekConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *PeekConn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

func (c *PeekConn) Peek(n int) ([]byte, error) {
	return c.r.Peek(n)
}

func newPeekConn(c net.Conn) *PeekConn {
	return &PeekConn{c, bufio.NewReader(c)}
}

type Listener struct {
	net.Listener
}

func (ln *Listener) Accept() (net.Conn, error) {
	conn, err := ln.Listener.Accept()
	if err != nil {
		return nil, err
	}

	peekConn := newPeekConn(conn)

	b, err := peekConn.Peek(3)
	if err != nil {
		peekConn.Close()
		if err != io.EOF {
			return nil, err
		}
	}

	if b[0] == 0x16 && b[1] == 0x03 && b[2] <= 0x03 {
		log.Println("HTTPS")
		return tls.Server(peekConn, &tls.Config{
			ClientAuth: tls.NoClientCert,
			GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				cert, err := tls.LoadX509KeyPair("server.cert.pem", "server.key.pem")
				if err != nil {
					return nil, err
				}
				return &cert, nil
			},
		}), nil
	}

	log.Println("HTTP")
	return peekConn, nil
}

func main() {
	ln, err := net.Listen("tcp", ":6789")
	if err != nil {
		panic(err)
	}

	fasthttp.Serve(&Listener{ln}, requestHandler)
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	log.Printf("%s %s %s %s\n", ctx.URI().Scheme(), ctx.Method(), ctx.Host(), ctx.Path())

	ctx.SetBodyString(fmt.Sprintf("hello from %s!", ctx.URI().Scheme()))
}
