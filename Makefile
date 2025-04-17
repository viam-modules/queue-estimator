MOD_ARCH := $(shell uname -m)
MOD_OS := $(shell uname -s)

test:
		go test ./waitsensor/

lint:
		go mod tidy
		golangci-lint run

module.tar.gz: go.mod go.sum waitsensor/waitsensor.go cmd/module/main.go
	go build -a -o module ./cmd/module
	tar -czf $@ module

clean:
	rm -rf module module.tar.gz
