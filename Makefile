.PHONY: all run test eval demo-data build clean

# Default binary name
BINARY=munim

all: build

build:
	go build -o $(BINARY) ./cmd/munim

run: build
	./$(BINARY)

test:
	go test -v ./...

demo-data:
	go run ./cmd/gen-demo-data

eval:
	go run ./cmd/eval -demo-dir demo_data

eval-quick:
	go run ./cmd/eval -demo-dir demo_data -limit 5

clean:
	rm -f $(BINARY)
	rm -rf data/bills/*
