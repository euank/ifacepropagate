## interfacepropagate - Generate go interface wrappers, while propagating other interfaces

This is a bit of codegen to hack around the problem described in [this blog
post](https://medium.com/@cep21/interface-wrapping-method-erasure-c523b3549912).

In Go, wrapping an interface to augment it with functionality is common. Unfortunately, if the interface being wrapped implements other interfaces, it's fraught with peril.

This program allows generating some code to improve the situation somewhat.

### Usage

```
Usage:
  ifacepropagate [package] [struct] [interfaces] > out_generated.go

ifacepropagate generates code to allow 'propagating' interface implementations
up from an embedded interface.
More specifically, it generates a '[struct].propagateInterfaces()' method which
returns a concrete type that implements only the interfaces in the [interfaces]
list that the embedded type implemented.

This is useful to, for example, embed a ResponseWriter, and then return a
ResponseWriter which only implements http.Hijacker if the inner response writer
did.

ARGS:
  package     The go package which contains your struct that embeds an
              interface such as 'github.com/user/project/pkg/type'.
              The following struct must be in this package, and this package
              should already compile.

  struct      A specifier for the struct that contains an embedded interface
               which we're wrapping. For example "s *MyStruct.Conn" if the
              struct is named 'MyStruct', has a pointer receiver, and is
              embedding a 'net.Conn' interface.

  interfaces  The list of interfaces to "propagate" up, comma separated.
              For example 'syscall.Conn,io.Reader,net.Conn'.
```

See also the example below


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
interfacepropagate my.go.package/path/logconn "l *logWritesConn.Conn" io.ReaderFrom,syscall.Conn > igen_generated.go
```

and then updating the 'New' function above like so:

```
func NewLogWritesConn(c net.Conn, l *log.Logger) net.Conn {
	return (&logWritesConn{c, l}).propagateInterfaces()
}
```
