package fontbuild

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestBuildWOFF2(t *testing.T) {
	cfg := testConfig()
	ttf, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	w2, err := BuildWOFF2(ttf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(w2[:4], []byte("wOF2")) {
		t.Fatalf("bad signature: %q", w2[:4])
	}
	if got := binary.BigEndian.Uint32(w2[8:]); int(got) != len(w2) {
		t.Fatalf("length %d != %d", got, len(w2))
	}
	numTables := int(binary.BigEndian.Uint16(w2[12:]))
	sfntTables := int(binary.BigEndian.Uint16(ttf[4:]))
	if numTables != sfntTables {
		t.Fatalf("woff2 tables %d != ttf tables %d", numTables, sfntTables)
	}
	// Directory ends before the compressed payload; every table used the null
	// transform, so transformLength is omitted from each entry.
	off := 48
	for i := 0; i < numTables; i++ {
		flags := w2[off]
		off++ // flags
		tag := woff2KnownTags[flags&0x3F]
		if flags&0x3F == 63 {
			tag = string(w2[off : off+4])
			off += 4
		}
		if tag == "" {
			t.Fatalf("empty tag at table %d", i)
		}
		_, n := readBase128(w2[off:])
		if n == 0 {
			t.Fatalf("bad origLength for %q", tag)
		}
		off += n
	}
	if off > len(w2) {
		t.Fatalf("directory overran payload")
	}
}

func TestBuildWOFF2Deterministic(t *testing.T) {
	cfg := testConfig()
	ttf, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	a, err := BuildWOFF2(ttf)
	if err != nil {
		t.Fatal(err)
	}
	b, err := BuildWOFF2(ttf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("WOFF2 output not deterministic")
	}
}

func readBase128(b []byte) (uint32, int) {
	var v uint32
	for i := 0; i < 5; i++ {
		if i >= len(b) {
			return 0, 0
		}
		v = v<<7 | uint32(b[i]&0x7F)
		if b[i]&0x80 == 0 {
			return v, i + 1
		}
	}
	return 0, 0
}
