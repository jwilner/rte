.PHONY: test test-cover bench gen check lint fix

MAX-VARS := 8

test:
	go test ./...

test-cover:
	go test -v -covermode=count -coverprofile=cover.out

bench:
	go test -test.bench=. ./...

gen:
	go run ./internal/cmd/rte-gen \
		-max-vars ${MAX-VARS} \
		-output internal/funcs/funcs.go \
		-test-output internal/funcs/funcs_test.go

check:
	devscripts/check.sh ${MAX-VARS}

lint:
	@golangci-lint run

fix:
	@golangci-lint run --fix
