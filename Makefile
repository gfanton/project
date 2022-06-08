go_files := $(shell find . -type f -name '*.go')
go_bin := $(GOBIN)/project

build: $(go_files)
	@mkdir -p ./build
	go build -o ./build/project
.PHONY: build

install: $(go_bin)
$(go_bin): $(go_files)
	@mkdir -p ./build
	go install .
.PHONY: install
