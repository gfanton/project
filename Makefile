template_files := $(shell find . -type f -name '*.init')
go_files := $(shell find . -type f -name '*.go')
go_bin := $(GOBIN)/project

lint:
	go vet -v ./...

test:
	go test -v ./...


build: $(go_files) go.mod Makefile
	@mkdir -p ./build
	go build -o ./build/project
.PHONY: build

install: $(go_bin)
$(go_bin): $(go_files) $(template_files) go.mod Makefile 
	@mkdir -p ./build
	go install -v .
.PHONY: install
