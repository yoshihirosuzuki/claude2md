.PHONY: build test update-expected vet fmt clean

build:
	go build -o bin/claude2md ./cmd/claude2md

test:
	go test ./...

update-expected:
	go test ./internal/render -update

vet:
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -rf bin
