.PHONY: all gen-tests

all:
	go build -o ifacepropagate ./cmd

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

gen-tests: all
	cd ./tests/cases/case01 && \
		$(ROOT_DIR)/ifacepropagate \
		ifacepropogate.testcase/test01 \
		"r readCloseFrobulator.Reader" \
		io.Reader,ifacepropogate.testcase/test01/pkg.Frobulator \
		> ./case_gen.go

clean:
	rm -f ./ifacepropagate
