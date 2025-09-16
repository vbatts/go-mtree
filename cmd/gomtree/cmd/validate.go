package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	cli "github.com/urfave/cli/v2"
	"github.com/vbatts/go-mtree"
)

func NewValidateCommand() *cli.Command {
	return &cli.Command{
		Name:   "validate",
		Usage:  "Create and validate a filesystem hierarchy (this is default tool behavior)",
		Action: validateAction,
		Flags: []cli.Flag{
			// Flags common with mtree(8)
			&cli.BoolFlag{
				Name:    "create",
				Aliases: []string{"c"},
				Usage:   "Create a directory hierarchy spec",
			},
			&cli.StringSliceFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Usage:   "Directory hierarchy spec to validate",
			},
			&cli.StringFlag{
				Name:    "path",
				Aliases: []string{"p"},
				Usage:   "Root path that the hierarchy spec is relative to",
			},
			&cli.StringFlag{
				Name:    "add-keywords",
				Aliases: []string{"K"},
				Usage:   "Add the specified (delimited by comma or space) keywords to the current set of keywords",
			},
			&cli.StringFlag{
				Name:    "use-keywords",
				Aliases: []string{"k"},
				Usage:   "Use only the specified (delimited by comma or space) keywords as the current set of keywords",
			},
			&cli.BoolFlag{
				Name:    "directory-only",
				Aliases: []string{"d"},
				Usage:   "Ignore everything except directory type files",
			},
			&cli.BoolFlag{
				Name:    "update-attributes",
				Aliases: []string{"u"},
				Usage:   "Modify the owner, group, permissions and xattrs of files, symbolic links and devices, to match the provided specification. This is not compatible with '-T'.",
			},

			// Flags unique to gomtree

			&cli.BoolFlag{
				Name:  "list-keywords",
				Usage: "List the keywords available",
			},
			&cli.BoolFlag{
				Name:  "list-used",
				Usage: "List all the keywords found in a validation manifest",
			},
			&cli.BoolFlag{
				Name:  "bsd-keywords",
				Usage: "Only operate on keywords that are supported by upstream mtree(8)",
			},
			&cli.StringFlag{
				Name:    "tar",
				Aliases: []string{"T"},
				Value:   "",
				Usage:   `Use tar archive to create or validate a directory hierarchy spec ("-" indicates stdin)`,
			},
			&cli.StringFlag{
				Name:  "result-format",
				Value: "bsd",
				Usage: "output the validation results using the given format (bsd, json, path)",
			},
		},
	}
}

