package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vbatts/go-mtree"
)

// rel builds a RelativeType entry, optionally parented.
func rel(name string, parent *mtree.Entry) mtree.Entry {
	return mtree.Entry{Type: mtree.RelativeType, Name: name, Parent: parent}
}

// dir builds a RelativeType directory entry.
func dir(name string) *mtree.Entry {
	return &mtree.Entry{Type: mtree.RelativeType, Name: name, Keywords: []mtree.KeyVal{"type=dir"}}
}

func TestTidyVisitor(t *testing.T) {
	tests := []struct {
		name         string
		entry        mtree.Entry
		keepComments bool
		keepBlank    bool
		wantDrop     bool
	}{
		{
			name:     "drops comment by default",
			entry:    mtree.Entry{Type: mtree.CommentType, Raw: "# a comment"},
			wantDrop: true,
		},
		{
			name:         "keeps comment when requested",
			entry:        mtree.Entry{Type: mtree.CommentType, Raw: "# a comment"},
			keepComments: true,
			wantDrop:     false,
		},
		{
			name:     "drops blank by default",
			entry:    mtree.Entry{Type: mtree.BlankType},
			wantDrop: true,
		},
		{
			name:      "keeps blank when requested",
			entry:     mtree.Entry{Type: mtree.BlankType},
			keepBlank: true,
			wantDrop:  false,
		},
		{
			name:     "passes through relative entry",
			entry:    mtree.Entry{Type: mtree.RelativeType, Name: "file"},
			wantDrop: false,
		},
		{
			name:     "passes through dotdot entry",
			entry:    mtree.Entry{Type: mtree.DotDotType, Name: ".."},
			wantDrop: false,
		},
		{
			name:     "passes through special entry",
			entry:    mtree.Entry{Type: mtree.SpecialType, Name: "/set"},
			wantDrop: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := tidyVisitor{keepComments: tc.keepComments, keepBlank: tc.keepBlank}
			drop, err := v.Visit(&tc.entry)
			require.NoError(t, err)
			assert.Equal(t, tc.wantDrop, drop)
		})
	}
}

func TestStripPrefixVisitor(t *testing.T) {
	libDir := dir("lib")

	tests := []struct {
		name     string
		entry    mtree.Entry
		prefixes []string
		wantDrop bool
		wantName string // non-empty: assert entry.Name was rewritten to this value
	}{
		{
			name:     "drops relative entry whose path equals prefix",
			entry:    *libDir,
			prefixes: []string{"lib"},
			wantDrop: true,
		},
		{
			name:     "keeps relative child whose path does not equal prefix",
			entry:    rel("foo", libDir),
			prefixes: []string{"lib"},
			wantDrop: false,
		},
		{
			name:     "no-op when prefix does not match",
			entry:    *libDir,
			prefixes: []string{"other"},
			wantDrop: false,
		},
		{
			name:     "passes through non-path entry types",
			entry:    mtree.Entry{Type: mtree.CommentType, Raw: "# lib"},
			prefixes: []string{"lib"},
			wantDrop: false,
		},
		{
			name:     "strips prefix from full-type name",
			entry:    mtree.Entry{Type: mtree.FullType, Name: "lib/dir/file"},
			prefixes: []string{"lib"},
			wantDrop: false,
			wantName: "dir/file",
		},
		{
			name:     "drops full-type entry when name is empty after stripping",
			entry:    mtree.Entry{Type: mtree.FullType, Name: "lib"},
			prefixes: []string{"lib"},
			wantDrop: true,
		},
		{
			name:     "multiple prefixes: drops when any matches",
			entry:    *libDir,
			prefixes: []string{"other", "lib"},
			wantDrop: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := stripPrefixVisitor{prefixes: tc.prefixes}
			drop, err := v.Visit(&tc.entry)
			require.NoError(t, err)
			assert.Equal(t, tc.wantDrop, drop)
			if tc.wantName != "" {
				assert.Equal(t, tc.wantName, tc.entry.Name)
			}
		})
	}
}

func TestMutateRoundtrip(t *testing.T) {
	// Parse a small spec and verify the mutate loop preserves all non-tidy
	// entries when no prefixes are stripped and comments/blanks are kept.
	const spec = `# a comment
/set type=file mode=0644
. type=dir

    file1 size=10

..
`
	dh, err := mtree.ParseSpec(strings.NewReader(spec))
	require.NoError(t, err)

	// Count entry types in the original
	counts := map[mtree.EntryType]int{}
	for _, e := range dh.Entries {
		counts[e.Type]++
	}

	// Run through the mutate loop keeping everything
	sv := stripPrefixVisitor{}
	tv := tidyVisitor{keepComments: true, keepBlank: true}
	visitors := []Visitor{&sv, &tv}

	dropped := map[int]bool{}
	var out []mtree.Entry
outer:
	for _, entry := range dh.Entries {
		for _, v := range visitors {
			drop, err := v.Visit(&entry)
			require.NoError(t, err)
			if drop {
				dropped[entry.Pos] = true
				continue outer
			}
		}
		out = append(out, entry)
	}

	assert.Equal(t, len(dh.Entries), len(out), "all entries should be preserved when keeping comments and blanks")
	_ = dropped
}
