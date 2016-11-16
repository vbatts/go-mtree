package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/vbatts/go-mtree"
)

var (
	flCreate           = flag.Bool("c", false, "create a directory hierarchy spec")
	flFile             = flag.String("f", "", "directory hierarchy spec to validate")
	flPath             = flag.String("p", "", "root path that the hierarchy spec is relative to")
	flAddKeywords      = flag.String("K", "", "Add the specified (delimited by comma or space) keywords to the current set of keywords")
	flUseKeywords      = flag.String("k", "", "Use the specified (delimited by comma or space) keywords as the current set of keywords")
	flListKeywords     = flag.Bool("list-keywords", false, "List the keywords available")
	flResultFormat     = flag.String("result-format", "bsd", "output the validation results using the given format (bsd, json, path)")
	flTar              = flag.String("T", "", "use tar archive to create or validate a directory hierarchy spec (\"-\" indicates stdin)")
	flBsdKeywords      = flag.Bool("bsd-keywords", false, "only operate on keywords that are supported by upstream mtree(8)")
	flListUsedKeywords = flag.Bool("list-used", false, "list all the keywords found in a validation manifest")
	flDebug            = flag.Bool("debug", false, "output debug info to STDERR")
	flVersion          = flag.Bool("version", false, "display the version of this tool")
)

