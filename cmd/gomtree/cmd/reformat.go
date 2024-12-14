package cmd

import (
	cli "github.com/urfave/cli/v2"
)

func NewMtree2JsonCommand() *cli.Command {
	return &cli.Command{
		Name:    "mtree2json,",
		Usage:   "represent an mtree as JSON format",
		Aliases: []string{"m2j"},
		Action:  validateAction,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Usage:   "Directory hierarchy spec to validate",
			},
		},
	}
}
