.PHONY: test gen check

MAX-VARS := 8

test:
	go test -cover ./...

bench:
	go test -test.bench=. ./...

gen:
	go run ./internal/cmd/rte-gen \
		-max-vars ${MAX-VARS} \
		-output internal/funcs/funcs.go \
		-test-output internal/funcs/funcs_test.go

check:
	devscripts/check.sh ${MAX-VARS}
