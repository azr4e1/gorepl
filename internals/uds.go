package internals

import "strings"

const UDSName = "unix_socket"

func GenerateUDSPath(tempPath string) string {
	return strings.Join([]string{tempPath, UDSName}, "_")
}
