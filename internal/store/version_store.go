package store

import (
	"database/sql"
	"time"

	"task268-bronzeform/internal/model"
)

// VersionStore 演变版本持久化。
type VersionStore struct{ db *DB }

func NewVersionStore(db *DB) *VersionStore { return &VersionStore{db: db} }

// Create 创建版本草稿。
func (s *VersionStore) Create(v *model.EvolutionVersion) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`INSERT INTO versions(batch_id, name, status, content_hash, created_at) VALUES(?,?,?,?,?)`,
		v.BatchID, v.Name, string(model.VersionDraft), v.ContentHash, now)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	v.ID = id
	v.Status = model.VersionDraft
	v.CreatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

// Get 按 ID 查询版本（含关系快照）。
func (s *VersionStore) Get(id int64) (*model.EvolutionVersion, error) {
	row := s.db.QueryRow(
		`SELECT id, batch_id, name, status, content_hash, created_at, frozen_at, superseded_at
		 FROM versions WHERE id=?`, id)
	var v model.EvolutionVersion
	var created string
	var frozen, superseded sql.NullString
	if err := row.Scan(&v.ID, &v.BatchID, &v.Name, &v.Status, &v.ContentHash,
		&created, &frozen, &superseded); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	v.CreatedAt, _ = time.Parse(time.RFC3339, created)
	if frozen.Valid {
		t, _ := time.Parse(time.RFC3339, frozen.String)
		v.FrozenAt = &t
	}
	if superseded.Valid {
		t, _ := time.Parse(time.RFC3339, superseded.String)
		v.SupersededAt = &t
	}
	ids, err := s.RelationIDs(id)
	if err != nil {
		return nil, err
	}
	v.RelationIDs = ids
	return &v, nil
}

// ListByBatch 列出批次内全部版本。
func (s *VersionStore) ListByBatch(batchID int64) ([]model.EvolutionVersion, error) {
	rows, err := s.db.Query(
		`SELECT id, batch_id, name, status, content_hash, created_at, frozen_at, superseded_at
		 FROM versions WHERE batch_id=? ORDER BY id`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.EvolutionVersion{}
	for rows.Next() {
		var v model.EvolutionVersion
		var created string
		var frozen, superseded sql.NullString
		if err := rows.Scan(&v.ID, &v.BatchID, &v.Name, &v.Status, &v.ContentHash,
			&created, &frozen, &superseded); err != nil {
			return nil, err
		}
		v.CreatedAt, _ = time.Parse(time.RFC3339, created)
		if frozen.Valid {
			t, _ := time.Parse(time.RFC3339, frozen.String)
			v.FrozenAt = &t
		}
		if superseded.Valid {
			t, _ := time.Parse(time.RFC3339, superseded.String)
			v.SupersededAt = &t
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		ids, err := s.RelationIDs(out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].RelationIDs = ids
	}
	return out, nil
}

// UpdateStatus 更新版本状态；冻结时记录 frozen_at，替代时记录 superseded_at。
func (s *VersionStore) UpdateStatus(id int64, status model.VersionStatus) error {
	var setClause string
	switch status {
	case model.VersionFrozen:
		setClause = `SET status=?, frozen_at=?`
	case model.VersionSuperseded:
		setClause = `SET status=?, superseded_at=?`
	default:
		setClause = `SET status=?`
	}
	now := time.Now().UTC().Format(time.RFC3339)
	var res sql.Result
	var err error
	if setClause == `SET status=?` {
		res, err = s.db.Exec(`UPDATE versions `+setClause+` WHERE id=?`, string(status), id)
	} else {
		res, err = s.db.Exec(`UPDATE versions `+setClause+` WHERE id=?`, string(status), now, id)
	}
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// SetRelations 覆盖式写入版本关系快照（先删后插）。
func (s *VersionStore) SetRelations(versionID int64, relationIDs []int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM version_relations WHERE version_id=?`, versionID); err != nil {
		return err
	}
	for _, rid := range relationIDs {
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO version_relations(version_id, relation_id) VALUES(?,?)`,
			versionID, rid); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// RelationIDs 查询版本的关系快照 ID 列表。
func (s *VersionStore) RelationIDs(versionID int64) ([]int64, error) {
	rows, err := s.db.Query(
		`SELECT relation_id FROM version_relations WHERE version_id=? ORDER BY relation_id`, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// HashContent 计算版本内容哈希（供不可变冻结校验）。
func (s *VersionStore) HashContent(batchID int64, relationIDs []int64) (string, error) {
	rows, err := s.db.Query(
		`SELECT id, source_glyph, target_glyph, kind, status, chrono_score, evidence
		 FROM relations WHERE batch_id=?`, batchID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	h := newContentHash()
	for rows.Next() {
		var id, src, tgt, score int64
		var kind, status, evidence string
		if err := rows.Scan(&id, &src, &tgt, &kind, &status, &score, &evidence); err != nil {
			return "", err
		}
		h.addInt(id).addInt(src).addInt(tgt).add(kind).add(status).add(evidence)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return h.sum(), nil
}
