.PHONY: all gen-tests test clean

all:
	go build -o ifacepropagate ./cmd/ifacepropagate

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

gen-tests: all
	cd ./tests/case01 && \
		$(ROOT_DIR)/ifacepropagate \
		ifacepropagate.testcase/test01 \
		"r readFrobulator.Reader" \
		ifacepropagate.testcase/test01/pkg.Frobulator \
		> ./case_gen.go
	cd ./tests/case01 && \
		$(ROOT_DIR)/ifacepropagate \
		ifacepropagate.testcase/test01 \
		"r *ptrReadFrobulator.Reader" \
		ifacepropagate.testcase/test01/pkg.Frobulator \
		> ./case_gen2.go
	cd ./tests/case02 && \
		$(ROOT_DIR)/ifacepropagate \
		ifacepropagate.testcase/case02 \
		"p *partialOverride.If1" \
		If2 \
		> ./case_gen.go


test:
	cd ./tests/case01 && go test ./...
	cd ./tests/case02 && go test ./...

clean:
	rm -f ./ifacepropagate
