// Command archlint checks architectural fitness functions.
package main

import (
	"fmt"
	"os"

	"minimal-dep/internal/linter/architecture"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	violations, err := architecture.Lint(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	for _, violation := range violations {
		fmt.Println(violation)
	}
	if len(violations) > 0 {
		fmt.Fprintf(os.Stderr, "archlint: %d violation(s)\n", len(violations))
		os.Exit(1)
	}
	fmt.Println("archlint: ok")
}
