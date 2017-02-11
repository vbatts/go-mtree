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
	"testing"
)

func TestUnvisOctalEscape(t *testing.T) {
	for _, test := range []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"\\1", "\001"},
		{"\\01\\02\\3", "\001\002\003"},
		{"\\001\\023\\32", "\001\023\032"},
		{"this is a test\\0k1\\133", "this is a test\000k1\133"},
		{"\\170YET\\01another test\\1\\\\82", "\170YET\001another test\001\\82"},
		{"\\177MORE tests\\09a", "\177MORE tests\x009a"},
		{"\\\\710more\\1215testing", "\\710more\1215testing"},
	} {
		got, err := Unvis(test.input, DefaultVisFlags)
		if err != nil {
			t.Errorf("unexpected error doing unvis(%q): %q", test.input, err)
			continue
		}
		if got != test.expected {
			t.Errorf("expected unvis(%q) = %q, got %q", test.input, test.expected, got)
		}
	}
}

func TestUnvisHexEscape(t *testing.T) {
	for _, test := range []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"\\x01", "\x01"},
		{"\\x01\\x02\\x7a", "\x01\x02\x7a"},
		{"this is a test\\x13\\x52\\x6f", "this is a test\x13\x52\x6f"},
		{"\\x170YET\\x01a\\x22nother test\\x11", "\x170YET\x01a\x22nother test\x11"},
		{"\\\\x007more\\\\x215testing", "\\x007more\\x215testing"},
		// Make sure that decoding unicode works properly.
		{"\\xf0\\x9f\\x95\\xb4", "\U0001f574"},
	} {
		got, err := Unvis(test.input, DefaultVisFlags)
		if err != nil {
			t.Errorf("unexpected error doing unvis(%q): %q", test.input, err)
			continue
		}
		if got != test.expected {
			t.Errorf("expected unvis(%q) = %q, got %q", test.input, test.expected, got)
		}
	}
}

func TestUnvisUnicode(t *testing.T) {
	// Ensure that unicode strings are not messed up by Unvis.
	for _, test := range []string{
		"",
		"this.is.a.normal_string",
		"AC_Raíz_Certicámara_S.A..pem",
		"NetLock_Arany_=Class_Gold=_Főtanúsítvány.pem",
		"TÜBİTAK_UEKAE_Kök_Sertifika_Hizmet_Sağlayıcısı_-_Sürüm_3.pem",
	} {
		got, err := Unvis(test, DefaultVisFlags)
		if err != nil {
			t.Errorf("unexpected error doing unvis(%q): %s", test, err)
			continue
		}
		if got != test {
			t.Errorf("expected %q to be unchanged, got %q", test, got)
		}
	}
}
