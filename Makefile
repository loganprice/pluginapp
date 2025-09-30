.PHONY: build build-app build-plugins clean

# Add a target to run the remote test plugin
.PHONY: start-remote-plugin

all: build

build: clean
	@mkdir -p bin
	go build -o bin/plugin-app cmd/main/main.go
	go build -o bin/hello plugins/hello/main.go
	go build -o bin/addition plugins/addition/main.go

clean:
	@echo "Cleaning up..."
	@rm -rf ./bin

start-remote-plugin:
	@echo "Starting the remote test plugin on port 50055..."
	@go run ./plugins/remote_test_plugin/main.go --port 50055

run: build
	./bin/plugin-app 