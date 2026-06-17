package domain

import "testing"

func TestContentHash_Deterministic(t *testing.T) {
	a := ContentHash("title", "body")
	b := ContentHash("title", "body")
	if a != b {
		t.Errorf("hash not deterministic: %q != %q", a, b)
	}
}

func TestContentHash_ChangesWithContent(t *testing.T) {
	a := ContentHash("title", "body")
	b := ContentHash("title", "body2")
	if a == b {
		t.Error("hash should change when content changes")
	}
}

func TestContentHash_SeparatorMatters(t *testing.T) {
	// "ab"+""  vs  "a"+"b" must differ thanks to the NUL separator.
	a := ContentHash("ab", "")
	b := ContentHash("a", "b")
	if a == b {
		t.Error("hash should distinguish title/content boundary")
	}
}
