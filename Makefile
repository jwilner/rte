.PHONY: test clean build gen lint

test:
	go test -cover ./...

clean:
	rm -rf target/

build:
	mkdir -p target/
	go build -o target/rte-gen ./cmd/rte-gen

gen:
	go run cmd/rte-gen/*.go > rte_func.go

check:
	devscripts/check.sh
