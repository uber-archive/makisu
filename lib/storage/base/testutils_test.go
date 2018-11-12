package base

import (
	"io"
	"io/ioutil"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/uber/makisu/lib/storage/metadata"
)

// Mock metadata
func init() {
	metadata.Register(regexp.MustCompile("_mocksuffix_\\w+"), &mockMetadataFactory{})
	metadata.Register(regexp.MustCompile("_mockmovable"), &mockMetadataFactoryMovable{})
}

type mockMetadataFactory struct{}

func (f mockMetadataFactory) Create(suffix string) metadata.Metadata {
	if strings.HasSuffix(suffix, getMockMetadataOne().GetSuffix()) {
		return getMockMetadataOne()
	}
	if strings.HasSuffix(suffix, getMockMetadataTwo().GetSuffix()) {
		return getMockMetadataTwo()
	}
	return nil
}

type mockMetadata struct {
	randomSuffix string
	content      []byte
}

func getMockMetadataOne() *mockMetadata {
	return &mockMetadata{
		randomSuffix: "_mocksuffix_one",
	}
}

func getMockMetadataTwo() *mockMetadata {
	return &mockMetadata{
		randomSuffix: "_mocksuffix_two",
	}
}

func (m *mockMetadata) GetSuffix() string {
	return m.randomSuffix
}

func (m *mockMetadata) Movable() bool {
	return false
}

func (m *mockMetadata) Serialize() ([]byte, error) {
	return m.content, nil
}

func (m *mockMetadata) Deserialize(b []byte) error {
	m.content = b
	return nil
}

type mockMetadataFactoryMovable struct{}

func (f mockMetadataFactoryMovable) Create(suffix string) metadata.Metadata {
	return getMockMetadataMovable()
}

type mockMetadataMovable struct {
	randomSuffix string
	content      []byte
}

func getMockMetadataMovable() *mockMetadataMovable {
	return &mockMetadataMovable{
		randomSuffix: "_mockmovable",
	}
}

func (m *mockMetadataMovable) GetSuffix() string {
	return m.randomSuffix
}

func (m *mockMetadataMovable) Movable() bool {
	return true
}

func (m *mockMetadataMovable) Serialize() ([]byte, error) {
	return m.content, nil
}

func (m *mockMetadataMovable) Deserialize(b []byte) error {
	m.content = b
	return nil
}

// blobFixture returns randomly generated blob data of length n.
func blobFixture(n uint64) []byte {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	lr := io.LimitReader(r, int64(n))
	b, _ := ioutil.ReadAll(lr)

	return b
}
