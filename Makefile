SHELL := /bin/bash

build:
	go build -v -o pal ./cmd

run:
	go run ./cmd/
