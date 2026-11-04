package httpapi

import (
	"errors"
	"github.com/wtiger001/teststudio/internal/catalog"
	"net/http"
	"strconv"
)

func (s *Server) catalogList(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	limit, err := pageLimit(r, 30)
	if err != nil || limit < 1 || limit > 100 {
		problem(w, errors.New("catalog limit must be between 1 and 100"))
		return
	}
	descendants, uncategorized := true, false
	for _, field := range []struct {
		name  string
		value *bool
	}{{"descendants", &descendants}, {"uncategorized", &uncategorized}} {
		if values.Has(field.name) {
			value, err := strconv.ParseBool(values.Get(field.name))
			if err != nil {
				problem(w, errors.New("catalog category flags require true or false"))
				return
			}
			*field.value = value
		}
	}
	q := catalog.ListQuery{Query: values.Get("query"), Suite: values.Get("suite"), Application: values.Get("application"), Category: values.Get("category"), Descendants: descendants, Uncategorized: uncategorized, Tag: values.Get("tag"), Owner: values.Get("owner"), Type: values.Get("type"), Runner: values.Get("runner"), State: values.Get("state"), Safety: values.Get("safety"), Source: values.Get("source"), Sort: values.Get("sort"), Limit: limit}
	page, err := s.Catalog.ListTests(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), q, values.Get("cursor"))
	if err != nil {
		problem(w, err)
		return
	}
	JSON(w, 200, page)
}
