BINARY := taskrunner
export GOCACHE := $(CURDIR)/.gocache

.PHONY: build test run lint

build:
	go build -buildvcs=false -o $(BINARY) ./cmd/taskrunner

test:
	go test ./...

run:
	go run ./cmd/taskrunner -file tasks.json -workers 3 -verbose

lint:
	go vet ./...
	test -z "$$(gofmt -l .)"
