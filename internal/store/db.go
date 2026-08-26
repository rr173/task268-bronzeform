// Package store 提供 SQLite 持久化：建表迁移与各实体 CRUD。
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB 封装 SQLite 连接与迁移。
type DB struct {
	*sql.DB
	Path string
}

// Open 打开（或创建）数据库并执行迁移。
func Open(path string) (*DB, error) {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("mkdir db dir: %w", err)
		}
	}
	// _pragma=busy_timeout(5000)&_pragma=journal_mode(WAL) 由 modernc 驱动支持，
	// 保证并发读写与崩溃恢复语义。
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", path)
	raw, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	raw.SetMaxOpenConns(1) // SQLite 单写者，串行化写路径
	db := &DB{DB: raw, Path: path}
	if err := db.migrate(); err != nil {
		raw.Close()
		return nil, err
	}
	return db, nil
}

// migrate 建表（幂等：CREATE TABLE IF NOT EXISTS）。
func (db *DB) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS batches (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS glyphs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL REFERENCES batches(id),
			code TEXT NOT NULL,
			graph TEXT NOT NULL,
			era_begin INTEGER NOT NULL,
			era_end INTEGER NOT NULL,
			source TEXT NOT NULL,
			status TEXT NOT NULL,
			fingerprint TEXT NOT NULL UNIQUE,
			component_sum TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			UNIQUE(batch_id, code)
		)`,
		`CREATE TABLE IF NOT EXISTS components (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			glyph_id INTEGER NOT NULL REFERENCES glyphs(id),
			part TEXT NOT NULL,
			position TEXT NOT NULL DEFAULT '',
			direction TEXT NOT NULL DEFAULT 'normal',
			seq INTEGER NOT NULL,
			UNIQUE(glyph_id, seq)
		)`,
		`CREATE TABLE IF NOT EXISTS relations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL REFERENCES batches(id),
			source_glyph INTEGER NOT NULL REFERENCES glyphs(id),
			target_glyph INTEGER NOT NULL REFERENCES glyphs(id),
			kind TEXT NOT NULL,
			status TEXT NOT NULL,
			chrono_score INTEGER NOT NULL DEFAULT 0,
			added_parts TEXT NOT NULL DEFAULT '',
			removed_parts TEXT NOT NULL DEFAULT '',
			dir_changed TEXT NOT NULL DEFAULT '',
			evidence TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			decided_at TEXT,
			UNIQUE(batch_id, source_glyph, target_glyph)
		)`,
		`CREATE TABLE IF NOT EXISTS rebuttals (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			relation_id INTEGER NOT NULL REFERENCES relations(id),
			kind TEXT NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id INTEGER NOT NULL REFERENCES batches(id),
			name TEXT NOT NULL,
			status TEXT NOT NULL,
			content_hash TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			frozen_at TEXT,
			superseded_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS version_relations (
			version_id INTEGER NOT NULL REFERENCES versions(id),
			relation_id INTEGER NOT NULL REFERENCES relations(id),
			PRIMARY KEY (version_id, relation_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_glyphs_batch ON glyphs(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_relations_batch ON relations(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_rebuttals_rel ON rebuttals(relation_id)`,
		`CREATE INDEX IF NOT EXISTS idx_versions_batch ON versions(batch_id)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

// Close 关闭连接。
func (db *DB) Close() error { return db.DB.Close() }
