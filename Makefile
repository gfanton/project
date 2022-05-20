go_files := $(shell find . -type f -name '*.go')

build: $(go_files)
	@mkdir -p ./build
	go build -o ./build/project
