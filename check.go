package mtree

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Result of a Check
type Result struct {
	// list of any failures in the Check
	Failures []Failure `json:"failures"`
}

// FailureType represents a type of Failure encountered when checking for
// discrepancies between a manifest and a directory state.
type FailureType string

const (
	// None means no discrepancy (unused).
	None FailureType = "none"

	// Missing represents a discrepancy where an object is present in the
	// manifest but is not present in the directory state being checked.
	Missing FailureType = "missing"

	// Modified represents a discrepancy where one or more of the keywords
	// present in the manifest do not match the keywords generated for the same
	// object in the directory state being checked.
	Modified FailureType = "modified"
)

// Failure represents a discrepancy between a manifest and the state it is
// being compared against. The discrepancy may take the form of a missing,
// erroneous or otherwise modified object.
type Failure interface {
	// String returns a "pretty" formatting for a Failure. It's based on the
	// BSD mtree(8) format, so that we are compatible.
	String() string

	// MarshalJSON ensures that we fulfil the JSON Marshaler interface.
	MarshalJSON() ([]byte, error)

	// Path returns the path (relative to the root of the tree) that this
	// discrepancy refers to.
	Path() string

	// Type returns the type of failure that occurred.
	Type() FailureType
}

// An object modified from the manifest state.
type modified struct {
	path     string
	keyword  string
	expected string
	got      string
}

func (m modified) String() string {
	return fmt.Sprintf("%q: keyword %q: expected %s; got %s", m.path, m.keyword, m.expected, m.got)
}

func (m modified) MarshalJSON() ([]byte, error) {
	// Because of Go's reflection policies, we have to make an anonymous struct
	// with the fields exported.
	return json.Marshal(struct {
		Type     FailureType `json:"type"`
		Path     string      `json:"path"`
		Keyword  string      `json:"keyword"`
		Expected string      `json:"expected"`
		Got      string      `json:"got"`
	}{
		Type:     m.Type(),
		Path:     m.path,
		Keyword:  m.keyword,
		Expected: m.expected,
		Got:      m.got,
	})
}

func (m modified) Path() string {
	return m.path
}

func (m modified) Type() FailureType {
	return Modified
}

// An object that is listed in the manifest but is not present in the state.
type missing struct {
	path string
}

func (m missing) String() string {
	return fmt.Sprintf("%q: expected object missing", m.path)
}

func (m missing) MarshalJSON() ([]byte, error) {
	// Because of Go's reflection policies, we have to make an anonymous struct
	// with the fields exported.
	return json.Marshal(struct {
		Type FailureType `json:"type"`
		Path string      `json:"path"`
	}{
		Type: m.Type(),
		Path: m.path,
	})
}

func (m missing) Path() string {
	return m.path
}

func (m missing) Type() FailureType {
	return Missing
}

// Check a root directory path against the DirectoryHierarchy, regarding only
// the available keywords from the list and each entry in the hierarchy.
// If keywords is nil, the check all present in the DirectoryHierarchy
func Check(root string, dh *DirectoryHierarchy, keywords []string) (*Result, error) {
	creator := dhCreator{DH: dh}
	curDir, err := os.Getwd()
	if err == nil {
		defer os.Chdir(curDir)
	}

	if err := os.Chdir(root); err != nil {
		return nil, err
	}
	sort.Sort(byPos(creator.DH.Entries))

	var result Result
	for i, e := range creator.DH.Entries {
		switch e.Type {
		case SpecialType:
			if e.Name == "/set" {
				creator.curSet = &creator.DH.Entries[i]
			} else if e.Name == "/unset" {
				creator.curSet = nil
			}
		case RelativeType, FullType:
			info, err := os.Lstat(e.Path())
			if err != nil {
				if os.IsNotExist(err) {
					result.Failures = append(result.Failures, missing{
						path: e.Path(),
					})
					continue
				}
				return nil, err
			}

			var kvs KeyVals
			if creator.curSet != nil {
				kvs = MergeSet(creator.curSet.Keywords, e.Keywords)
			} else {
				kvs = NewKeyVals(e.Keywords)
			}

			for _, kv := range kvs {
				keywordFunc, ok := KeywordFuncs[kv.Keyword()]
				if !ok {
					return nil, fmt.Errorf("Unknown keyword %q for file %q", kv.Keyword(), e.Path())
				}
				if keywords != nil && !inSlice(kv.Keyword(), keywords) {
					continue
				}
				curKeyVal, err := keywordFunc(filepath.Join(root, e.Path()), info)
				if err != nil {
					return nil, err
				}
				if string(kv) != curKeyVal {
					result.Failures = append(result.Failures, modified{
						path:     e.Path(),
						keyword:  kv.Keyword(),
						expected: kv.Value(),
						got:      KeyVal(curKeyVal).Value(),
					})
				}
			}
		}
	}
	return &result, nil
}
