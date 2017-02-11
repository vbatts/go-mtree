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

func TestVisUnchanged(t *testing.T) {
	for _, test := range []string{
		"helloworld",
		"THIS_IS_A_TEST1234",
		"SomeEncodingsAreCool",
		"AC_Raíz_Certicámara_S.A..pem",
	} {
		enc, err := Vis(test, DefaultVisFlags)
		if err != nil {
			t.Errorf("unexpected error with %q: %s", test, err)
		}
		if enc != test {
			t.Errorf("expected encoding of %q to be unchanged, got %q", test, enc)
		}
	}
}

func TestVisChanged(t *testing.T) {
	for _, test := range []string{
		"hello world",
		"THIS\\IS_A_TEST1234",
	} {
		enc, err := Vis(test, DefaultVisFlags)
		if err != nil {
			t.Errorf("unexpected error with %q: %s", test, err)
		}
		if enc == test {
			t.Errorf("expected encoding of %q to be changed")
		}
	}
}
