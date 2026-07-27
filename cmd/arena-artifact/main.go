// Command arena-artifact validates arena-bundle archives for local CI.
package main

import (
	"fmt"
	"os"

	"github.com/yoskeoka/ai-arena/artifactbundle"
)

func main() {
	if len(os.Args) != 3 || os.Args[1] != "validate" {
		fmt.Fprintln(os.Stderr, "usage: arena-artifact validate <bundle.zip>")
		os.Exit(2)
	}
	// #nosec G304,G703 -- this CLI intentionally validates the user-supplied bundle path.
	data, err := os.ReadFile(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	bundle, err := artifactbundle.Read(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(bundle.Digest)
}
