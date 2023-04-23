SHELL := /bin/bash

build:
	go build -v -o pal ./
	chmod +rwx ./pal

run:
	go run ./
