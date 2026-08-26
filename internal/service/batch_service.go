package service

import (
	"fmt"

	"task268-bronzeform/internal/model"
)

// CreateBatch 创建铭文批次。
func (s *Service) CreateBatch(code, name string) (*model.Batch, error) {
	if code == "" || name == "" {
		return nil, fmt.Errorf("%w: code and name required", model.ErrBadInput)
	}
	b := &model.Batch{Code: code, Name: name}
	if err := s.Batches.Create(b); err != nil {
		return nil, err
	}
	return b, nil
}

// SubmitBatch 提交批次进入待比较。
func (s *Service) SubmitBatch(batchID int64) (*model.Batch, error) {
	b, err := s.Batches.Get(batchID)
	if err != nil {
		return nil, err
	}
	if b.Status != model.BatchOrganizing {
		return nil, fmt.Errorf("%w: batch must be organizing, got %s", model.ErrInvalidState, b.Status)
	}
	if err := s.Batches.UpdateStatus(batchID, model.BatchPendingCompare); err != nil {
		return nil, err
	}
	b.Status = model.BatchPendingCompare
	return b, nil
}

// SealBatch 封存批次（只读终态）。
func (s *Service) SealBatch(batchID int64) error {
	b, err := s.Batches.Get(batchID)
	if err != nil {
		return err
	}
	if b.Status != model.BatchPublished {
		return fmt.Errorf("%w: batch must be published to seal, got %s", model.ErrInvalidState, b.Status)
	}
	return s.Batches.UpdateStatus(batchID, model.BatchSealed)
}
