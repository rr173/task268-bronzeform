package store

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// contentHash 链式哈希，用于版本内容指纹。
type contentHash struct {
	buf []byte
}

func newContentHash() *contentHash { return &contentHash{} }

func (c *contentHash) add(s string) *contentHash {
	c.buf = append(c.buf, []byte(s)...)
	c.buf = append(c.buf, '|')
	return c
}

func (c *contentHash) addInt(v int64) *contentHash {
	return c.add(fmt.Sprintf("%d", v))
}

func (c *contentHash) sum() string {
	h := sha256.New()
	h.Write(c.buf)
	return hex.EncodeToString(h.Sum(nil))
}
