package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.10.3", "0.10.4", -1},
		{"v0.10.4", "0.10.3", 1},
		{"1.0.0", "1.0.0", 0},
		{"0.9.0", "0.10.0", -1},
		{"2.0", "2.0.1", -1},
	}
	for _, c := range cases {
		if got := compareSemver(c.a, c.b); got != c.want {
			t.Errorf("compareSemver(%q,%q)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

// The newest tag is resolved from the un-followed 302 of /releases/latest.
func TestTagViaRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/releases/latest") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		http.Redirect(w, r, "https://github.com/neko233-com/fontcreator233/releases/tag/v1.2.3", http.StatusFound)
	}))
	defer srv.Close()

	a := &app{ctx: context.Background()}
	tag, err := a.tagViaRedirect(srv.URL + "/neko233-com/fontcreator233/releases/latest")
	if err != nil {
		t.Fatalf("tagViaRedirect: %v", err)
	}
	if tag != "v1.2.3" {
		t.Fatalf("tag=%q want v1.2.3", tag)
	}
}

func TestTagViaRedirectNoLocation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := &app{ctx: context.Background()}
	if _, err := a.tagViaRedirect(srv.URL); err == nil {
		t.Fatal("expected error when Location is absent")
	}
}

func TestIsHex(t *testing.T) {
	if !isHex("0123456789abcdefABCDEF") {
		t.Fatal("valid hex rejected")
	}
	if isHex("xyz") || isHex("") {
		t.Fatal("invalid hex accepted")
	}
}

func TestVerifySHA(t *testing.T) {
	// sha256("abc")
	sum := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if !verifySHA([]byte("abc"), sum) {
		t.Fatal("matching checksum rejected")
	}
	if !verifySHA([]byte("abc"), strings.ToUpper(sum)) {
		t.Fatal("uppercase checksum should match")
	}
	if verifySHA([]byte("abc"), "deadbeef") {
		t.Fatal("mismatched checksum accepted")
	}
	if !verifySHA([]byte("abc"), "") {
		t.Fatal("empty checksum should be tolerated")
	}
}

func TestArchiveForUsesGoOSNaming(t *testing.T) {
	_, archive, ext := archiveFor("v0.10.4")
	if !strings.HasPrefix(archive, "fontcreator233-v0.10.4-") {
		t.Fatalf("unexpected archive name %q", archive)
	}
	if !strings.HasSuffix(archive, ".tar.gz") || ext != "tar.gz" {
		t.Fatalf("unexpected archive %q ext %q", archive, ext)
	}
}
