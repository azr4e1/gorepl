package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"

	"github.com/azr4e1/gorepl/internals"
	"github.com/spf13/cobra"
)

type Connection struct {
	Type    string
	Address string
}

var (
	getCmd = &cobra.Command{
		Use:   "get",
		Short: "Get the active connection in this directory",
		Run:   Get,
		Args:  cobra.NoArgs,
	}
)

const (
	UDSType       = "Unix Domain Socket"
	NamedPipeType = "Named Pipe"
	TCPType       = "TCP"
)

func Get(command *cobra.Command, args []string) {
	output, err := getConnections()
	if err != nil {
		cobra.CheckErr(err)
	}
	fmt.Fprintln(os.Stdout, output)
}

func getConnections() (string, error) {
	connections := []Connection{}
	var namedPipeConns []Connection
	var tcpConns []Connection
	var udsConns []Connection
	var err error
	if allConnections {
		if runtime.GOOS != "windows" {
			namedPipeConns, err = getAllConnections("namedpipe")
			if err != nil {
				return "", err
			}
		} else {
			namedPipeConns = []Connection{}
		}
		tcpConns, err = getAllConnections("tcp")
		if err != nil {
			return "", err
		}
		udsConns, err = getAllConnections("uds")
		if err != nil {
			return "", err
		}
	} else {
		if runtime.GOOS != "windows" {
			namedPipeConns, err = getCurConnections("namedpipe")
			if err != nil {
				return "", err
			}
		}
		tcpConns, err = getCurConnections("tcp")
		if err != nil {
			return "", err
		}
		udsConns, err = getCurConnections("uds")
		if err != nil {
			return "", err
		}

	}
	connections = append(connections, namedPipeConns...)
	connections = append(connections, tcpConns...)
	connections = append(connections, udsConns...)

	if jsonVar {
		jsonFormat, err := marshallConnectionsJSON(connections)
		if err != nil {
			return "", err
		}
		return jsonFormat, nil
	}
	textFormat := formatConnections(connections)
	return textFormat, nil
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().BoolVarP(&jsonVar, "json", "j", false, "show output in JSON format")
	getCmd.Flags().BoolVarP(&allConnections, "all", "a", false, "show all connections")
}

func getAllConnections(connType string) ([]Connection, error) {
	var matchPattern string
	var Type string
	tempPath := os.TempDir()
	switch connType {
	case "uds":
		Type = UDSType
		matchPattern = fmt.Sprintf("gorepl_[a-zA-Z0-9]{78}_%s", internals.UDSName)
	case "namedpipe":
		Type = NamedPipeType
		matchPattern = fmt.Sprintf("gorepl_[a-zA-Z0-9]{78}_%s", internals.NPipeName)
	case "tcp":
		Type = TCPType
		matchPattern = fmt.Sprintf("gorepl_[a-zA-Z0-9]{78}_%s", internals.TCPName)
	default:
		return []Connection{}, errors.New("unknown type of connection")
	}
	tmpFiles, err := os.ReadDir(tempPath)
	if err != nil {
		return []Connection{}, err
	}
	connections := []Connection{}
	pattern := regexp.MustCompile(matchPattern)
	for _, dir := range tmpFiles {
		name := dir.Name()
		if pattern.Match([]byte(name)) {
			var Address string
			if Type == TCPType {
				fd, err := os.Open(path.Join(tempPath, name))
				if err != nil {
					continue
				}
				addressVal, err := io.ReadAll(fd)
				if err != nil {
					continue
				}
				Address = string(addressVal)
			} else {
				Address = path.Join(tempPath, name)
			}
			connection := Connection{Type: Type, Address: Address}
			connections = append(connections, connection)
		}
	}

	return connections, nil
}

func getCurConnections(connType string) ([]Connection, error) {
	var Type, Address string
	var err error
	switch connType {
	case "uds":
		Type = UDSType
		Address, err = internals.GenerateUDSPath()
		if err != nil {
			return []Connection{}, err
		}
		ok, err := internals.Exists(Address)
		if err != nil {
			return []Connection{}, err
		}
		if !ok {
			return []Connection{}, nil
		}
	case "namedpipe":
		Type = NamedPipeType
		Address, err = internals.GenerateNPipePath()
		if err != nil {
			return []Connection{}, err
		}
		ok, err := internals.Exists(Address)
		if err != nil {
			return []Connection{}, err
		}
		if !ok {
			return []Connection{}, nil
		}
	case "tcp":
		Type = TCPType
		Address, err = internals.RetrieveTCPAddress()
		if err != nil {
			return []Connection{}, nil
		}
	default:
		return []Connection{}, errors.New("unknown type of connection")
	}
	return []Connection{{Type: Type, Address: Address}}, nil
}

func formatConnections(conns []Connection) string {
	if len(conns) == 0 {
		return "There are no active connections"
	}
	format := func(conn Connection) string {
		return fmt.Sprintf("Connection:\n  Type:    %s\n  Address: %s", conn.Type, conn.Address)
	}

	connStr := []string{}
	verb := "is"
	name := "connection"
	if len(conns) > 1 {
		verb = "are"
		name = "connections"
	}
	header := fmt.Sprintf("There %s %d active %s", verb, len(conns), name)
	connStr = append(connStr, header)
	for _, conn := range conns {
		connStr = append(connStr, format(conn))
	}

	return strings.Join(connStr, "\n\n")
}

func marshallConnectionsJSON(conns []Connection) (string, error) {
	res, err := json.MarshalIndent(conns, "", "  ")
	if err != nil {
		return "", err
	}
	return string(res), nil
}
