package version

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// versionHash 版本内容哈希构建器。
type versionHash struct {
	buf []byte
}

func newVersionHash() *versionHash { return &versionHash{} }

func (v *versionHash) add(s string) *versionHash {
	v.buf = append(v.buf, []byte(s)...)
	v.buf = append(v.buf, '|')
	return v
}

func (v *versionHash) addInt(n int64) *versionHash {
	return v.add(fmt.Sprintf("%d", n))
}

func (v *versionHash) sum() string {
	h := sha256.New()
	h.Write(v.buf)
	return hex.EncodeToString(h.Sum(nil))
}
