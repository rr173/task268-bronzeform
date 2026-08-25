package store

import (
	"database/sql"
	"strings"
	"time"

	"task268-bronzeform/internal/model"
)

// BatchStore 铭文批次持久化。
type BatchStore struct{ db *DB }

func NewBatchStore(db *DB) *BatchStore { return &BatchStore{db: db} }

// Create 创建批次；code 冲突返回 ErrConflict。
func (s *BatchStore) Create(b *model.Batch) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`INSERT INTO batches(code, name, status, created_at, updated_at) VALUES(?,?,?,?,?)`,
		b.Code, b.Name, string(model.BatchOrganizing), now, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return model.ErrConflict
		}
		return err
	}
	id, _ := res.LastInsertId()
	b.ID = id
	b.Status = model.BatchOrganizing
	b.CreatedAt, _ = time.Parse(time.RFC3339, now)
	b.UpdatedAt = b.CreatedAt
	return nil
}

// Get 按 ID 查询批次。
func (s *BatchStore) Get(id int64) (*model.Batch, error) {
	row := s.db.QueryRow(`SELECT id, code, name, status, created_at, updated_at FROM batches WHERE id=?`, id)
	var b model.Batch
	var created, updated string
	if err := row.Scan(&b.ID, &b.Code, &b.Name, &b.Status, &created, &updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	b.CreatedAt, _ = time.Parse(time.RFC3339, created)
	b.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &b, nil
}

// GetByCode 按 code 查询批次。
func (s *BatchStore) GetByCode(code string) (*model.Batch, error) {
	row := s.db.QueryRow(`SELECT id, code, name, status, created_at, updated_at FROM batches WHERE code=?`, code)
	var b model.Batch
	var created, updated string
	if err := row.Scan(&b.ID, &b.Code, &b.Name, &b.Status, &created, &updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	b.CreatedAt, _ = time.Parse(time.RFC3339, created)
	b.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &b, nil
}

// List 列出全部批次。
func (s *BatchStore) List() ([]model.Batch, error) {
	rows, err := s.db.Query(`SELECT id, code, name, status, created_at, updated_at FROM batches ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Batch{}
	for rows.Next() {
		var b model.Batch
		var created, updated string
		if err := rows.Scan(&b.ID, &b.Code, &b.Name, &b.Status, &created, &updated); err != nil {
			return nil, err
		}
		b.CreatedAt, _ = time.Parse(time.RFC3339, created)
		b.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		out = append(out, b)
	}
	return out, rows.Err()
}

// UpdateStatus 更新批次状态并刷新 updated_at；返回受影响行数。
func (s *BatchStore) UpdateStatus(id int64, status model.BatchStatus) error {
	res, err := s.db.Exec(
		`UPDATE batches SET status=?, updated_at=? WHERE id=?`,
		string(status), time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// Count 统计批次总数（含按状态分组）。
func (s *BatchStore) Count() (total int, byStatus map[model.BatchStatus]int, err error) {
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM batches`).Scan(&total); err != nil {
		return 0, nil, err
	}
	byStatus = map[model.BatchStatus]int{}
	rows, err := s.db.Query(`SELECT status, COUNT(*) FROM batches GROUP BY status`)
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
		byStatus[model.BatchStatus(st)] = n
	}
	return total, byStatus, rows.Err()
}
