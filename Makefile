BIN ?= $(HOME)/.local/bin

.PHONY: all test build clean

all: test build

test:
	go test ./...

build:
	go build -o $(BIN)/md2html ./cmd/md2html/
