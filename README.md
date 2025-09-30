# Go gRPC Plugin Framework

This project is a framework for building applications that can be extended with plugins written in different languages. It uses gRPC for robust and safe inter-process communication between the main application and its plugins.

This example demonstrates support for plugins written in both Go and Python.

## Features

- **Extensible Plugin Architecture**: Easily add new functionality by creating new plugins.
- **Language Agnostic**: Supports plugins written in any language that supports gRPC. (Go and Python examples included).
- **Process Isolation**: Plugins run as separate processes, preventing a plugin crash from taking down the main application.
- **Structured Communication**: gRPC and Protocol Buffers provide a clearly defined interface for plugin communication.
- **Simple CLI**: A command-line interface for listing, inspecting, and running plugins.

## Prerequisites

- Go (version 1.18+ recommended)
- Python (version 3.x recommended)
- `make`

## Building

You can build the main application and the included plugins using the provided `Makefile`.

- **Build everything (app + plugins):**
  ```sh
  make build
  ```

- **Build only the main application:**
  ```sh
  make build-app
  ```

- **Build only the plugins:**
  ```sh
  make build-plugins
  ```

The main application binary will be placed at `./bin/app`, and plugin binaries at `./bin/<plugin-name>`.

## Usage

The application is controlled via the `./bin/app` CLI.

### List Available Plugins

To see a list of all plugins registered in the configuration:

```sh
./bin/app list
```

### Get Plugin Information & Help

To see detailed information about a specific plugin, including its parameters and a usage example, use the `info` command or the `--help` flag on the `run` command.

```sh
# Using the info command
./bin/app info hello

# Using the --help flag
./bin/app run hello --help
```

This will output the plugin's description, parameters, defaults, and a usage string.

### Run a Plugin

To execute a plugin, use the `run` command, followed by the plugin name and any parameters using standard flag syntax.

```sh
# Run the 'hello' plugin with default parameters
./bin/app run hello

# Run the 'hello' plugin with a custom message
./bin/app run hello --message World

# Run the 'addition' plugin with custom numbers
./bin/app run addition --num1 100 --num2 200

# Run the Python 'multiply' plugin
./bin/app run python-multiply --num1 5 --num2 10
```

## Configuration

Plugins are registered in the `config.json` file. Each plugin entry defines how the main application should run it.

There are three types of plugins:

- `binary`: A self-contained executable. The application runs it directly.
- `command`: A script that needs an interpreter. The application uses the `command` template to execute it.
- `remote`: A plugin already running on a remote server. The application connects to it directly via its address.

**Example `config.json`:**

```json
{
  "plugins": {
    "hello": {
      "type": "binary",
      "path": "./bin/hello",
      "description": "A simple greeting plugin"
    },
    "python-multiply": {
      "type": "command",
      "path": "./plugins/multiply/multiply.py",
      "command": "python3 {path} --port {port}",
      "description": "A Python multiplication plugin",
      "workdir": "./plugins/multiply"
    },
    "remote-plugin-example": {
      "type": "remote",
      "address": "localhost:50055",
      "description": "An example of a remote plugin"
    }
  }
}
```

## Testing Remote Plugins

A test plugin server is included to demonstrate the remote plugin functionality. You can run it using the `make` target.

**1. Start the remote plugin server**

In one terminal, run the following command. The server will start and listen on port `50055`.

```sh
make start-remote-plugin
```

**2. Run the remote plugin**

In another terminal, you can now use the main application to execute the remote plugin.

```sh
./bin/app run remote-plugin-example --message "Hello from a remote plugin!"
```

## How to Create a Plugin

1.  **Implement the gRPC Service**: Create a gRPC server that implements the `Plugin` service defined in `proto/plugin.proto`. The service definition is:

    ```proto
    service Plugin {
      rpc GetInfo(google.protobuf.Empty) returns (PluginInfo);
      rpc Execute(ExecuteRequest) returns (stream ExecuteResponse);
      rpc ReportExecutionSummary(ExecutionSummary) returns (google.protobuf.Empty);
    }
    ```

    - `GetInfo`: Should return information about the plugin, such as its name and the parameters it accepts.
    - `Execute`: The main logic of the plugin. It receives parameters and can stream back output, progress, or errors.
    - `ReportExecutionSummary`: Called by the host after `Execute` finishes to provide a summary of the execution.

2.  **Examine the Examples**: The `plugins/addition` (Go) and `plugins/multiply` (Python) directories provide working examples of how to implement this interface.

3.  **Add to `config.json`**: Add a new entry for your plugin in the `config.json` file, specifying its `type` and `path`.

4.  **Build Your Plugin**: If your plugin is a compiled language, make sure it's built and the path in the config is correct. Consider adding it to the `Makefile`.
