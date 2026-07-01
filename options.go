package mtree

import "slices"

// WalkOptions configures the creation or validation of a DirectoryHierarchy.
// The zero value is not useful; use NewWalkOptions to obtain a properly
// initialised instance.
//
// Methods return the receiver so calls can be chained:
//
//	dh, err := mtree.NewWalkOptions().
//	    AddKeywords("sha256digest").
//	    RemoveKeywords("time").
//	    Walk("/path/to/dir")
type WalkOptions struct {
	keywords []Keyword
	excludes []ExcludeFunc
	fsEval   FsEval
}

// NewWalkOptions returns a WalkOptions initialised with DefaultKeywords and no
// excludes.
func NewWalkOptions() *WalkOptions {
	return &WalkOptions{
		keywords: append([]Keyword{}, DefaultKeywords...),
	}
}

// UseKeywords replaces the keyword set entirely with kws.
// "type" is always prepended if not already present, because the hierarchy
// traversal depends on it.
func (o *WalkOptions) UseKeywords(kws []Keyword) *WalkOptions {
	if !InKeywordSlice("type", kws) {
		o.keywords = append([]Keyword{"type"}, kws...)
	} else {
		o.keywords = append([]Keyword{}, kws...)
	}
	return o
}

// AddKeywords appends kws to the current keyword set, skipping duplicates.
func (o *WalkOptions) AddKeywords(kws ...Keyword) *WalkOptions {
	for _, kw := range kws {
		if !InKeywordSlice(kw, o.keywords) {
			o.keywords = append(o.keywords, kw)
		}
	}
	return o
}

// RemoveKeywords removes kws from the current keyword set.
func (o *WalkOptions) RemoveKeywords(kws ...Keyword) *WalkOptions {
	o.keywords = slices.DeleteFunc(o.keywords, func(kw Keyword) bool {
		return InKeywordSlice(kw, kws)
	})
	return o
}

// AddExclude appends an ExcludeFunc to the exclude list.
func (o *WalkOptions) AddExclude(fn ExcludeFunc) *WalkOptions {
	o.excludes = append(o.excludes, fn)
	return o
}

// SetFsEval sets a custom FsEval implementation. Passing nil restores the
// default (DefaultFsEval).
func (o *WalkOptions) SetFsEval(fsEval FsEval) *WalkOptions {
	o.fsEval = fsEval
	return o
}

// Keywords returns a copy of the current keyword set.
func (o *WalkOptions) Keywords() []Keyword {
	return o.keywords[:]
}

// Walk creates a DirectoryHierarchy rooted at root using these options.
// It is equivalent to calling mtree.Walk(root, excludes, keywords, fsEval).
func (o *WalkOptions) Walk(root string) (*DirectoryHierarchy, error) {
	return Walk(root, o.excludes, o.keywords, o.fsEval)
}

// Check validates root against dh using these options.
// It is equivalent to calling mtree.Check(root, dh, keywords, fsEval).
func (o *WalkOptions) Check(root string, dh *DirectoryHierarchy) ([]InodeDelta, error) {
	return Check(root, dh, o.keywords, o.fsEval)
}
