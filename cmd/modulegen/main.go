// Command modulegen creates a new feature module skeleton.
package main

import (
	"flag"
	"fmt"
	"os"

	"minimal-dep/internal/generator/module"
)

func main() {
	name := flag.String("name", "", "module name (lowercase letters, digits, underscores)")
	root := flag.String("root", ".", "repository root")
	flag.Parse()

	guidance, err := module.Generate(*root, *name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(guidance)
}
