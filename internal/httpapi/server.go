// Package httpapi 提供 HTTP 接口层：路由注册、JSON 编解码与错误映射。
package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"task268-bronzeform/internal/model"
	"task268-bronzeform/internal/service"
)

// Server HTTP 服务。
type Server struct {
	svc *service.Service
	mux *http.ServeMux
}

// New 创建 HTTP 服务并注册全部路由。
func New(svc *service.Service) *Server {
	s := &Server{svc: svc, mux: http.NewServeMux()}
	s.routes()
	return s
}

// Handler 返回 http.Handler（含日志/恢复中间件）。
func (s *Server) Handler() http.Handler { return WithMiddleware(s.mux) }

// routes 注册 /api 路由（统一前缀）。
func (s *Server) routes() {
	// 批次
	s.mux.HandleFunc("POST /api/batches", s.handleCreateBatch)
	s.mux.HandleFunc("GET /api/batches", s.handleListBatches)
	s.mux.HandleFunc("GET /api/batches/{id}", s.handleGetBatch)
	s.mux.HandleFunc("POST /api/batches/{id}/submit", s.handleSubmitBatch)
	s.mux.HandleFunc("POST /api/batches/{id}/generate", s.handleGenerateCandidates)
	s.mux.HandleFunc("POST /api/batches/{id}/seal", s.handleSealBatch)
	// 字形
	s.mux.HandleFunc("POST /api/glyphs", s.handleImportGlyph)
	s.mux.HandleFunc("GET /api/glyphs", s.handleListGlyphs)
	s.mux.HandleFunc("GET /api/glyphs/{id}", s.handleGetGlyph)
	s.mux.HandleFunc("POST /api/glyphs/{id}/split", s.handleSplitGlyph)
	s.mux.HandleFunc("POST /api/glyphs/{id}/defective", s.handleMarkDefective)
	s.mux.HandleFunc("POST /api/glyphs/{id}/exclude", s.handleExcludeGlyph)
	// 关系
	s.mux.HandleFunc("POST /api/relations", s.handleCreateManualRelation)
	s.mux.HandleFunc("POST /api/relations/{id}/confirm", s.handleConfirmRelation)
	s.mux.HandleFunc("POST /api/relations/{id}/reject", s.handleRejectRelation)
	s.mux.HandleFunc("POST /api/relations/{id}/borrow", s.handleMarkBorrowed)
	s.mux.HandleFunc("POST /api/relations/{id}/rebuttals", s.handleAddRebuttal)
	s.mux.HandleFunc("GET /api/relations", s.handleListRelations)
	// 版本
	s.mux.HandleFunc("POST /api/versions", s.handleCreateVersion)
	s.mux.HandleFunc("GET /api/versions", s.handleListVersions)
	s.mux.HandleFunc("POST /api/versions/{id}/share", s.handleShareVersion)
	s.mux.HandleFunc("POST /api/versions/{id}/freeze", s.handleFreezeVersion)
	s.mux.HandleFunc("POST /api/versions/{id}/supersede", s.handleSupersedeVersion)
	// 统计与健康
	s.mux.HandleFunc("GET /api/stats", s.handleStats)
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
}

func (s *Server) handleCreateBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	b, err := s.svc.CreateBatch(req.Code, req.Name)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (s *Server) handleListBatches(w http.ResponseWriter, r *http.Request) {
	bs, err := s.svc.Batches.List()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bs)
}

func (s *Server) handleGetBatch(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	b, err := s.svc.Batches.Get(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) handleSubmitBatch(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	b, err := s.svc.SubmitBatch(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) handleGenerateCandidates(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	n, err := s.svc.GenerateCandidates(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"batch_id": id, "created": n})
}

func (s *Server) handleSealBatch(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.svc.SealBatch(id); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"batch_id": id, "status": "sealed"})
}

func (s *Server) handleImportGlyph(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BatchID  int64             `json:"batch_id"`
		Code     string            `json:"code"`
		Graph    string            `json:"graph"`
		EraBegin int               `json:"era_begin"`
		EraEnd   int               `json:"era_end"`
		Source   string            `json:"source"`
		Comps    []model.Component `json:"components"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	g, err := s.svc.ImportGlyph(req.BatchID, req.Code, req.Graph, req.EraBegin, req.EraEnd, req.Source, req.Comps)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

func (s *Server) handleListGlyphs(w http.ResponseWriter, r *http.Request) {
	batchID, err := strconv.ParseInt(r.URL.Query().Get("batch_id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrBadInput)
		return
	}
	gs, err := s.svc.Glyphs.ListByBatch(batchID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, gs)
}

func (s *Server) handleGetGlyph(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	g, err := s.svc.Glyphs.Get(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (s *Server) handleSplitGlyph(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	var req struct {
		Graph string `json:"graph"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	g, err := s.svc.SplitGlyph(id, req.Graph)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (s *Server) handleMarkDefective(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.svc.MarkGlyphDefective(id); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"glyph_id": id, "status": "defective"})
}

