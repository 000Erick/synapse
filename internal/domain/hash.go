package domain

import (
	"crypto/sha256"
	"encoding/hex"
)

// ContentHash returns a stable hash of an observation's title and content,
// used to detect whether an observation changed since it was last embedded.
// Title and content are joined with a NUL separator so that moving text across
// the boundary still changes the hash.
func ContentHash(title, content string) string {
	h := sha256.New()
	h.Write([]byte(title))
	h.Write([]byte{0})
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))
}
