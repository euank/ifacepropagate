.PHONY: all gen-tests test clean

all:
	go build -o ifacepropagate ./cmd

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

gen-tests: all
	cd ./tests/case01 && \
		$(ROOT_DIR)/ifacepropagate \
		ifacepropogate.testcase/test01 \
		"r readFrobulator.Reader" \
		ifacepropogate.testcase/test01/pkg.Frobulator \
		> ./case_gen.go
	cd ./tests/case01 && \
		$(ROOT_DIR)/ifacepropagate \
		ifacepropogate.testcase/test01 \
		"r *ptrReadFrobulator.Reader" \
		ifacepropogate.testcase/test01/pkg.Frobulator \
		> ./case_gen2.go

test:
	cd ./tests/case01 && go test ./...

clean:
	rm -f ./ifacepropagate
