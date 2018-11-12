package image

import (
	"crypto"
	"fmt"
	"hash"
	"io"
)

// SHA256 is the only algorithm supported.
var SHA256 = "sha256"

// Digester calculates the digest of written data.
type Digester struct {
	hash hash.Hash
}

// NewDigester instantiates and returns a new Digester object.
func NewDigester() *Digester {
	return &Digester{
		hash: crypto.SHA256.New(),
	}
}

// Digest returns the digest of existing data.
func (d *Digester) Digest() Digest {
	return Digest(fmt.Sprintf("%s:%x", SHA256, d.hash.Sum(nil)))
}

// FromReader returns the digest of data from reader.
func (d Digester) FromReader(rd io.Reader) (Digest, error) {
	if _, err := io.Copy(d.hash, rd); err != nil {
		return "", err
	}

	return d.Digest(), nil
}

// FromBytes digests the input and returns a Digest.
func (d Digester) FromBytes(p []byte) (Digest, error) {
	if _, err := d.hash.Write(p); err != nil {
		return "", err
	}

	return d.Digest(), nil
}
