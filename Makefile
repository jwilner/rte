.PHONY: test gen check

test:
	go test -cover ./...

gen:
	go run cmd/rte-gen/*.go -output rte_func.go -test-output rte_func_test.go

check:
	devscripts/check.sh
