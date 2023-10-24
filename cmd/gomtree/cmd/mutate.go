package cmd

import (
	"fmt"
	"io"
	"math"
	"os"
	"slices"
	"strings"

	cli "github.com/urfave/cli/v2"
	"github.com/vbatts/go-mtree"
)

func NewMutateCommand() *cli.Command {

	return &cli.Command{
		Name:  "mutate",
		Usage: "mutate an mtree",
		Description: `Mutate an mtree to have different shapes.
TODO: more info examples`,
		Action:    mutateAction,
		ArgsUsage: "<path to mtree> [path to output]",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name: "strip-prefix",
			},
			&cli.BoolFlag{
				Name:  "keep-comments",
				Value: false,
			},
			&cli.BoolFlag{
				Name:  "keep-blank",
				Value: false,
			},
			&cli.StringFlag{
				Name:      "output",
				TakesFile: true,
			},
		},
	}
}

func mutateAction(c *cli.Context) error {
	mtreePath := c.Args().Get(0)
	outputPath := c.Args().Get(1)
	stripPrexies := c.StringSlice("strip-prefix")
	keepComments := c.Bool("keep-comments")
	keepBlank := c.Bool("keep-blank")

	if mtreePath == "" {
		return fmt.Errorf("mtree path is required.")
	} else if outputPath == "" {
		outputPath = mtreePath
	}

	file, err := os.Open(mtreePath)
	if err != nil {
		return fmt.Errorf("opening %s: %w", mtreePath, err)
	}

	spec, err := mtree.ParseSpec(file)
	if err != nil {
		return fmt.Errorf("parsing mtree %s: %w", mtreePath, err)
	}

	stripPrefixVisitor := stripPrefixVisitor{
		prefixes: stripPrexies,
	}
	tidyVisitor := tidyVisitor{
		keepComments: keepComments,
		keepBlank:    keepBlank,
	}
	visitors := []Visitor{
		&stripPrefixVisitor,
		&tidyVisitor,
	}

	dropped := []int{}
	entries := []mtree.Entry{}

skip:
	for _, entry := range spec.Entries {

		if entry.Parent != nil && slices.Contains(dropped, entry.Parent.Pos) {
			if entry.Type == mtree.DotDotType {
				// directory for this .. has been dropped so shall this
				continue
			}
			entry.Parent = entry.Parent.Parent
			// TODO: i am not sure if this is the correct behavior
			entry.Raw = strings.TrimPrefix(entry.Raw, " ")
		}

		for _, visitor := range visitors {
			drop, err := visitor.Visit(&entry)
			if err != nil {
				return err
			}

			if drop {
				dropped = append(dropped, entry.Pos)
				continue skip
			}
		}

		entries = append(entries, entry)
	}

	spec.Entries = entries

	var writer io.Writer = os.Stdout
	if outputPath != "-" {
		writer, err = os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("creating output %s: %w", outputPath, err)
		}
	}

	spec.WriteTo(writer)

	return nil
}

type Visitor interface {
	Visit(entry *mtree.Entry) (bool, error)
}

type tidyVisitor struct {
	keepComments bool
	keepBlank    bool
}

func (m *tidyVisitor) Visit(entry *mtree.Entry) (bool, error) {
	if !m.keepComments && entry.Type == mtree.CommentType {
		return true, nil
	} else if !m.keepBlank && entry.Type == mtree.BlankType {
		return true, nil
	}
	return false, nil
}

type stripPrefixVisitor struct {
	prefixes []string
}

func (m *stripPrefixVisitor) Visit(entry *mtree.Entry) (bool, error) {
	if entry.Type != mtree.FullType && entry.Type != mtree.RelativeType {
		return false, nil
	}

	fp, err := entry.Path()
	if err != nil {
		return false, err
	}
	pathSegments := strings.Split(fp, "/")

	for _, prefix := range m.prefixes {

		prefixSegments := strings.Split(prefix, "/")
		minLen := int(math.Min(float64(len(pathSegments)), float64(len(prefixSegments))))
		matches := make([]string, minLen)
		for i := 0; i < minLen; i++ {
			if pathSegments[i] == prefixSegments[i] {
				matches[i] = prefixSegments[i]
			}
		}

		strip := strings.Join(matches, "/")
		if entry.Type == mtree.FullType {
			entry.Name = strings.TrimPrefix(entry.Name, strip)
			entry.Name = strings.TrimPrefix(entry.Name, "/")
			if entry.Name == "" {
				return true, nil
			}
		} else if fp == strip {
			return true, nil
		}
	}
	return false, nil
}
