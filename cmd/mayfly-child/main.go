// mayfly-child is a tiny local child-process demo for MayFly's executor.
package main

import (
	"os"

	"mayfly/internal/childdemo"
)

func main() {
	os.Exit(childdemo.Run(os.Args[1:], os.Stdout))
}
