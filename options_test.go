package mtree

import (
	"testing"
)

func TestNewWalkOptions_defaults(t *testing.T) {
	o := NewWalkOptions()
	if len(o.Keywords()) == 0 {
		t.Fatal("expected default keywords to be non-empty")
	}
	if !InKeywordSlice("type", o.Keywords()) {
		t.Error("expected 'type' in default keywords")
	}
}

func TestWalkOptions_UseKeywords(t *testing.T) {
	o := NewWalkOptions().UseKeywords([]Keyword{"size", "mode"})
	kws := o.Keywords()
	// type is always prepended
	if kws[0] != "type" {
		t.Errorf("expected first keyword to be 'type', got %q", kws[0])
	}
	if !InKeywordSlice("size", kws) || !InKeywordSlice("mode", kws) {
		t.Error("expected 'size' and 'mode' in keywords")
	}
}

func TestWalkOptions_UseKeywords_typeAlreadyPresent(t *testing.T) {
	o := NewWalkOptions().UseKeywords([]Keyword{"type", "size"})
	count := 0
	for _, kw := range o.Keywords() {
		if kw == "type" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly one 'type' keyword, got %d", count)
	}
}

func TestWalkOptions_AddKeywords(t *testing.T) {
	o := NewWalkOptions().AddKeywords("sha256digest")
	if !InKeywordSlice("sha256digest", o.Keywords()) {
		t.Error("expected 'sha256digest' after AddKeywords")
	}
}

func TestWalkOptions_AddKeywords_noDuplicates(t *testing.T) {
	o := NewWalkOptions().AddKeywords("type", "type")
	count := 0
	for _, kw := range o.Keywords() {
		if kw == "type" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly one 'type' after AddKeywords with duplicates, got %d", count)
	}
}

func TestWalkOptions_RemoveKeywords(t *testing.T) {
	o := NewWalkOptions().RemoveKeywords("time", "size")
	if InKeywordSlice("time", o.Keywords()) {
		t.Error("expected 'time' to be removed")
	}
	if InKeywordSlice("size", o.Keywords()) {
		t.Error("expected 'size' to be removed")
	}
	// other keywords still present
	if !InKeywordSlice("mode", o.Keywords()) {
		t.Error("expected 'mode' to remain after removing unrelated keywords")
	}
}

func TestWalkOptions_chaining(t *testing.T) {
	o := NewWalkOptions().
		UseKeywords([]Keyword{"size", "mode"}).
		AddKeywords("sha256digest").
		RemoveKeywords("size")

	kws := o.Keywords()
	if InKeywordSlice("size", kws) {
		t.Error("'size' should have been removed")
	}
	if !InKeywordSlice("sha256digest", kws) {
		t.Error("'sha256digest' should be present")
	}
	if !InKeywordSlice("mode", kws) {
		t.Error("'mode' should be present")
	}
}

func TestWalkOptions_RemoveKeywords_all(t *testing.T) {
	o := NewWalkOptions().RemoveKeywords("all")
	if len(o.Keywords()) != 0 {
		t.Errorf("expected empty keyword set after RemoveKeywords(all), got %v", o.Keywords())
	}
}

func TestNewWalkOptionsFrom(t *testing.T) {
	o := NewWalkOptionsFrom(DefaultTarKeywords)
	if !InKeywordSlice("tar_time", o.Keywords()) {
		t.Error("expected tar_time in keywords from DefaultTarKeywords")
	}
	if InKeywordSlice("time", o.Keywords()) {
		t.Error("expected time NOT in keywords from DefaultTarKeywords")
	}
	// Must be independent from the global — mutations must not affect DefaultTarKeywords.
	before := len(DefaultTarKeywords)
	o.RemoveKeywords("tar_time")
	if len(DefaultTarKeywords) != before {
		t.Error("RemoveKeywords mutated DefaultTarKeywords")
	}
}

func TestWalkOptions_Excludes(t *testing.T) {
	o := NewWalkOptions().AddExclude(ExcludeNonDirectories)
	if len(o.Excludes()) != 1 {
		t.Errorf("expected 1 exclude, got %d", len(o.Excludes()))
	}
}

func TestWalkOptions_Walk(t *testing.T) {
	dh, err := NewWalkOptions().
		UseKeywords([]Keyword{"type", "mode"}).
		Walk("testdata")
	if err != nil {
		t.Fatal(err)
	}
	if len(dh.Entries) == 0 {
		t.Fatal("expected entries from walking testdata/")
	}
}

func TestWalkOptions_Check(t *testing.T) {
	o := NewWalkOptions().UseKeywords([]Keyword{"type", "mode"})

	dh, err := o.Walk("testdata")
	if err != nil {
		t.Fatal(err)
	}

	deltas, err := o.Check("testdata", dh)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range deltas {
		if d.Type() != Same {
			t.Errorf("unexpected delta: %v", d)
		}
	}
}
