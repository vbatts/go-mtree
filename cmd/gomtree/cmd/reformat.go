package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	cli "github.com/urfave/cli/v2"
	"github.com/vbatts/go-mtree"
)

func NewMtree2JsonCommand() *cli.Command {
	return &cli.Command{
		Name:    "mtree2json",
		Usage:   "represent an mtree spec (or walked directory) as JSON",
		Aliases: []string{"m2j"},
		Description: `Converts an mtree hierarchy spec to a JSON array of entry objects.

This is a go-mtree extension and is not compatible with BSD mtree(8).

Each entry in the output array has the form:

  {
    "path":     "<full relative path>",
    "keywords": { "<keyword>": "<value>", ... }
  }

Keyword values are strings matching the mtree spec representation.
/set defaults are resolved per-entry (AllKeys semantics).

Input: -f <spec-file>, -p <dir> to walk, or stdin when neither is given.
Keyword filtering: -k (use only), -K (add), -R (remove).`,
		Action: mtree2JsonAction,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Usage:   "mtree spec file to convert (\"-\" for stdin)",
			},
			&cli.StringFlag{
				Name:    "path",
				Aliases: []string{"p"},
				Usage:   "root directory to walk and represent as JSON (mutually exclusive with -f)",
			},
			&cli.StringFlag{
				Name:    "use-keywords",
				Aliases: []string{"k"},
				Usage:   "comma/space-separated list of keywords to include (default: all from spec)",
			},
			&cli.StringFlag{
				Name:    "add-keywords",
				Aliases: []string{"K"},
				Usage:   "comma/space-separated keywords to add to the keyword set",
			},
			&cli.StringFlag{
				Name:    "remove-keywords",
				Aliases: []string{"R"},
				Usage:   "comma/space-separated keywords to remove from the keyword set (\"all\" removes all)",
			},
		},
	}
}

// jsonEntry is the per-file record written to the JSON output.
type jsonEntry struct {
	Path     string            `json:"path"`
	Keywords map[string]string `json:"keywords"`
}

func mtree2JsonAction(c *cli.Context) error {
	if c.String("file") != "" && c.String("path") != "" {
		return fmt.Errorf("-f and -p are mutually exclusive")
	}

	var (
		dh  *mtree.DirectoryHierarchy
		err error
	)

	switch {
	case c.String("path") != "":
		keywords := buildKeywords(c)
		if len(keywords) == 0 {
			keywords = mtree.DefaultKeywords[:]
		}
		dh, err = mtree.Walk(c.String("path"), nil, keywords, nil)
		if err != nil {
			return err
		}

	default:
		// -f or stdin
		var r io.Reader
		if c.String("file") == "" || c.String("file") == "-" {
			r = os.Stdin
		} else {
			fh, ferr := os.Open(c.String("file"))
			if ferr != nil {
				return ferr
			}
			defer fh.Close()
			r = fh
		}
		dh, err = mtree.ParseSpec(r)
		if err != nil {
			return err
		}
	}

	// If no keyword flags were given, default to all keywords in the DH.
	filterKws := buildKeywords(c)
	if len(filterKws) == 0 {
		filterKws = dh.UsedKeywords()
	}

	var entries []jsonEntry
	for _, e := range dh.Entries {
		if e.Type != mtree.RelativeType && e.Type != mtree.FullType {
			continue
		}
		fp, perr := e.Path()
		if perr != nil {
			return perr
		}
		kwmap := make(map[string]string)
		for _, kv := range e.AllKeys() {
			if mtree.InKeywordSlice(kv.Keyword(), filterKws) {
				kwmap[string(kv.Keyword())] = kv.Value()
			}
		}
		entries = append(entries, jsonEntry{Path: fp, Keywords: kwmap})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(entries)
}

// buildKeywords assembles the keyword list from -k/-K/-R flags.
// Returns nil when no keyword flags were given.
func buildKeywords(c *cli.Context) []mtree.Keyword {
	if c.String("use-keywords") == "" && c.String("add-keywords") == "" && c.String("remove-keywords") == "" {
		return nil
	}
	var kws []mtree.Keyword
	if c.String("use-keywords") != "" {
		kws = splitKeywordsArg(c.String("use-keywords"))
	} else {
		kws = mtree.DefaultKeywords[:]
	}
	for _, kw := range splitKeywordsArg(c.String("add-keywords")) {
		if !mtree.InKeywordSlice(kw, kws) {
			kws = append(kws, kw)
		}
	}
	if c.String("remove-keywords") != "" {
		removeKws := splitKeywordsArg(c.String("remove-keywords"))
		if mtree.InKeywordSlice("all", removeKws) {
			return []mtree.Keyword{}
		}
		filtered := kws[:0:0]
		for _, kw := range kws {
			if !mtree.InKeywordSlice(kw, removeKws) {
				filtered = append(filtered, kw)
			}
		}
		kws = filtered
	}
	return kws
}
