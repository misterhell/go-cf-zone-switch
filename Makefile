# Simple Makefile for building and running the cmd/app Go application

APP_NAME=app
APP_PATH=./cmd/app
BUILD_DIR=bin
BINARY=$(BUILD_DIR)/$(APP_NAME)

.PHONY: build start clean

build:
	mkdir -p bin
	go build -o $(BINARY) $(APP_PATH)

start: build
	./$(BINARY)

clean:
	rm -f $(BINARY)
