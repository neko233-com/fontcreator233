package skeleton

import "testing"

func TestGreekCoverage(t *testing.T) {
	b := builder(t, nil)
	uppercase := []rune("ΑΒΓΔΕΖΗΘΙΚΛΜΝΞΟΠΡΣΤΥΦΧΨΩ")
	lowercase := []rune("αβγδεζηθικλμνξοπρστυφχψω")
	for _, r := range append(uppercase, lowercase...) {
		p, adv, ok, err := b.Glyph(r)
		if err != nil {
			t.Fatalf("%U: %v", r, err)
		}
		if !ok || len(p) == 0 || adv <= 0 {
			t.Fatalf("%U missing/empty (adv=%v)", r, adv)
		}
	}
	if _, _, ok, _ := b.Glyph('ς'); !ok {
		t.Fatal("final sigma missing")
	}
}

func TestCyrillicCoverage(t *testing.T) {
	b := builder(t, nil)
	uppercase := []rune("АБВГДЕЖЗИКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯ")
	lowercase := []rune("абвгдежзиклмнопрстуфхцчшщъыьэюя")
	for _, r := range append(uppercase, lowercase...) {
		p, adv, ok, err := b.Glyph(r)
		if err != nil {
			t.Fatalf("%U: %v", r, err)
		}
		if !ok || len(p) == 0 || adv <= 0 {
			t.Fatalf("%U missing/empty (adv=%v)", r, adv)
		}
	}
	for _, r := range []rune{'Є', 'є', 'І', 'і', 'Ґ', 'ґ'} {
		if _, _, ok, _ := b.Glyph(r); !ok {
			t.Fatalf("%U missing", r)
		}
	}
}
