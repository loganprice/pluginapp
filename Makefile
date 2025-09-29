.PHONY: all build clean

all: build

build: clean
	@mkdir -p bin
	go build -o bin/plugin-app cmd/main/main.go
	go build -o bin/hello plugins/hello/main.go
	go build -o bin/addition plugins/addition/main.go

clean:
	@rm -rf bin/

run: build
	./bin/plugin-app 