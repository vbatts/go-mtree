/*
 * govis: unicode aware vis(3) encoding implementation
 * Copyright (C) 2017 SUSE LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package govis

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

func isunsafe(ch rune) bool {
	return ch == '\b' || ch == '\007' || ch == '\r'
}

func isglob(ch rune) bool {
	return ch == '*' || ch == '?' || ch == '[' || ch == '#'
}

func ishttp(ch rune) bool {
	return unicode.IsDigit(ch) || unicode.IsLetter(ch) ||
		// Safe characters.
		ch == '$' || ch == '-' || ch == '_' || ch == '.' || ch == '+' ||
		// Extra characters.
		ch == '!' || ch == '*' || ch == '\'' || ch == '(' ||
		ch == ')' || ch == ','
}

func mapRuneBytes(ch rune, fn func(byte) string) string {
	bytes := make([]byte, utf8.RuneLen(ch))
	n := utf8.EncodeRune(bytes, ch)

	mapped := ""
	for i := 0; i < n; i++ {
		mapped += fn(bytes[i])
	}
	return mapped
}

// vis converts a single rune into its encoding, ensuring that it is "safe"
// (for some definition of safe). Note that some visual characters (such as
// accented characters or similar things) can be made up of several runes -- in
// order to maintain my sanity Vis() makes no attempt to handle such cases
// specially.
func vis(ch rune, flag VisFlag) (string, error) {
	// XXX: Currently we are just allowing regular multi-byte characters such
	//      as accents and so on to be passed through without encoding. Is this
	//      really the best idea? In order to maintain compatibility with
	//      vis(3) such that an older unvis(3) will do the right thing maybe we
	//      should only output 7-bit ASCII? I'm not sure.

	if flag&VisHTTPStyle == VisHTTPStyle {
		// This is described in RFC 1808.
		if !ishttp(ch) {
			return mapRuneBytes(ch, func(b byte) string {
				return fmt.Sprintf("%.2X", b)
			}), nil
		}
	}

	// Handle all "ordinary" characters which don't need to be encoded.
	if !(flag&VisGlob == VisGlob && isglob(ch)) &&
		((unicode.IsGraphic(ch) && !unicode.IsSpace(ch)) ||
			(flag&VisSpace == 0 && ch == ' ') ||
			(flag&VisTab == 0 && ch == '\t') ||
			(flag&VisNewline == 0 && ch == '\n') ||
			(flag&VisSafe == VisSafe && isunsafe(ch))) {
		enc := string(ch)
		if ch == '\\' && flag&VisNoSlash == 0 {
			enc += "\\"
		}
		return enc, nil
	}

	if flag&VisCStyle == VisCStyle {
		switch ch {
		case '\n':
			return "\\n", nil
		case '\r':
			return "\\r", nil
		case '\b':
			return "\\b", nil
		case '\a':
			return "\\a", nil
		case '\v':
			return "\\v", nil
		case '\t':
			return "\\t", nil
		case '\f':
			return "\\f", nil
		case 0:
			// TODO: Handle isoctal properly.
			return "\\000", nil
		}
	}

	// TODO: ch & 0177 is not implemented...
	if flag&VisOctal == VisOctal || unicode.IsGraphic(ch) {
		return mapRuneBytes(ch, func(b byte) string {
			return fmt.Sprintf("\\%.3o", b)
		}), nil
	}

	return mapRuneBytes(ch, func(b byte) string {
		enc := ""
		if flag&VisNoSlash == 0 {
			enc += "\\"
		}

		// This logic is stolen from cvis, I don't understand any of it.
		if b&0200 != 0 {
			b &= 0177
			enc += "M"
		}
		if unicode.IsControl(rune(b)) {
			enc += "^"
			if b == 0177 {
				enc += "?"
			} else {
				enc += string(b + '@')
			}
		} else {
			enc += fmt.Sprintf("-%s", b)
		}

		return enc
	}), nil
}

// Vis encodes the provided string to a BSD-compatible encoding using BSD's
// vis() flags. However, it will correctly handle multi-byte encoding (which is
// not done properly by BSD's vis implementation).
func Vis(src string, flag VisFlag) (string, error) {
	if !utf8.ValidString(src) {
		return "", fmt.Errorf("vis: input string is invalid utf8 literal")
	}

	output := ""
	for _, ch := range src {
		encodedCh, err := vis(ch, flag)
		if err != nil {
			return "", err
		}
		output += encodedCh
	}

	return output, nil
}
