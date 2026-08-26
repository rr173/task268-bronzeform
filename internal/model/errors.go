package model

import "errors"

// 业务错误集合。所有可预期失败都映射到这些哨兵错误，
// 由 HTTP 层转换为对应状态码，store 层负责校验唯一约束。
var (
	ErrNotFound          = errors.New("record not found")
	ErrConflict          = errors.New("conflict: record already exists")
	ErrInvalidState      = errors.New("invalid state transition")
	ErrBadInput          = errors.New("bad input")
	ErrForbidden         = errors.New("forbidden: frozen or sealed object")
	ErrSelfReference     = errors.New("source and target glyph must differ")
	ErrEraInverted       = errors.New("era interval inverted: begin > end")
	ErrMissingSource     = errors.New("artifact source is required")
	ErrComponentCycle    = errors.New("component cannot contain itself")
	ErrFingerprintExists = errors.New("glyph observation with same fingerprint already exists")
	ErrNoActiveRelation  = errors.New("no active relation between the two glyphs")
	ErrVersionNotEmpty   = errors.New("cannot delete version with relations")
)