func validateAction(c *cli.Context) error {
	// -list-keywords
	if c.Bool("list-keywords") {
		fmt.Println("Available keywords:")
		for k := range mtree.KeywordFuncs {
			fmt.Print(" ")
			fmt.Print(k)
			if mtree.Keyword(k).Default() {
				fmt.Print(" (default)")
			}
			if !mtree.Keyword(k).Bsd() {
				fmt.Print(" (not upstream)")
			}
			fmt.Print("\n")
		}
		return nil
	}

	// --result-format
	formatFunc, ok := formats[c.String("result-format")]
	if !ok {
		return fmt.Errorf("invalid output format: %s", c.String("result-format"))
	}

	var (
		err             error
		tmpKeywords     []mtree.Keyword
		currentKeywords []mtree.Keyword
	)

	// -k <keywords>
	if c.String("use-keywords") != "" {
		tmpKeywords = splitKeywordsArg(c.String("use-keywords"))
		if !mtree.InKeywordSlice("type", tmpKeywords) {
			tmpKeywords = append([]mtree.Keyword{"type"}, tmpKeywords...)
		}
	} else {
		if c.String("tar") != "" {
			tmpKeywords = mtree.DefaultTarKeywords[:]
		} else {
			tmpKeywords = mtree.DefaultKeywords[:]
		}
	}

	// -K <keywords>
	if c.String("add-keywords") != "" {
		for _, kw := range splitKeywordsArg(c.String("add-keywords")) {
			if !mtree.InKeywordSlice(kw, tmpKeywords) {
				tmpKeywords = append(tmpKeywords, kw)
			}
		}
	}

	// -bsd-keywords
	if c.Bool("bsd-keywords") {
		for _, k := range tmpKeywords {
			if mtree.Keyword(k).Bsd() {
				currentKeywords = append(currentKeywords, k)
			} else {
				fmt.Fprintf(os.Stderr, "INFO: ignoring %q as it is not an upstream keyword\n", k)
			}
		}
	} else {
		currentKeywords = tmpKeywords
	}

	// Check mutual exclusivity of keywords.
	// TODO(cyphar): Abstract this inside keywords.go.
	if mtree.InKeywordSlice("tar_time", currentKeywords) && mtree.InKeywordSlice("time", currentKeywords) {
		return fmt.Errorf("tar_time and time are mutually exclusive keywords")
	}

	// If we're doing a comparison, we always are comparing between a spec and
	// state DH. If specDh is nil, we are generating a new one.
	var (
		specDh       *mtree.DirectoryHierarchy
		stateDh      *mtree.DirectoryHierarchy
		specKeywords []mtree.Keyword
	)

	// -f <file>
	if len(c.StringSlice("file")) > 0 && !c.Bool("create") {
		// load the hierarchy, if we're not creating a new spec
		fh, err := os.Open(c.StringSlice("file")[0])
		if err != nil {
			return err
		}
		specDh, err = mtree.ParseSpec(fh)
		fh.Close()
		if err != nil {
			return err
		}

		// We can't check against more fields than in the specKeywords list, so
		// currentKeywords can only have a subset of specKeywords.
		specKeywords = specDh.UsedKeywords()
	}

	// -list-used
	if c.Bool("list-used") {
		if specDh == nil {
			return fmt.Errorf("no specification provided. please provide a validation manifest")
		}

		if c.String("result-format") == "json" {
			for _, file := range c.StringSlice("file") {
				// if they're asking for json, give it to them
				data := map[string][]mtree.Keyword{file: specKeywords}
				buf, err := json.MarshalIndent(data, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(buf))
			}
		} else {
			for _, file := range c.StringSlice("file") {
				fmt.Printf("Keywords used in [%s]:\n", file)
				for _, kw := range specKeywords {
					fmt.Printf(" %s", kw)
					if _, ok := mtree.KeywordFuncs[kw]; !ok {
						fmt.Print(" (unsupported)")
					}
					fmt.Printf("\n")
				}
			}
		}
		return nil
	}

	if specKeywords != nil {
		// If we didn't actually change the set of keywords, we can just use specKeywords.
		if c.String("use-keywords") == "" && c.String("add-keywords") == "" {
			currentKeywords = specKeywords
		}
	}

	// -p and -T are mutually exclusive
	if c.String("path") != "" && c.String("tar") != "" {
		return fmt.Errorf("options -T and -p are mutually exclusive")
	}

	// -p <path>
	var rootPath = "."
	if c.String("path") != "" {
		rootPath = c.String("path")
	}

	excludes := []mtree.ExcludeFunc{}
	// -d
	if c.Bool("directory-only") {
		excludes = append(excludes, mtree.ExcludeNonDirectories)
	}

	// -u
	// Failing early here. Processing is done below.
	if c.Bool("update-attributes") && c.String("tar") != "" {
		return fmt.Errorf("ERROR: -u can not be used with -T")
	}

	// -T <tar file>
	if c.String("tar") != "" {
		var input io.Reader
		if c.String("tar") == "-" {
			input = os.Stdin
		} else {
			fh, err := os.Open(c.String("tar"))
			if err != nil {
				return err
			}
			defer fh.Close()
			input = fh
		}
		ts := mtree.NewTarStreamer(input, excludes, currentKeywords)

		if _, err := io.Copy(io.Discard, ts); err != nil && err != io.EOF {
			return err
		}
		if err := ts.Close(); err != nil {
			return err
		}
		var err error
		stateDh, err = ts.Hierarchy()
		if err != nil {
			return err
		}
	} else if len(c.StringSlice("file")) > 1 {
		// load this second hierarchy file provided
		fh, err := os.Open(c.StringSlice("file")[1])
		if err != nil {
			return err
		}
		stateDh, err = mtree.ParseSpec(fh)
		fh.Close()
		if err != nil {
			return err
		}
	} else {
		// with a root directory
		stateDh, err = mtree.Walk(rootPath, excludes, currentKeywords, nil)
		if err != nil {
			return err
		}
	}

	// -u
	if c.Bool("update-attributes") && stateDh != nil {
		// -u
		// this comes before the next case, intentionally.
		result, err := mtree.Update(rootPath, specDh, mtree.DefaultUpdateKeywords, nil)
		if err != nil {
			return err
		}
		if len(result) > 0 {
			fmt.Printf("%#v\n", result)
		}

		var res []mtree.InodeDelta
		// only check the keywords that we just updated
		res, err = mtree.Check(rootPath, specDh, mtree.DefaultUpdateKeywords, nil)
		if err != nil {
			return err
		}
		if res != nil {
			out := formatFunc(res)
			if _, err := os.Stdout.Write([]byte(out)); err != nil {
				return err
			}

			// TODO: This should be a flag. Allowing files to be added and
			//       removed and still returning "it's all good" is simply
			//       unsafe IMO.
			for _, diff := range res {
				if diff.Type() == mtree.Modified {
					return fmt.Errorf("manifest validation failed")
				}
			}
		}

		return nil
	}

	// -c
	if c.Bool("create") {
		fh := os.Stdout
		if len(c.StringSlice("file")) > 0 {
			fh, err = os.Create(c.StringSlice("file")[0])
			if err != nil {
				return err
			}
		}

		// output stateDh
		_, err = stateDh.WriteTo(fh)
		return err
	}

	// no spec manifest has been provided yet, so look for it on stdin
	if specDh == nil {
		// load the hierarchy
		specDh, err = mtree.ParseSpec(os.Stdin)
		if err != nil {
			return err
		}

		// We can't check against more fields than in the specKeywords list, so
		// currentKeywords can only have a subset of specKeywords.
		// TODO this specKeywords is not even used
		_ = specDh.UsedKeywords()
	}

	// This is a validation.
	if specDh != nil && stateDh != nil {
		var res []mtree.InodeDelta
		res, err = mtree.Compare(specDh, stateDh, currentKeywords)
		if err != nil {
			return err
		}

		// Apply filters.
		var filters []deltaFilterFn
		if isTarSpec(specDh) || c.String("tar") != "" {
			filters = append(filters, tarKeywordFilter)
		}
		filters = append(filters, freebsdCompatKeywordFilter)
		res = filterDeltas(res, filters...)

		if len(res) > 0 {
			out := formatFunc(res)
			if _, err := os.Stdout.Write([]byte(out)); err != nil {
				return err
			}

			// TODO: This should be a flag. Allowing files to be added and
			//       removed and still returning "it's all good" is simply
			//       unsafe IMO.
			for _, diff := range res {
				if diff.Type() == mtree.Modified {
					return fmt.Errorf("manifest validation failed")
				}
			}
		}
	} else {
		return fmt.Errorf("neither validating or creating a manifest. Please provide additional arguments")
	}
	return nil
}

