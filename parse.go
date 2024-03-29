package mtree

import (
	"bufio"
	"io"
	"path/filepath"
	"strings"
)

// ParseSpec reads a stream of an mtree specification, and returns the DirectoryHierarchy
func ParseSpec(r io.Reader) (*DirectoryHierarchy, error) {
	s := bufio.NewScanner(r)
	i := int(0)
	creator := dhCreator{
		DH: &DirectoryHierarchy{},
	}
	for s.Scan() {
		str := s.Text()
		trimmedStr := strings.TrimLeftFunc(str, func(c rune) bool {
			return c == ' ' || c == '\t'
		})
		e := Entry{Pos: i}
		switch {
		case strings.HasPrefix(trimmedStr, "#"):
			e.Raw = str
			if strings.HasPrefix(trimmedStr, "#mtree") {
				e.Type = SignatureType
			} else {
				e.Type = CommentType
				// from here, the comment could be "# key: value" metadata
				// or a relative path hint
			}
		case str == "":
			e.Type = BlankType
			// nothing else to do here
		case strings.HasPrefix(str, "/"):
			e.Type = SpecialType
			// collapse any escaped newlines
			for {
				if strings.HasSuffix(str, `\`) {
					str = str[:len(str)-1]
					s.Scan()
					str += s.Text()
				} else {
					break
				}
			}
			// parse the options
			f := strings.Fields(str)
			e.Name = f[0]
			e.Keywords = StringToKeyVals(f[1:])
			if e.Name == "/set" {
				creator.curSet = &e
			} else if e.Name == "/unset" {
				creator.curSet = nil
			}
		case len(strings.Fields(str)) > 0 && strings.Fields(str)[0] == "..":
			e.Type = DotDotType
			e.Raw = str
			if creator.curDir != nil {
				creator.curDir = creator.curDir.Parent
			}
			// nothing else to do here
		case len(strings.Fields(str)) > 0:
			// collapse any escaped newlines
			for {
				if strings.HasSuffix(str, `\`) {
					str = str[:len(str)-1]
					s.Scan()
					str += s.Text()
				} else {
					break
				}
			}

			// parse the options
			f := strings.Fields(str)
			e.Name = f[0]
			e.Keywords = StringToKeyVals(f[1:])
			// TODO: gather keywords if using tar stream
			var isDir bool
			for _, kv := range e.Keywords {
				if kv.Keyword() == "type" {
					isDir = kv.Value() == "dir"
				}
			}
			if strings.Contains(e.Name, "/") {
				e.Type = FullType
			} else {
				e.Type = RelativeType
				e.Parent = creator.curDir
				if isDir {
					creator.curDir = &e
				}
			}
			if !isDir {
				creator.curEnt = &e
			}
			e.Set = creator.curSet
			// we need to clean the filepath at the end because '/'s can be
			// stripped, which would cause FullTypes to be treated as
			// RelativeTypes above
			e.Name = filepath.Clean(e.Name)
		default:
			// TODO(vbatts) log a warning?
			continue
		}
		creator.DH.Entries = append(creator.DH.Entries, e)
		i++
	}
	return creator.DH, s.Err()
}
