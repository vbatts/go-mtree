package main

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
	"github.com/vbatts/go-mtree/cmd/gomtree/cmd"
)

var Version string

func main() {
	app := cli.NewApp()
	app.Name = "gomtree"
	app.Version = Version
	app.Usage = "map a directory hierarchy"
	app.Description = `The gomtree utility compares the file hierarchy rooted in
the current directory against a specification read from file or standard input.
Messages are written to the standard output for any files whose characteristics
do not match the specification, or which are missing from either the file
hierarchy or the specification.

This tool is written in likeness to the BSD MTREE(6), with notable additions
to support xattrs and interacting with tar archives.`
	// cli docs --> https://github.com/urfave/cli/blob/master/docs/v2/manual.md
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Output debug info to STDERR",
			Value: false,
			Action: func(ctx *cli.Context, b bool) error {
				if b {
					os.Setenv("DEBUG", "1")
					logrus.SetLevel(logrus.DebugLevel)
				}
				return nil
			},
		},
	}
	app.Commands = []*cli.Command{
		cmd.NewValidateCommand(),
	}

	// Unfortunately urfave/cli is not at good at using DefaultCommand
	// as if you run the cli without a command but with flags like gomtree -K`
	// it fails horribly with a "flag provided but not defined" error.
	runValidate := false
	app.OnUsageError = func(ctx *cli.Context, err error, isSubcommand bool) error {
		if ctx.Command.Name == "gomtree" && strings.Contains(err.Error(), "flag provided but not defined") {
			runValidate = true
		}
		return err
	}

	err := app.Run(os.Args)
	// If it failed, run the command again with the validate command as the
	// default if it failed.
	if err != nil && runValidate {
		app.OnUsageError = nil
		args := []string{os.Args[0], "validate"}
		args = append(args, os.Args[1:]...)
		err = app.Run(args)
	}
	if err != nil {
		logrus.Fatal(err)
	}
}