var formats = map[string]func([]mtree.InodeDelta) string{
	// Outputs the errors in the BSD format.
	"bsd": func(d []mtree.InodeDelta) string {
		var buffer bytes.Buffer
		for _, delta := range d {
			if delta.Type() == mtree.Modified {
				fmt.Fprintln(&buffer, delta)
			} else if delta.Type() == mtree.Missing {
				fmt.Fprintln(&buffer, delta)
			} else if delta.Type() == mtree.Extra {
				fmt.Fprintln(&buffer, delta)
			}
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

// filterMissingKeywords is a fairly annoying hack to get around the fact that
// tar archive manifest generation has certain unsolveable problems regarding
// certain keywords. For example, the size=... keyword cannot be implemented
// for directories in a tar archive (which causes Missing errors for that
// keyword).
//
// This function just removes all instances of Missing errors for keywords.
// This makes certain assumptions about the type of issues tar archives have.
// Only call this on tar archive manifest comparisons.
func filterMissingKeywords(diffs []mtree.InodeDelta) []mtree.InodeDelta {
	newDiffs := []mtree.InodeDelta{}
loop:
	for _, diff := range diffs {
		if diff.Type() == mtree.Modified {
			// We only apply this filtering to directories.
			// NOTE: This will probably break if someone drops the size keyword.
			if isDirEntry(*diff.Old()) || isDirEntry(*diff.New()) {
				// If this applies to '.' then we just filter everything
				// (meaning we remove this entry). This is because note all tar
				// archives include a '.' entry. Which makes checking this not
				// practical.
				if diff.Path() == "." {
					continue
				}

				// Only filter out the size keyword.
				// NOTE: This currently takes advantage of the fact the
				//       diff.Diff() returns the actual slice to diff.keys.
				keys := diff.Diff()
				for idx, k := range keys {
					// Delete the key if it's "size". Unfortunately in Go you
					// can't delete from a slice without reassigning it. So we
					// just overwrite it with the last value.
					if k.Name() == "size" {
						if len(keys) < 2 {
							continue loop
						}
						keys[idx] = keys[len(keys)-1]
					}
				}
			}
		}

		// If we got here, append to the new set.
		newDiffs = append(newDiffs, diff)
	}
	return newDiffs
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

func main() {
	flag.Parse()

	if *flDebug {
		os.Setenv("DEBUG", "1")
	}

	// so that defers cleanly exec
	// TODO: Switch everything to being inside a function, to remove the need for isErr.
	var isErr bool
	defer func() {
		if isErr {
			os.Exit(1)
		}
	}()

	if *flVersion {
		fmt.Printf("%s :: %s\n", os.Args[0], mtree.Version)
		return
	}

	// -list-keywords
	if *flListKeywords {
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
		return
	}

	// --result-format
	formatFunc, ok := formats[*flResultFormat]
	if !ok {
		log.Printf("invalid output format: %s", *flResultFormat)
		isErr = true
		return
	}

	var (
		err             error
		tmpKeywords     []string
		currentKeywords []string
	)

	// -k <keywords>
	if *flUseKeywords != "" {
		tmpKeywords = splitKeywordsArg(*flUseKeywords)
		if !inSlice("type", tmpKeywords) {
			tmpKeywords = append([]string{"type"}, tmpKeywords...)
		}
	} else {
		if *flTar != "" {
			tmpKeywords = mtree.DefaultTarKeywords[:]
		} else {
			tmpKeywords = mtree.DefaultKeywords[:]
		}
	}

	// -K <keywords>
	if *flAddKeywords != "" {
		for _, kw := range splitKeywordsArg(*flAddKeywords) {
			if !inSlice(kw, tmpKeywords) {
				tmpKeywords = append(tmpKeywords, kw)
			}
		}
	}

	// -bsd-keywords
	if *flBsdKeywords {
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
	if inSlice("tar_time", currentKeywords) && inSlice("time", currentKeywords) {
		log.Printf("tar_time and time are mutually exclusive keywords")
		isErr = true
		return
	}

	// If we're doing a comparison, we always are comparing between a spec and
	// state DH. If specDh is nil, we are generating a new one.
	var (
		specDh       *mtree.DirectoryHierarchy
		stateDh      *mtree.DirectoryHierarchy
		specKeywords []string
	)

	// -f <file>
	if *flFile != "" && !*flCreate {
		// load the hierarchy, if we're not creating a new spec
		fh, err := os.Open(*flFile)
		if err != nil {
			log.Println(err)
			isErr = true
			return
		}
		specDh, err = mtree.ParseSpec(fh)
		fh.Close()
		if err != nil {
			log.Println(err)
			isErr = true
			return
		}

		// We can't check against more fields than in the specKeywords list, so
		// currentKeywords can only have a subset of specKeywords.
		specKeywords = mtree.CollectUsedKeywords(specDh)
	}

	// -list-used
	if *flListUsedKeywords {
		if specDh == nil {
			log.Println("no specification provided. please provide a validation manifest")
			isErr = true
			return
		}

		if *flResultFormat == "json" {
			// if they're asking for json, give it to them
			data := map[string][]string{*flFile: specKeywords}
			buf, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				defer os.Exit(1)
				isErr = true
				return
			}
			fmt.Println(string(buf))
		} else {
			fmt.Printf("Keywords used in [%s]:\n", *flFile)
			for _, kw := range specKeywords {
				fmt.Printf(" %s", kw)
				if _, ok := mtree.KeywordFuncs[kw]; !ok {
					fmt.Print(" (unsupported)")
				}
				fmt.Printf("\n")
			}
		}
		return
	}

	if specKeywords != nil {
		// If we didn't actually change the set of keywords, we can just use specKeywords.
		if *flUseKeywords == "" && *flAddKeywords == "" {
			currentKeywords = specKeywords
		}

		for _, keyword := range currentKeywords {
			// As always, time is a special case.
			// TODO: Fix that.
			if (keyword == "time" && inSlice("tar_time", specKeywords)) || (keyword == "tar_time" && inSlice("time", specKeywords)) {
				continue
			}

			if !inSlice(keyword, specKeywords) {
				log.Printf("cannot verify keywords not in mtree specification: %s\n", keyword)
				isErr = true
			}
			if isErr {
				return
			}
		}
	}

	// -p and -T are mutually exclusive
	if *flPath != "" && *flTar != "" {
		log.Println("options -T and -p are mutually exclusive")
		isErr = true
		return
	}

	// -p <path>
	var rootPath = "."
	if *flPath != "" {
		rootPath = *flPath
	}

	// -T <tar file>
	if *flTar != "" {
		var input io.Reader
		if *flTar == "-" {
			input = os.Stdin
		} else {
			fh, err := os.Open(*flTar)
			if err != nil {
				log.Println(err)
				isErr = true
				return
			}
			defer fh.Close()
			input = fh
		}
		ts := mtree.NewTarStreamer(input, currentKeywords)

		if _, err := io.Copy(ioutil.Discard, ts); err != nil && err != io.EOF {
			log.Println(err)
			isErr = true
			return
		}
		if err := ts.Close(); err != nil {
			log.Println(err)
			isErr = true
			return
		}
		var err error
		stateDh, err = ts.Hierarchy()
		if err != nil {
			log.Println(err)
			isErr = true
			return
		}
	} else {
		// with a root directory
		stateDh, err = mtree.Walk(rootPath, nil, currentKeywords)
		if err != nil {
			log.Println(err)
			isErr = true
			return
		}
	}

	// -c
	if *flCreate {
		fh := os.Stdout
		if *flFile != "" {
			fh, err = os.Create(*flFile)
			if err != nil {
				log.Println(err)
				isErr = true
				return
			}
		}

		// output stateDh
		stateDh.WriteTo(fh)
		return
	}

	// This is a validation.
	if specDh != nil && stateDh != nil {
		var res []mtree.InodeDelta

		res, err = mtree.Compare(specDh, stateDh, currentKeywords)
		if err != nil {
			log.Println(err)
			isErr = true
			return
		}
		if res != nil {
			if isTarSpec(specDh) || *flTar != "" {
				res = filterMissingKeywords(res)
			}
			if len(res) > 0 {
				defer os.Exit(1)
			}
			out := formatFunc(res)
			if _, err := os.Stdout.Write([]byte(out)); err != nil {
				log.Println(err)
				isErr = true
				return
			}
		}
	} else {
		log.Println("neither validating or creating a manifest. Please provide additional arguments")
		isErr = true
		return
	}
}

func splitKeywordsArg(str string) []string {
	return strings.Fields(strings.Replace(str, ",", " ", -1))
}

func inSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
