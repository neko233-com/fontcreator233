package unicodes

import "testing"

func TestScriptOf(t *testing.T) {
	tests := map[rune]string{
		'A':     "latin",
		'é':     "latin",
		'Ω':     "greek",
		'Ж':     "cyrillic",
		'中':     "cjk",
		'あ':     "kana",
		'한':     "hangul",
		'€':     "symbols",
		0x1F600: "other",
		0x20000: "cjk",
	}
	for r, want := range tests {
		if got := ScriptOf(r); got != want {
			t.Fatalf("%U = %q, want %q", r, got, want)
		}
	}
}
