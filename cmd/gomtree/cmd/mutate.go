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

	dropDotDot := 0
	droppedParents := []*mtree.Entry{}

	entries := []mtree.Entry{}

skip:
	for _, entry := range spec.Entries {

		if !keepComments && entry.Type == mtree.CommentType {
			continue
		}
		if !keepBlank && entry.Type == mtree.BlankType {
			continue
		}

		if entry.Parent != nil && slices.Contains(droppedParents, &entry) {
			entry.Parent = nil
			entry.Type = mtree.FullType
			entry.Raw = ""
		}

		if entry.Type == mtree.FullType || entry.Type == mtree.RelativeType {
			fp, err := entry.Path()
			// fmt.Println("fp", fp, entry.Name)
			if err != nil {
				return err
			}
			pathSegments := strings.Split(fp, "/")

			for _, prefix := range stripPrexies {

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
						continue skip
					}
				} else {
					if entry.IsDir() {
						dropDotDot++
						droppedParents = append(droppedParents, &entry)
					}
					if fp == strip {
						continue skip
					}
				}
			}
		} else if dropDotDot > 0 && entry.Type == mtree.DotDotType {
			dropDotDot--
			continue skip
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
