// Package childdemo is a small standard-library child used by local tests and
// manual executor demonstrations. It is intentionally not part of MayFly's
// runtime execution path.
package childdemo

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
)

// Run executes one deterministic demo operation and returns its desired
// process exit code.
func Run(args []string, output io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(output, "usage: mayfly-child env|args|stdin|exit")
		return 2
	}
	switch args[0] {
	case "env":
		values := make([]string, 0, len(args)-1)
		for _, name := range args[1:] {
			values = append(values, os.Getenv(name))
		}
		if err := json.NewEncoder(output).Encode(values); err != nil {
			return 1
		}
		return 0
	case "args":
		if err := json.NewEncoder(output).Encode(args[1:]); err != nil {
			return 1
		}
		return 0
	case "stdin":
		if _, err := io.Copy(output, os.Stdin); err != nil {
			return 1
		}
		return 0
	case "exit":
		if len(args) != 2 {
			return 2
		}
		code, err := strconv.Atoi(args[1])
		if err != nil || code < 0 || code > 125 {
			return 2
		}
		return code
	default:
		_, _ = fmt.Fprintln(output, "unknown demo operation")
		return 2
	}
}
