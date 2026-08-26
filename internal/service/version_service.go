package service

import (
	"fmt"

	"task268-bronzeform/internal/model"
	"task268-bronzeform/internal/version"
)

// CreateVersion 从已裁决关系创建版本草稿。
func (s *Service) CreateVersion(batchID int64, name string) (*model.EvolutionVersion, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: version name required", model.ErrBadInput)
	}
	rels, err := s.Rels.ListByBatch(batchID)
	if err != nil {
		return nil, err
	}
	sc, err := version.CollectDecided(rels, name)
	if err != nil {
		return nil, err
	}
	v := &model.EvolutionVersion{BatchID: batchID, Name: name, ContentHash: sc.ContentHash}
	if err := s.Versions.Create(v); err != nil {
		return nil, err
	}
	if err := s.Versions.SetRelations(v.ID, sc.RelationIDs()); err != nil {
		return nil, err
	}
	_ = sc.BatchID
	v.RelationIDs = sc.RelationIDs()
	return v, nil
}

// ShareVersion 共享版本。
func (s *Service) ShareVersion(verID int64) error {
	v, err := s.Versions.Get(verID)
	if err != nil {
		return err
	}
	if v.Status != model.VersionDraft {
		return fmt.Errorf("%w: only draft can be shared, got %s", model.ErrInvalidState, v.Status)
	}
	return s.Versions.UpdateStatus(verID, model.VersionShared)
}

// FreezeVersion 冻结发布版本（不可变）。冻结时校验内容哈希未被篡改。
func (s *Service) FreezeVersion(verID int64) error {
	v, err := s.Versions.Get(verID)
	if err != nil {
		return err
	}
	if v.Status != model.VersionDraft && v.Status != model.VersionShared {
		return fmt.Errorf("%w: only draft/shared can be frozen, got %s", model.ErrInvalidState, v.Status)
	}
	rels, err := s.Rels.ListByBatch(v.BatchID)
	if err != nil {
		return err
	}
	sc, err := version.CollectDecided(rels, v.Name)
	if err != nil {
		return err
	}
	if sc.ContentHash != v.ContentHash {
		return fmt.Errorf("%w: version content mutated before freeze", model.ErrForbidden)
	}
	if err := s.Versions.UpdateStatus(verID, model.VersionFrozen); err != nil {
		return err
	}
	// 批次发布。
	if err := s.Batches.UpdateStatus(v.BatchID, model.BatchPublished); err != nil {
		return err
	}
	return nil
}

// SupersedeVersion 替代版本：旧版本转为 superseded，冻结版本不可再修改。
func (s *Service) SupersedeVersion(verID int64, newVerID int64) error {
	oldV, err := s.Versions.Get(verID)
	if err != nil {
		return err
	}
	newV, err := s.Versions.Get(newVerID)
	if err != nil {
		return err
	}
	if oldV.BatchID != newV.BatchID {
		return fmt.Errorf("%w: versions belong to different batches", model.ErrBadInput)
	}
	if oldV.Status != model.VersionFrozen {
		return fmt.Errorf("%w: only frozen version can be superseded, got %s", model.ErrInvalidState, oldV.Status)
	}
	if newV.Status != model.VersionFrozen {
		return fmt.Errorf("%w: new version must be frozen before superseding, got %s", model.ErrInvalidState, newV.Status)
	}
	return s.Versions.UpdateStatus(verID, model.VersionSuperseded)
}
