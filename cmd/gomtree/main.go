package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/vbatts/go-mtree"
)

var (
	flCreate       = flag.Bool("c", false, "create a directory hierarchy spec")
	flFile         = flag.String("f", "", "directory hierarchy spec to validate")
	flPath         = flag.String("p", "", "root path that the hierarchy spec is relative to")
	flAddKeywords  = flag.String("K", "", "Add the specified (delimited by comma or space) keywords to the current set of keywords")
	flUseKeywords  = flag.String("k", "", "Use the specified (delimited by comma or space) keywords as the current set of keywords")
	flListKeywords = flag.Bool("list-keywords", false, "List the keywords available")
	flResultFormat = flag.String("result-format", "bsd", "output the validation results using the given format (bsd, json, text)")
)

var formats = map[string]func(*mtree.Result) string{
	// Outputs the errors in the BSD format.
	"bsd": func(r *mtree.Result) string {
		var buffer bytes.Buffer
		for _, fail := range r.Failures {
			fmt.Fprintln(&buffer, fail)
		}
		return buffer.String()
	},

	// Outputs the full result struct in JSON.
	"json": func(r *mtree.Result) string {
		var buffer bytes.Buffer
		if err := json.NewEncoder(&buffer).Encode(r); err != nil {
			panic(err)
		}
		return buffer.String()
	},

	// Outputs only the failure type and paths which failed to validate.
	"text": func(r *mtree.Result) string {
		var buffer bytes.Buffer
		for _, fail := range r.Failures {
			fmt.Fprintf(&buffer, "%s\t%s\n", fail.Type(), fail.Path())
		}
		return buffer.String()
	},
}

func main() {
	flag.Parse()

	// so that defers cleanly exec
	var isErr bool
	defer func() {
		if isErr {
			os.Exit(1)
		}
	}()

	// -l
	if *flListKeywords {
		fmt.Println("Available keywords:")
		for k := range mtree.KeywordFuncs {
			if inSlice(k, mtree.DefaultKeywords) {
				fmt.Println(" ", k, " (default)")
			} else {
				fmt.Println(" ", k)
			}
		}
		return
	}

	// --output
	formatFunc, ok := formats[*flResultFormat]
	if !ok {
		log.Printf("invalid output format: %s", *flResultFormat)
		isErr = true
		return
	}

	var currentKeywords []string
	// -k <keywords>
	if *flUseKeywords != "" {
		currentKeywords = splitKeywordsArg(*flUseKeywords)
		if !inSlice("type", currentKeywords) {
			currentKeywords = append([]string{"type"}, currentKeywords...)
		}
	} else {
		currentKeywords = mtree.DefaultKeywords[:]
	}
	// -K <keywords>
	if *flAddKeywords != "" {
		currentKeywords = append(currentKeywords, splitKeywordsArg(*flAddKeywords)...)
	}

	// -f <file>
	var dh *mtree.DirectoryHierarchy
	if *flFile != "" && !*flCreate {
		// load the hierarchy, if we're not creating a new spec
		fh, err := os.Open(*flFile)
		if err != nil {
			log.Println(err)
			isErr = true
			return
		}
		dh, err = mtree.ParseSpec(fh)
		fh.Close()
		if err != nil {
			log.Println(err)
			isErr = true
			return
		}
	}

	// -p <path>
	var rootPath = "."
	if *flPath != "" {
		rootPath = *flPath
	}

	// -c
	if *flCreate {
		// create a directory hierarchy
		dh, err := mtree.Walk(rootPath, nil, currentKeywords)
		if err != nil {
			log.Println(err)
			isErr = true
			return
		}
		dh.WriteTo(os.Stdout)
	} else if dh != nil {
		// else this is a validation
		res, err := mtree.Check(rootPath, dh, currentKeywords)
		if err != nil {
			log.Println(err)
			isErr = true
			return
		}
		if res != nil && len(res.Failures) > 0 {
			defer os.Exit(1)
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
		defer os.Exit(1)
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
