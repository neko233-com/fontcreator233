package assets

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"golang.org/x/image/font/sfnt"
)

func TestEmbeddedBaseFonts(t *testing.T) {
	if len(BaseSC) < 1_000_000 {
		t.Fatalf("SC base too small: %d bytes", len(BaseSC))
	}
	if len(BaseKR) < 1_000_000 {
		t.Fatalf("KR base too small: %d bytes", len(BaseKR))
	}

	sc, err := sfnt.Parse(BaseSC)
	if err != nil {
		t.Fatalf("parse SC: %v", err)
	}
	kr, err := sfnt.Parse(BaseKR)
	if err != nil {
		t.Fatalf("parse KR: %v", err)
	}

	var buf sfnt.Buffer
	scName, err := sc.Name(&buf, sfnt.NameIDFamily)
	if err != nil || scName != "fontcreator233 Base SC" {
		t.Fatalf("SC family = %q, err=%v", scName, err)
	}
	krName, err := kr.Name(&buf, sfnt.NameIDFamily)
	if err != nil || krName != "fontcreator233 Base KR" {
		t.Fatalf("KR family = %q, err=%v", krName, err)
	}

	if sc.NumGlyphs() < 30_000 {
		t.Fatalf("SC glyphs = %d, want >= 30000", sc.NumGlyphs())
	}
	if kr.NumGlyphs() < 10_000 {
		t.Fatalf("KR glyphs = %d, want >= 10000", kr.NumGlyphs())
	}

	assertHas := func(f *sfnt.Font, r rune, want bool) {
		t.Helper()
		gid, err := f.GlyphIndex(&buf, r)
		if err != nil {
			t.Fatalf("GlyphIndex(%q): %v", r, err)
		}
		if got := gid != 0; got != want {
			t.Fatalf("GlyphIndex(%q) present=%v, want %v (gid=%d)", r, got, want, gid)
		}
	}
	assertHas(sc, '中', true)
	assertHas(sc, 'あ', true)
	assertHas(sc, '한', false)
	assertHas(kr, '한', true)
	assertHas(kr, 'A', false)

	if !bytes.Contains(OFL, []byte("SIL OPEN FONT LICENSE")) {
		t.Fatal("embedded OFL text missing license header")
	}

	sum := sha256.Sum256(BaseSC)
	if got := hex.EncodeToString(sum[:]); got != BaseSCHash {
		t.Fatalf("BaseSC sha256 = %s, want %s", got, BaseSCHash)
	}
	sum = sha256.Sum256(BaseKR)
	if got := hex.EncodeToString(sum[:]); got != BaseKRHash {
		t.Fatalf("BaseKR sha256 = %s, want %s", got, BaseKRHash)
	}
}
