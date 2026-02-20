# gorepl

A REPL multiplexer that lets you spin up any REPL and send commands to it from
multiple terminals in the same directory.

Start a Python, Node, or any interactive shell in one terminal, then pipe
commands into it from others - all inputs are multiplexed into a single REPL
session, and output is echoed back to the server terminal.

## Installation

### From source

```sh
go install github.com/azr4e1/gorepl@latest
```

### From releases

Download the latest binary from the
[releases page](https://github.com/azr4e1/gorepl/releases).

## Quick start

**Terminal 1** -- start a Python REPL server:

```sh
gorepl run python3
```

**Terminal 2** -- send commands to it (from the same directory):

```sh
gorepl send "print('hello from another terminal')"
```

The output appears in Terminal 1.

You can also pipe input:

```sh
echo "1 + 1" | gorepl send
```

## Commands

### `gorepl run <command>`

Start a REPL process and listen for incoming connections.

```
Usage:
  gorepl run [flags] <command>

Flags:
  -c, --connection <type>   Connection type: uds, tcp, namedpipe (default "uds")
  -p, --port <port>         Port for TCP connections (default 4501)
  -f, --force               Force start even if a connection already exists
  -l, --log <path>          Path to log file (default ~/.cache/gorepl/<timestamp>-logs)
```

**Examples:**

```sh
# Start with default Unix domain socket
gorepl run python3

# Start with TCP on a specific port
gorepl run -c tcp -p 5000 python3

# Start a Node.js REPL with named pipe
gorepl run -c namedpipe node

# Force restart, replacing an existing session
gorepl run -f python3
```

### `gorepl send [input]`

Send a command to a running REPL in the current directory.

```
Usage:
  gorepl send [flags] [input...]

Flags:
  -c, --connection <type>   Connection type: uds, tcp, namedpipe (default "uds")
  -p, --port <port>         Port for TCP connections (default -1, auto-detect)
  -a, --address <addr>      Address for UDS or named pipe connections
```

**Examples:**

```sh
# Send a single command
gorepl send "print('hello')"

# Pipe a script
cat script.py | gorepl send

# Send to a TCP REPL on a specific port
gorepl send -c tcp -p 5000 "console.log('hi')"

# Send to a specific UDS address
gorepl send -a /tmp/gorepl_abc123_unix_socket "1 + 1"
```

### `gorepl get`

Show active REPL connections.

```
Usage:
  gorepl get [flags]

Flags:
  -a, --all    Show all connections (not just current directory)
  -j, --json   Output in JSON format
```

**Examples:**

```sh
# Show connections in the current directory
gorepl get

# Show all connections across the system
gorepl get --all

# Machine-readable output
gorepl get --all --json
```

## Connection types

gorepl supports three connection methods. All are directory-scoped by default --
the REPL server and client must be in the same working directory to auto-discover
each other.

| Type           | Flag           | Description                                               |
|----------------|----------------|-----------------------------------------------------------|
| **UDS**        | `-c uds`       | Unix domain socket (default). Fast, local-only.           |
| **TCP**        | `-c tcp`       | TCP socket on localhost. Useful when UDS isn't available. |
| **Named Pipe** | `-c namedpipe` | FIFO-based. Single writer at a time.                      |

## How it works

```
Terminal 2               Terminal 3
gorepl send "x = 1"     gorepl send "print(x)"
      |                        |
      +--- socket/pipe --------+
                |
          [ Multiplexer ]
                |
          [ REPL process ]  <-- Terminal 1: gorepl run python3
                |
           stdout/stderr  --> displayed in Terminal 1
```

1. `gorepl run` starts the target REPL process and opens a socket (or named
   pipe) in a well-known location derived from the current directory.
2. `gorepl send` connects to that socket and writes the input.
3. A multiplexer merges input from all connected clients and the server's own
   stdin, forwarding everything to the REPL's stdin.
4. REPL output (stdout/stderr) is displayed in the server terminal. Inputs from
   remote clients are echoed to the server terminal as well.

Connection addresses are stored in the system temp directory (`/tmp`) using a
SHA-256 hash of the working directory path, so each directory gets its own
isolated REPL session.

## Shell completion

gorepl supports shell completions for connection types, ports, and addresses via
cobra's built-in completion system:

```sh
# Bash
gorepl completion bash > /etc/bash_completion.d/gorepl

# Zsh
gorepl completion zsh > "${fpath[1]}/_gorepl"

# Fish
gorepl completion fish > ~/.config/fish/completions/gorepl.fish
```

## Logs

By default, logs are written to `~/.cache/gorepl/` with a timestamped filename.
Use `--log` to specify a custom path:

```sh
gorepl run --log /tmp/my-repl.log python3
```

## Building

```sh
git clone https://github.com/azr4e1/gorepl.git
cd gorepl
go build -o gorepl .
```

## License

See [LICENSE](LICENSE) for details.
