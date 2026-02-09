package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
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

func Get(command *cobra.Command, args []string) {
	conns, err := getNamedPipe()
	if err != nil {
		cobra.CheckErr(err)
	}

	if jsonVar {
		jsonFormat, err := marshallConnectionsJSON(conns)
		if err != nil {
			cobra.CheckErr(err)
		}
		fmt.Fprint(os.Stdout, jsonFormat)
		return
	}
	textFormat := formatConnections(conns)
	fmt.Fprint(os.Stdout, textFormat)
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().BoolVarP(&jsonVar, "json", "j", false, "Show output in JSON format")
}

func getNamedPipe() ([]Connection, error) {
	tempDir, err := internals.GetNPipePathCurDir()
	if err != nil {
		return []Connection{}, err
	}
	nPipePath := path.Join(tempDir, internals.NPipeName)
	ok, err := internals.Exists(nPipePath)
	if err != nil {
		return []Connection{}, err
	}
	if !ok {
		return []Connection{}, nil
	}
	return []Connection{{Type: "Named Pipe", Address: nPipePath}}, nil
}

func formatConnections(conns []Connection) string {
	if len(conns) == 0 {
		return "There are no connections at this location\n"
	}
	format := func(conn Connection) string {
		return fmt.Sprintf("  Type:    %s\n  Address: %s\n", conn.Type, conn.Address)
	}

	connStr := []string{}
	verb := "is"
	name := "connection"
	if len(conns) > 1 {
		verb = "are"
		name = "connections"
	}
	header := fmt.Sprintf("There %s %d %s at this location\n", verb, len(conns), name)
	connStr = append(connStr, header)
	for _, conn := range conns {
		connStr = append(connStr, format(conn))
	}

	return strings.Join(connStr, "\n")
}

func marshallConnectionsJSON(conns []Connection) (string, error) {
	res, err := json.MarshalIndent(conns, "", "  ")
	if err != nil {
		return "", err
	}
	return string(res), nil
}