func (s *Server) handleExcludeGlyph(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.svc.ExcludeGlyph(id); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"glyph_id": id, "status": "excluded"})
}

func (s *Server) handleCreateManualRelation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BatchID      int64 `json:"batch_id"`
		SourceGlyph  int64 `json:"source_glyph"`
		TargetGlyph  int64 `json:"target_glyph"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	rel, err := s.svc.CreateManualRelation(req.BatchID, req.SourceGlyph, req.TargetGlyph)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rel)
}

func (s *Server) handleConfirmRelation(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.svc.ConfirmRelation(id); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"relation_id": id, "status": "confirmed"})
}

func (s *Server) handleRejectRelation(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.svc.RejectRelation(id); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"relation_id": id, "status": "rejected"})
}

func (s *Server) handleMarkBorrowed(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.svc.MarkBorrowed(id); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"relation_id": id, "status": "borrowed"})
}

func (s *Server) handleAddRebuttal(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	var req struct {
		Kind string `json:"kind"`
		Note string `json:"note"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	rb, err := s.svc.AddRebuttal(id, req.Kind, req.Note)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rb)
}

func (s *Server) handleListRelations(w http.ResponseWriter, r *http.Request) {
	batchID, err := strconv.ParseInt(r.URL.Query().Get("batch_id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrBadInput)
		return
	}
	rels, err := s.svc.Rels.ListByBatch(batchID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rels)
}

func (s *Server) handleCreateVersion(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BatchID int64  `json:"batch_id"`
		Name    string `json:"name"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	v, err := s.svc.CreateVersion(req.BatchID, req.Name)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (s *Server) handleListVersions(w http.ResponseWriter, r *http.Request) {
	batchID, err := strconv.ParseInt(r.URL.Query().Get("batch_id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrBadInput)
		return
	}
	vs, err := s.svc.Versions.ListByBatch(batchID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, vs)
}

func (s *Server) handleShareVersion(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.svc.ShareVersion(id); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"version_id": id, "status": "shared"})
}

func (s *Server) handleFreezeVersion(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.svc.FreezeVersion(id); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"version_id": id, "status": "frozen"})
}

func (s *Server) handleSupersedeVersion(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	var req struct {
		NewVersionID int64 `json:"new_version_id"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	if err := s.svc.SupersedeVersion(id, req.NewVersionID); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"version_id": id, "status": "superseded"})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	totalBatches, batchByStatus, err := s.svc.Batches.Count()
	if err != nil {
		writeErr(w, err)
		return
	}
	allGlyphs, err := s.svc.Glyphs.ListAll()
	if err != nil {
		writeErr(w, err)
		return
	}
	glyphByStatus := map[model.GlyphStatus]int{}
	for _, g := range allGlyphs {
		glyphByStatus[g.Status]++
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"batches":          totalBatches,
		"batch_by_status":  batchByStatus,
		"glyphs":           len(allGlyphs),
		"glyph_by_status":  glyphByStatus,
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func pathID(r *http.Request) (int64, error) {
	v := r.PathValue("id")
	if v == "" {
		return 0, model.ErrBadInput
	}
	return strconv.ParseInt(v, 10, 64)
}

func decode(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return model.ErrBadInput
	}
	return nil
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, model.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, model.ErrConflict), errors.Is(err, model.ErrFingerprintExists):
		status = http.StatusConflict
	case errors.Is(err, model.ErrBadInput):
		status = http.StatusBadRequest
	case errors.Is(err, model.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, model.ErrInvalidState), errors.Is(err, model.ErrSelfReference),
		errors.Is(err, model.ErrEraInverted), errors.Is(err, model.ErrMissingSource),
		errors.Is(err, model.ErrComponentCycle), errors.Is(err, model.ErrVersionNotEmpty),
		errors.Is(err, model.ErrNoActiveRelation):
		status = http.StatusUnprocessableEntity
	}
	writeJSON(w, status, map[string]any{"error": err.Error()})
}