var formats = map[string]func([]mtree.InodeDelta) string{
	// Outputs the errors in the BSD format.
	"bsd": func(d []mtree.InodeDelta) string {
		var buffer bytes.Buffer
		for _, delta := range d {
			fmt.Fprintln(&buffer, delta)
		}
		return buffer.String()
	},

	// Outputs the full result struct in JSON.
	"json": func(d []mtree.InodeDelta) string {
		var buffer bytes.Buffer
		if err := json.NewEncoder(&buffer).Encode(d); err != nil {
			panic(err)
		}
		return buffer.String()
	},

	// Outputs only the paths which failed to validate.
	"path": func(d []mtree.InodeDelta) string {
		var buffer bytes.Buffer
		for _, delta := range d {
			if delta.Type() == mtree.Modified {
				fmt.Fprintln(&buffer, delta.Path())
			}
		}
		return buffer.String()
	},
}

// isDirEntry returns wheter an mtree.Entry describes a directory.
func isDirEntry(e mtree.Entry) bool {
	for _, kw := range e.Keywords {
		kv := mtree.KeyVal(kw)
		if kv.Keyword() == "type" {
			return kv.Value() == "dir"
		}
	}
	// Shouldn't be reached.
	return false
}

// tarKeywordFilter is a filter for diffs produced where one half is a tar
// archive. tar archive manifests do not have a "size" key associated with
// directories (due to limitations in manifest generation for tar archives) and
// so any deltas due to size missing should be removed.
func tarKeywordFilter(delta *mtree.InodeDelta) bool {
	if delta.Path() == "." {
		// Not all tar archives include a root entry so we should skip that
		// entry if we run into a diff that claims there is an issue with
		// it.
		return false
	}
	if delta.Type() != mtree.Modified {
		return true
	}
	// Strip out "size" entries for directory entries.
	if isDirEntry(*delta.Old()) || isDirEntry(*delta.New()) {
		keys := delta.DiffPtr()
		*keys = slices.DeleteFunc(*keys, func(kd mtree.KeyDelta) bool {
			return kd.Name() == "size"
		})
	}
	return true
}

