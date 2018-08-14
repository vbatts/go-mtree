// +build ignore

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	flag.Parse()
	for _, arg := range flag.Args() {
		path, err := filepath.Abs(arg)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("%s", path)
	}
}
