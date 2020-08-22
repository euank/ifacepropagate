## interfacepropogate -- Generate go interface wrappers correctly

This is a bit of codegen to hack around the problem described in [this blog
post](https://medium.com/@cep21/interface-wrapping-method-erasure-c523b3549912).

In Go, wrapping an interface to augment it with functionality is common. Unfortunately, if the interface being wrapped implements other interfaces, it's fraught with peril.

This program allows generating some code to improve the situation somewhat.

### Usage

Just see the example, it's simpler to explain that way tbqh.

### Example

Let's say you wish to wrap a `net.Conn` such that 'Write' logs a debug message
each time it's called.

```go
type logWritesConn struct {
	net.Conn
	logger *log.Logger
}

func NewLogWritesConn(c net.Conn, l *log.Logger) net.Conn {
	return &logWritesConn{c, l}
}

func (l *logWritesConn) Write(b []byte) (int, error) {
	n, err := l.Conn.Write(b)
	l.logger.Printf("write occured: %v bytes, %v", n, err)
	return n, err
}
```

After wrapping a connection with this, you realize that calling
`NewLogWritesConn` with a `*net.TCPConn` returns a `net.Conn` that doesn't
implement `io.ReaderFrom`, even though the wrapped `TCPConn` does implement
`io.ReaderFrom`.

That's where this codegen comes in! Retaining those interfaces is a simple matter of generating some code to do so:

```
interfacepropogate my.go.package/path/logconn "l *logWritesConn.Conn" io.ReaderFrom,syscall.Conn > igen_generated.go
```

and then updating the 'New' function above like so:

```
func NewLogWritesConn(c net.Conn, l *log.Logger) net.Conn {
	return (&logWritesConn{c, l}).propogateInterfaces()
}
```
