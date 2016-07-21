package mtree

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

// DirectoryHierarchy is the mapped structure for an mtree directory hierarchy
// spec
type DirectoryHierarchy struct {
	Entries []Entry
}

// WriteTo simplifies the output of the resulting hierarchy spec
func (dh DirectoryHierarchy) WriteTo(w io.Writer) (n int64, err error) {
	sort.Sort(byPos(dh.Entries))
	var sum int64
	for _, e := range dh.Entries {
		str, err := e.String()
		if err != nil {
			return sum, err
		}
		i, err := io.WriteString(w, str+"\n")
		if err != nil {
			return sum, err
		}
		sum += int64(i)
	}
	return sum, nil
}

type byPos []Entry

func (bp byPos) Len() int           { return len(bp) }
func (bp byPos) Less(i, j int) bool { return bp[i].Pos < bp[j].Pos }
func (bp byPos) Swap(i, j int)      { bp[i], bp[j] = bp[j], bp[i] }

// Entry is each component of content in the mtree spec file
type Entry struct {
	Parent, Child *Entry   // up, down
	Prev, Next    *Entry   // left, right
	Set           *Entry   // current `/set` for additional keywords
	Pos           int      // order in the spec
	Raw           string   // file or directory name
	Name          string   // file or directory name
	Keywords      []string // TODO(vbatts) maybe a keyword typed set of values?
	Type          EntryType
}

// Path provides the full path of the file, despite RelativeType or FullType
func (e Entry) Path() (string, error) {
	decodedName, err := Unvis(e.Name)
	if err != nil {
		return "", err
	}
	if e.Parent == nil || e.Type == FullType {
		return filepath.Clean(decodedName), nil
	}
	parentName, err := e.Parent.Path()
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(parentName, decodedName)), nil
}

func (e Entry) String() (string, error) {
	if e.Raw != "" {
		return e.Raw, nil
	}
	if e.Type == BlankType {
		return "", nil
	}
	if e.Type == DotDotType {
		return e.Name, nil
	}
	decodedName, err := Unvis(e.Name)
	if err != nil {
		return "", err
	}
	if e.Type == SpecialType || e.Type == FullType || inSlice("type=dir", e.Keywords) {
		return fmt.Sprintf("%s %s", decodedName, strings.Join(e.Keywords, " ")), nil
	}
	return fmt.Sprintf("    %s %s", decodedName, strings.Join(e.Keywords, " ")), nil
}

// EntryType are the formats of lines in an mtree spec file
type EntryType int

// The types of lines to be found in an mtree spec file
const (
	SignatureType EntryType = iota // first line of the file, like `#mtree v2.0`
	BlankType                      // blank lines are ignored
	CommentType                    // Lines beginning with `#` are ignored
	SpecialType                    // line that has `/` prefix issue a "special" command (currently only /set and /unset)
	RelativeType                   // if the first white-space delimited word does not have a '/' in it. Options/keywords are applied.
	DotDotType                     // .. - A relative path step. keywords/options are ignored
	FullType                       // if the first word on the line has a `/` after the first character, it interpretted as a file pathname with options
)

// String returns the name of the EntryType
func (et EntryType) String() string {
	return typeNames[et]
}

var typeNames = map[EntryType]string{
	SignatureType: "SignatureType",
	BlankType:     "BlankType",
	CommentType:   "CommentType",
	SpecialType:   "SpecialType",
	RelativeType:  "RelativeType",
	DotDotType:    "DotDotType",
	FullType:      "FullType",
}
