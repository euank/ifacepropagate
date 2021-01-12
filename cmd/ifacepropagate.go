package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/euank/ifacepropagate/pkg/ifacepropagate"
	"golang.org/x/tools/go/packages"
)

// Code partly inspired by https://github.com/josharian/impl

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  ifacepropagate [package] [struct] [interfaces] > out_generated.go

ifacepropagate generates code to allow 'propogating' interface implementations
up from an embedded interface.
More specifically, it generates a '[struct].propogateInterfaces()' method which
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

  interfaces  The list of interfaces to "propogate" up, comma separated.
              For example 'syscall.Conn,io.Reader,net.Conn'.


`)
}

func main() {
	args := os.Args

	if len(args) != 4 {
		usage()
		os.Exit(1)
	}
	pkgSel, ifaceSel, ifacesList := args[1], args[2], args[3]
	ifaces := strings.Split(ifacesList, ",")

	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedTypes | packages.NeedName | packages.NeedSyntax | packages.NeedImports,
	}, pkgSel)
	if err != nil {
		log.Fatalf("error loading pkg %q: %v", pkgSel, err)
	}
	if len(pkgs) != 1 {
		log.Fatalf("multiple packages found, but we needed to load only one package: %v", pkgs)
	}
	pkg := pkgs[0]

	ret, err := ifacepropagate.PropogateInterfaces(
		pkg, "propogateInterfaces", ifaceSel, ifaces,
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(ret)
	os.Exit(0)
}
