package example

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

func ExampleConn() {
	srv, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("error serving: %v", err)
	}
	addr := srv.Addr()
	tcpConn, err := net.Dial("tcp", addr.String())
	if err != nil {
		log.Fatalf("error serving: %v", err)
	}
	// Wrapping a TCP connection should give you a readerfrom
	conn := newLoggedConn(log.New(os.Stdout, "", log.Lshortfile), tcpConn)

	_, readerFrom := conn.(io.ReaderFrom)
	if readerFrom {
		fmt.Println("wrapped tcpconn does implement io.ReadFrom")
	} else {
		fmt.Println("wrapped tcpconn does not implement io.ReadFrom")
	}

	pipeConn, _ := net.Pipe()
	// Wrapping a connection that doesn't support readerfrom, so the wrapped conn shouldn't either
	conn = newLoggedConn(log.New(os.Stdout, "", log.Lshortfile), pipeConn)
	_, readerFrom = conn.(io.ReaderFrom)
	if readerFrom {
		fmt.Println("wrapped pipeconn does implement io.ReadFrom")
	} else {
		fmt.Println("wrapped pipeconn does not implement io.ReadFrom")
	}

	// Output:
	// wrapped tcpconn does implement io.ReadFrom
	// wrapped pipeconn does not implement io.ReadFrom
}
