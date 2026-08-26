package store

import (
	"database/sql"
	"strings"
	"time"

	"task268-bronzeform/internal/model"
)

// RelationStore 演变关系与反证持久化。
type RelationStore struct{ db *DB }

func NewRelationStore(db *DB) *RelationStore { return &RelationStore{db: db} }

// Create 创建演变关系候选；(batch, source, target) 唯一，重复返回 ErrConflict。
func (s *RelationStore) Create(r *model.EvolutionRelation) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`INSERT INTO relations(batch_id, source_glyph, target_glyph, kind, status, chrono_score,
			added_parts, removed_parts, dir_changed, evidence, created_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		r.BatchID, r.SourceGlyph, r.TargetGlyph, string(r.Kind), string(r.Status), r.ChronoScore,
		joinList(r.AddedParts), joinList(r.RemovedParts), joinList(r.DirChanged), r.Evidence, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return model.ErrConflict
		}
		return err
	}
	id, _ := res.LastInsertId()
	r.ID = id
	r.CreatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

// Get 按 ID 查询关系。
func (s *RelationStore) Get(id int64) (*model.EvolutionRelation, error) {
	row := s.db.QueryRow(
		`SELECT id, batch_id, source_glyph, target_glyph, kind, status, chrono_score,
			added_parts, removed_parts, dir_changed, evidence, created_at, decided_at
		 FROM relations WHERE id=?`, id)
	var r model.EvolutionRelation
	var added, removed, dir, created string
	var decided sql.NullString
	if err := row.Scan(&r.ID, &r.BatchID, &r.SourceGlyph, &r.TargetGlyph, &r.Kind, &r.Status,
		&r.ChronoScore, &added, &removed, &dir, &r.Evidence, &created, &decided); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	r.CreatedAt, _ = time.Parse(time.RFC3339, created)
	r.AddedParts = splitList(added)
	r.RemovedParts = splitList(removed)
	r.DirChanged = splitList(dir)
	if decided.Valid {
		t, _ := time.Parse(time.RFC3339, decided.String)
		r.DecidedAt = &t
	}
	n, err := s.CountRebuttals(r.ID)
	if err != nil {
		return nil, err
	}
	r.RebutCount = n
	return &r, nil
}

// ListByBatch 列出批次内全部关系（含反证数）。
func (s *RelationStore) ListByBatch(batchID int64) ([]model.EvolutionRelation, error) {
	rows, err := s.db.Query(
		`SELECT id, batch_id, source_glyph, target_glyph, kind, status, chrono_score,
			added_parts, removed_parts, dir_changed, evidence, created_at, decided_at
		 FROM relations WHERE batch_id=? ORDER BY id`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.EvolutionRelation{}
	for rows.Next() {
		var r model.EvolutionRelation
		var added, removed, dir, created string
		var decided sql.NullString
		if err := rows.Scan(&r.ID, &r.BatchID, &r.SourceGlyph, &r.TargetGlyph, &r.Kind, &r.Status,
			&r.ChronoScore, &added, &removed, &dir, &r.Evidence, &created, &decided); err != nil {
			return nil, err
		}
		r.CreatedAt, _ = time.Parse(time.RFC3339, created)
		r.AddedParts = splitList(added)
		r.RemovedParts = splitList(removed)
		r.DirChanged = splitList(dir)
		if decided.Valid {
			t, _ := time.Parse(time.RFC3339, decided.String)
			r.DecidedAt = &t
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		n, err := s.CountRebuttals(out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].RebutCount = n
	}
	return out, nil
}

// UpdateStatus 更新关系状态并记录裁决时间。
func (s *RelationStore) UpdateStatus(id int64, status model.RelationStatus) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(`UPDATE relations SET status=?, decided_at=? WHERE id=?`, string(status), now, id)
	if err != nil {
		return err
	}
	_ = status
	n, _ := res.RowsAffected()
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// AddRebuttal 添加反证。
func (s *RelationStore) AddRebuttal(relationID int64, kind, note string) (*model.Rebuttal, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`INSERT INTO rebuttals(relation_id, kind, note, created_at) VALUES(?,?,?,?)`,
		relationID, kind, note, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	created, _ := time.Parse(time.RFC3339, now)
	return &model.Rebuttal{ID: id, RelationID: relationID, Kind: kind, Note: note, CreatedAt: created}, nil
}

// Rebuttals 查询关系的全部反证。
func (s *RelationStore) Rebuttals(relationID int64) ([]model.Rebuttal, error) {
	rows, err := s.db.Query(
		`SELECT id, relation_id, kind, note, created_at FROM rebuttals WHERE relation_id=? ORDER BY id`,
		relationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Rebuttal{}
	for rows.Next() {
		var r model.Rebuttal
		if err := rows.Scan(&r.ID, &r.RelationID, &r.Kind, &r.Note, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CountRebuttals 统计反证数。
func (s *RelationStore) CountRebuttals(relationID int64) (int, error) {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM rebuttals WHERE relation_id=?`, relationID).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// CountByBatch 统计批次内关系数量（按状态分组）。
func (s *RelationStore) CountByBatch(batchID int64) (total int, byStatus map[model.RelationStatus]int, err error) {
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM relations WHERE batch_id=?`, batchID).Scan(&total); err != nil {
		return 0, nil, err
	}
	byStatus = map[model.RelationStatus]int{}
	rows, err := s.db.Query(`SELECT status, COUNT(*) FROM relations WHERE batch_id=? GROUP BY status`, batchID)
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
		byStatus[model.RelationStatus(st)] = n
	}
	return total, byStatus, rows.Err()
}

func joinList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return strings.Join(items, ",")
}

func splitList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