// freebsdCompatKeywordFilter removes any deltas where a key is not present in
// both manifests being compared. This is necessary for compatibility with
// FreeBSD's mtree(8) but is generally undesireable for most users.
func freebsdCompatKeywordFilter(delta *mtree.InodeDelta) bool {
	if delta.Type() != mtree.Modified {
		return true
	}
	keys := delta.DiffPtr()
	*keys = slices.DeleteFunc(*keys, func(kd mtree.KeyDelta) bool {
		if kd.Name().Prefix() == "xattr" {
			// Even in FreeBSD compatibility mode, any xattr changes should
			// still be treated as a proper change and not filtered out.
			return false
		}
		return kd.Type() != mtree.Modified
	})
	return true
}

type deltaFilterFn func(*mtree.InodeDelta) bool

// filterDeltas takes the set of deltas generated by mtree and applies the
// given set of filters to it.
func filterDeltas(deltas []mtree.InodeDelta, filters ...deltaFilterFn) []mtree.InodeDelta {
	filtered := make([]mtree.InodeDelta, 0, len(deltas))
next:
	for _, delta := range deltas {
		for _, filter := range filters {
			if !filter(&delta) {
				continue next
			}
		}
		// Some filters might modify the entry to remove keyword deltas --
		// if there are no deltas left then we should skip the entry entirely.
		if delta.Type() == mtree.Modified && len(delta.Diff()) == 0 {
			continue next
		}
		filtered = append(filtered, delta)
	}
	return filtered
}

// isTarSpec returns whether the spec provided came from the tar generator.
// This takes advantage of an unsolveable problem in tar generation.
func isTarSpec(spec *mtree.DirectoryHierarchy) bool {
	// Find a directory and check whether it's missing size=...
	// NOTE: This will definitely break if someone drops the size=... keyword.
	for _, e := range spec.Entries {
		if !isDirEntry(e) {
			continue
		}

		for _, kw := range e.Keywords {
			kv := mtree.KeyVal(kw)
			if kv.Keyword() == "size" {
				return false
			}
		}
		return true
	}

	// Should never be reached.
	return false
}

func splitKeywordsArg(str string) []mtree.Keyword {
	keywords := []mtree.Keyword{}
	for _, kw := range strings.Fields(strings.Replace(str, ",", " ", -1)) {
		keywords = append(keywords, mtree.KeywordSynonym(kw))
	}
	return keywords
}
