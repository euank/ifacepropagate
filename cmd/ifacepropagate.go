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
ifacepropagate [package] [xxx] ...
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
