package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
)

func main() {
	flag.Parse()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	failed := 0
	for _, arg := range flag.Args() {
		cmd := exec.Command("bash", arg)
		if os.Getenv("TMPDIR") != "" {
			cmd.Env = append(cmd.Env, os.Environ()...)
			cmd.Env = append(cmd.Env, "TMPDIR="+os.Getenv("TMPDIR"))
		}
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			failed++
			fmt.Fprintf(os.Stderr, red("FAILED: %s\n"), arg)
		}
	}
	if failed > 0 {
		fmt.Fprintf(os.Stderr, red("%d FAILED tests\n"), failed)
		os.Exit(1)
	}
	fmt.Fprint(os.Stdout, green("SUCCESS: no cli tests failed\n"))
}
