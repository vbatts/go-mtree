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
	"strconv"
	"unicode/utf8"
)

// unvisParser stores the current state of the token parser.
type unvisParser struct {
	tokens []rune
	idx    int
	flag   VisFlag
}

// Next moves the index to the next character.
func (p *unvisParser) Next() {
	p.idx++
}

// Peek gets the current token.
func (p *unvisParser) Peek() (rune, error) {
	if p.idx >= len(p.tokens) {
		return utf8.RuneError, fmt.Errorf("tried to read past end of token list")
	}
	return p.tokens[p.idx], nil
}

// End returns whether all of the tokens have been consumed.
func (p *unvisParser) End() bool {
	return p.idx >= len(p.tokens)
}

func newParser(input string, flag VisFlag) *unvisParser {
	return &unvisParser{
		tokens: []rune(input),
		idx:    0,
		flag:   flag,
	}
}

// While a recursive descent parser is overkill for parsing simple escape
// codes, this is IMO much easier to read than the ugly 80s coroutine code used
// by the original unvis(3) parser. Here's the EBNF for an unvis sequence:
//
// <input>           ::= (<rune>)*
// <rune>            ::= ("\" <escape-sequence>) | ("%" <escape-hex>) | <plain-rune>
// <plain-rune>      ::= any rune
// <escape-sequence> ::= ("x" <escape-hex>) | ("M") | <escape-cstyle> | <escape-octal>
// <escape-hex>      ::= [0-9a-f] [0-9a-f]
// <escape-cstyle>   ::= "\" | "n" | "r" | "b" | "a" | "v" | "t" | "f"
// <escape-octal>    ::= [0-7] ([0-7] ([0-7])?)?

func unvisPlainRune(p *unvisParser) (string, error) {
	ch, err := p.Peek()
	if err != nil {
		return "", fmt.Errorf("plain rune: %s", ch)
	}
	p.Next()
	return string(ch), nil
}

func unvisEscapeCStyle(p *unvisParser) (string, error) {
	ch, err := p.Peek()
	if err != nil {
		return "", fmt.Errorf("escape hex: %s", err)
	}

	output := ""
	switch ch {
	case 'n':
		output = "\n"
	case 'r':
		output = "\r"
	case 'b':
		output = "\b"
	case 'a':
		output = "\x07"
	case 'v':
		output = "\v"
	case 't':
		output = "\t"
	case 'f':
		output = "\f"
	case 's':
		output = " "
	case 'E':
		output = "\x1b"
	case '\n':
		// Hidden newline.
	case '$':
		// Hidden marker.
	default:
		// XXX: We should probably allow falling through and return "\" here...
		return "", fmt.Errorf("escape cstyle: unknown escape character: %q", ch)
	}

	p.Next()
	return output, nil
}

func unvisEscapeHex(p *unvisParser) (string, error) {
	var output rune

	for i := 0; i < 2; i++ {
		ch, err := p.Peek()
		if err != nil {
			return "", fmt.Errorf("escape hex: %s", err)
		}

		digit, err := strconv.ParseInt(string(ch), 16, 32)
		if err != nil {
			return "", fmt.Errorf("escape hex: parse int: %s", err)
		}

		output = (output << 4) | rune(digit)
		p.Next()
	}

	// TODO: We need to handle runes properly to output byte strings again. In
	//       particular, if rune has 0xf0 set then we know that we're currently
	//       decoding a messed up string.
	return string(output), nil
}

func unvisEscapeOctal(p *unvisParser) (string, error) {
	var output rune
	var err error

	for i := 0; i < 3; i++ {
		ch, err := p.Peek()
		if err != nil {
			if i == 0 {
				err = fmt.Errorf("escape octal[first]: %s", err)
			}
			break
		}

		digit, err := strconv.ParseInt(string(ch), 8, 32)
		if err != nil {
			if i == 0 {
				err = fmt.Errorf("escape octal[first]: parse int: %s", err)
			}
			break
		}

		output = (output << 3) | rune(digit)
		p.Next()
	}

	// TODO: We need to handle runes properly to output byte strings again. In
	//       particular, if rune has 0xf0 set then we know that we're currently
	//       decoding a messed up string.
	return string(output), err
}

func unvisEscapeSequence(p *unvisParser) (string, error) {
	ch, err := p.Peek()
	if err != nil {
		return "", fmt.Errorf("escape sequence: %s", err)
	}

	switch ch {
	case '\\':
		p.Next()
		return "\\", nil

	case '0', '1', '2', '3', '4', '5', '6', '7':
		return unvisEscapeOctal(p)

	case 'x':
		p.Next()
		return unvisEscapeHex(p)

	case 'M':
		// TODO
	case '^':
		// TODO

	default:
		return unvisEscapeCStyle(p)
	}

	return "", fmt.Errorf("escape sequence: unsupported sequence: %q", ch)
}

func unvisRune(p *unvisParser) (string, error) {
	ch, err := p.Peek()
	if err != nil {
		return "", fmt.Errorf("rune: %s", err)
	}

	switch ch {
	case '\\':
		p.Next()
		return unvisEscapeSequence(p)

	case '%':
		// % HEX HEX only applies to HTTPStyle encodings.
		if p.flag&VisHTTPStyle == VisHTTPStyle {
			p.Next()
			return unvisEscapeHex(p)
		}
		fallthrough

	default:
		return unvisPlainRune(p)
	}
}

func unvis(p *unvisParser) (string, error) {
	output := ""
	for !p.End() {
		ch, err := unvisRune(p)
		if err != nil {
			return "", fmt.Errorf("input: %s", err)
		}
		output += ch
	}
	return output, nil
}

// Unvis takes a string formatted with the given Vis flags (though only the
// VisHTTPStyle flag is checked) and output the un-encoded version of the
// encoded string. An error is returned if any escape sequences in the input
// string were invalid.
func Unvis(input string, flag VisFlag) (string, error) {
	// TODO: Check all of the VisFlag bits.
	p := newParser(input, flag)
	output, err := unvis(p)
	if err != nil {
		return "", fmt.Errorf("unvis: %s", err)
	}
	if !p.End() {
		return "", fmt.Errorf("unvis: trailing characters at end of input")
	}
	return output, nil
}
