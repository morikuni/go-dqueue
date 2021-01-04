.PHONY: test
test:
	go test -race -v -count 1 ./...

.PHONY: bench
bench:
	go test -bench . -benchmem -count 1 -benchtime 10s ./...

.PHONY: coverage
coverage:
	go test -v -race -count 1 -covermode=atomic -coverprofile=coverage.out ./...