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

const DefaultVisFlags = VisWhite | VisOctal | VisGlob

func TestVisUnvis(t *testing.T) {
	// Round-trip testing.
	for _, test := range []string{
		"",
		"this.is.a.normal_string",
		"AC_Raíz_Certicámara_S.A..pem",
		"NetLock_Arany_=Class_Gold=_Főtanúsítvány.pem",
		"TÜBİTAK_UEKAE_Kök_Sertifika_Hizmet_Sağlayıcısı_-_Sürüm_3.pem",
		"hello world [ this string needs=enco ding! ]",
		"even \n more encoding necessary\a\a ",
		"\024 <-- some more weird characters --> 你好，世界",
	} {
		enc, err := Vis(test, DefaultVisFlags)
		if err != nil {
			t.Errorf("unexpected error doing vis(%q): %s", test, err)
			continue
		}
		dec, err := Unvis(enc, DefaultVisFlags)
		if err != nil {
			t.Errorf("unexpected error doing unvis(%q): %s", enc, err)
			continue
		}
		if dec != test {
			t.Errorf("roundtrip failed: unvis(vis(%q) = %q) = %q", test, enc, dec)
		}
	}
}
