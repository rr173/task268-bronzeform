package store

import (
	"database/sql"
	"strings"
	"time"

	"task268-bronzeform/internal/model"
)

// GlyphStore 字形观察与构件持久化。
type GlyphStore struct{ db *DB }

func NewGlyphStore(db *DB) *GlyphStore { return &GlyphStore{db: db} }

// Create 创建字形观察；指纹重复返回 ErrFingerprintExists，批次内 code 冲突返回 ErrConflict。
func (s *GlyphStore) Create(g *model.Glyph) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`INSERT INTO glyphs(batch_id, code, graph, era_begin, era_end, source, status, fingerprint, component_sum, created_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?)`,
		g.BatchID, g.Code, g.Graph, g.EraBegin, g.EraEnd, g.Source,
		string(model.GlyphPendingSplit), g.Fingerprint, g.ComponentSum, now)
	if err != nil {
		if strings.Contains(err.Error(), "fingerprint") {
			return model.ErrFingerprintExists
		}
		if strings.Contains(err.Error(), "UNIQUE") {
			return model.ErrConflict
		}
		return err
	}
	id, _ := res.LastInsertId()
	g.ID = id
	g.Status = model.GlyphPendingSplit
	g.CreatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

// Get 按 ID 查询字形（含构件）。
func (s *GlyphStore) Get(id int64) (*model.Glyph, error) {
	row := s.db.QueryRow(
		`SELECT id, batch_id, code, graph, era_begin, era_end, source, status, fingerprint, component_sum, created_at
		 FROM glyphs WHERE id=?`, id)
	var g model.Glyph
	var created string
	if err := row.Scan(&g.ID, &g.BatchID, &g.Code, &g.Graph, &g.EraBegin, &g.EraEnd,
		&g.Source, &g.Status, &g.Fingerprint, &g.ComponentSum, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	g.CreatedAt, _ = time.Parse(time.RFC3339, created)
	comps, err := s.Components(g.ID)
	if err != nil {
		return nil, err
	}
	g.Components = comps
	return &g, nil
}

// ListByBatch 列出批次内全部字形。
func (s *GlyphStore) ListByBatch(batchID int64) ([]model.Glyph, error) {
	rows, err := s.db.Query(
		`SELECT id, batch_id, code, graph, era_begin, era_end, source, status, fingerprint, component_sum, created_at
		 FROM glyphs WHERE batch_id=? ORDER BY id`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Glyph{}
	for rows.Next() {
		var g model.Glyph
		var created string
		if err := rows.Scan(&g.ID, &g.BatchID, &g.Code, &g.Graph, &g.EraBegin, &g.EraEnd,
			&g.Source, &g.Status, &g.Fingerprint, &g.ComponentSum, &created); err != nil {
			return nil, err
		}
		g.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, g)
	}
	return out, rows.Err()
}

// ListAll 列出全部字形（用于指纹去重审计）。
func (s *GlyphStore) ListAll() ([]model.Glyph, error) {
	rows, err := s.db.Query(
		`SELECT id, batch_id, code, graph, era_begin, era_end, source, status, fingerprint, component_sum, created_at
		 FROM glyphs ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Glyph{}
	for rows.Next() {
		var g model.Glyph
		var created string
		if err := rows.Scan(&g.ID, &g.BatchID, &g.Code, &g.Graph, &g.EraBegin, &g.EraEnd,
			&g.Source, &g.Status, &g.Fingerprint, &g.ComponentSum, &created); err != nil {
			return nil, err
		}
		g.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, g)
	}
	return out, rows.Err()
}

// UpdateStatus 更新字形状态（pending_split → valid / defective / excluded）。
func (s *GlyphStore) UpdateStatus(id int64, status model.GlyphStatus) error {
	res, err := s.db.Exec(`UPDATE glyphs SET status=? WHERE id=?`, string(status), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// SaveComponents 覆盖式保存构件列表（先删后插，事务内保证一致性）。
func (s *GlyphStore) SaveComponents(glyphID int64, comps []model.Component) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM components WHERE glyph_id=?`, glyphID); err != nil {
		return err
	}
	for i, c := range comps {
		c.Seq = i
		if _, err := tx.Exec(
			`INSERT INTO components(glyph_id, part, position, direction, seq) VALUES(?,?,?,?,?)`,
			glyphID, c.Part, c.Position, string(c.Direction), i); err != nil {
			return err
		}
	}
	// 同步更新构件摘要，保证指纹输入一致。
	sum := model.ComponentSumOf(comps)
	if _, err := tx.Exec(`UPDATE glyphs SET component_sum=? WHERE id=?`, sum, glyphID); err != nil {
		return err
	}
	return tx.Commit()
}

// Components 查询字形构件列表。
func (s *GlyphStore) Components(glyphID int64) ([]model.Component, error) {
	rows, err := s.db.Query(
		`SELECT id, glyph_id, part, position, direction, seq FROM components WHERE glyph_id=? ORDER BY seq`,
		glyphID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Component{}
	for rows.Next() {
		var c model.Component
		if err := rows.Scan(&c.ID, &c.GlyphID, &c.Part, &c.Position, &c.Direction, &c.Seq); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CountByBatch 统计批次内字形数量（按状态分组）。
func (s *GlyphStore) CountByBatch(batchID int64) (total int, byStatus map[model.GlyphStatus]int, err error) {
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM glyphs WHERE batch_id=?`, batchID).Scan(&total); err != nil {
		return 0, nil, err
	}
	byStatus = map[model.GlyphStatus]int{}
	rows, err := s.db.Query(`SELECT status, COUNT(*) FROM glyphs WHERE batch_id=? GROUP BY status`, batchID)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var st string
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			return 0, nil, err
		}
		byStatus[model.GlyphStatus(st)] = n
	}
	return total, byStatus, rows.Err()
}
