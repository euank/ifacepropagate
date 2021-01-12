package example

import (
	"fmt"
	"log"
	"net"
)

func newLoggedConn(l *log.Logger, conn net.Conn) net.Conn {
	return (&closeLoggedConn{conn, l}).propagateInterfaces()
}

type closeLoggedConn struct {
	net.Conn
	l *log.Logger
}

func (c *closeLoggedConn) Close() error {
	err := c.Conn.Close()
	fmt.Println("connection closed")
	return err
}
