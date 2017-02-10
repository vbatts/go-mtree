package mtree

import "testing"

type runeCheck func(rune) bool

func TestUnvisHelpers(t *testing.T) {
	testset := []struct {
		R      rune
		Check  runeCheck
		Expect bool
	}{
		{'a', ishex, true},
		{'A', ishex, true},
		{'z', ishex, false},
		{'Z', ishex, false},
		{'G', ishex, false},
		{'1', ishex, true},
		{'0', ishex, true},
		{'9', ishex, true},
		{'0', isoctal, true},
		{'3', isoctal, true},
		{'7', isoctal, true},
		{'9', isoctal, false},
		{'a', isoctal, false},
		{'z', isoctal, false},
		{'3', isalnum, true},
		{'a', isalnum, true},
		{';', isalnum, false},
		{'!', isalnum, false},
		{' ', isalnum, false},
		{'3', isgraph, true},
		{'a', isgraph, true},
		{';', isgraph, true},
		{'!', isgraph, true},
		{' ', isgraph, false},
	}

	for i, ts := range testset {
		got := ts.Check(ts.R)
		if got != ts.Expect {
			t.Errorf("%d: %q expected: %t; got %t", i, string(ts.R), ts.Expect, got)
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
		got, err := Unvis(test)
		if err != nil {
			t.Errorf("unexpected error doing unvis(%q): %s", test, err)
			continue
		}
		if got != test {
			t.Errorf("expected %q to be unchanged, got %q", test, got)
		}
	}
}

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
		dec, err := Unvis(enc)
		if err != nil {
			t.Errorf("unexpected error doing unvis(%q): %s", enc, err)
			continue
		}
		if dec != test {
			t.Errorf("roundtrip failed: unvis(vis(%q) = %q) = %q", test, enc, dec)
		}
	}
}
