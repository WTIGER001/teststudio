package httpapi

import (
	"net/http"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Server) catalogTransferRoutes(m *http.ServeMux) {
	base := "/api/v1/projects/{project}/suites/{suite}"
	m.HandleFunc("GET "+base+"/catalog-exports", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		document, err := s.Catalog.Export(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, document)
	}))
	m.HandleFunc("POST "+base+"/catalog-imports", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var document catalog.Document
		if err := body(r, &document); err != nil {
			problem(w, err)
			return
		}
		p, suite := r.PathValue("project"), r.PathValue("suite")
		plan, err := s.Catalog.PreviewImport(r.Context(), actor(r), p, suite, document)
		if err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, catalog.Bucket("imports", p, suite), plan.ID)
	}))
	m.HandleFunc("POST "+base+"/tests/bulk-preview", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input catalog.BulkRequest
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		p, suite := r.PathValue("project"), r.PathValue("suite")
		plan, err := s.Catalog.PreviewBulk(r.Context(), actor(r), p, suite, input)
		if err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, catalog.Bucket("imports", p, suite), plan.ID)
	}))
	m.HandleFunc("GET "+base+"/catalog-imports/{id}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		p, suite := r.PathValue("project"), r.PathValue("suite")
		if err := s.Auth.Check(r.Context(), actor(r), p, "suite/"+suite, "catalog.read"); err != nil {
			problem(w, err)
			return
		}
		row, err := s.Store.Get(r.Context(), catalog.Bucket("imports", p, suite), r.PathValue("id"))
		if err != nil {
			problem(w, err)
			return
		}
		plan, err := storage.Decode[catalog.ImportPlan](row)
		if err != nil {
			problem(w, err)
			return
		}
		if plan.Actor != actor(r) {
			problem(w, authorization.ErrForbidden)
			return
		}
		for _, change := range plan.Changes {
			if err := s.Auth.Check(r.Context(), actor(r), p, "test/"+suite+"/"+change.Test.ID, "catalog.read"); err != nil {
				problem(w, err)
				return
			}
		}
		JSON(w, 200, row)
	}))
	m.HandleFunc("POST "+base+"/catalog-imports/{id}/apply", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		p, suite := r.PathValue("project"), r.PathValue("suite")
		var plan catalog.ImportPlan
		if r.URL.Query().Get("save") == "true" {
			plan, err = s.Catalog.SaveImport(r.Context(), actor(r), p, suite, r.PathValue("id"), rev)
		} else {
			plan, err = s.Catalog.ApplyImport(r.Context(), actor(r), p, suite, r.PathValue("id"), rev)
		}
		if err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, catalog.Bucket("imports", p, suite), plan.ID)
	}))
	m.HandleFunc("POST "+base+"/tests/{test}/actions/duplicate", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			ID string `json:"id"`
		}
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		p, suite := r.PathValue("project"), r.PathValue("suite")
		if err := s.Catalog.Duplicate(r.Context(), actor(r), p, suite, r.PathValue("test"), input.ID, rev); err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, catalog.Bucket("tests", p, suite), input.ID)
	}))
}
