package mtree

import (
	"os"
	"sort"

	"github.com/Sirupsen/logrus"
)

// DefaultUpdateKeywords is the default set of keywords that can take updates to the files on disk
var DefaultUpdateKeywords = []Keyword{
	"uid",
	"gid",
	"mode",
	"time",
	"xattr",
}

// Update attempts to set the attributes of root directory path, given the values of `keywords` in dh DirectoryHierarchy.
func Update(root string, dh *DirectoryHierarchy, keywords []Keyword, fs FsEval) ([]InodeDelta, error) {
	creator := dhCreator{DH: dh}
	curDir, err := os.Getwd()
	if err == nil {
		defer os.Chdir(curDir)
	}

	if err := os.Chdir(root); err != nil {
		return nil, err
	}
	sort.Sort(byPos(creator.DH.Entries))

	results := []InodeDelta{}
	for i, e := range creator.DH.Entries {
		switch e.Type {
		case SpecialType:
			if e.Name == "/set" {
				creator.curSet = &creator.DH.Entries[i]
			} else if e.Name == "/unset" {
				creator.curSet = nil
			}
			logrus.Debugf("%#v", e)
			continue
		case RelativeType, FullType:
			e.Set = creator.curSet
			pathname, err := e.Path()
			if err != nil {
				return nil, err
			}

			// filter the keywords to update on the file, from the keywords available for this entry:
			var kvToUpdate []KeyVal
			kvToUpdate = keyvalSelector(e.AllKeys(), keywords)
			logrus.Debugf("kvToUpdate(%q): %#v", pathname, kvToUpdate)

			for _, kv := range kvToUpdate {
				if !InKeywordSlice(kv.Keyword().Prefix(), keywordPrefixes(keywords)) {
					continue
				}
				logrus.Debugf("finding function for %q (%q)", kv.Keyword(), kv.Keyword().Prefix())
				ukFunc, ok := UpdateKeywordFuncs[kv.Keyword().Prefix()]
				if !ok {
					logrus.Debugf("no UpdateKeywordFunc for %s; skipping", kv.Keyword())
					continue
				}
				if _, err := ukFunc(kv.Keyword(), pathname, kv.Value()); err != nil {
					results = append(results, InodeDelta{
						diff: ErrorDifference,
						path: pathname,
						old:  e,
						keys: []KeyDelta{
							{
								diff: ErrorDifference,
								name: kv.Keyword(),
								err:  err,
							},
						}})
				}
			}
		}
	}

	return results, nil
}

// Result of an "update" returns the produced results
type Result struct {
	Path    string
	Keyword Keyword
	Got     string
}
