BINARY := dsync.exe
PKG := ./...

.PHONY: build run test vet tidy clean

build:
	go build -o $(BINARY) .

run: build
	./$(BINARY)

test:
	go test $(PKG)

vet:
	go vet $(PKG)

tidy:
	go mod tidy

clean:
	rm -f $(BINARY)
