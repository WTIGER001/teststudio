package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strings"

	"github.com/wtiger001/teststudio/internal/sourceprovider"
)

func (s *session) json(ctx context.Context, method, endpoint string, body, output any) error {
	response, err := s.request(ctx, method, s.api+endpoint, body, s.token, "application/vnd.github+json")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return statusError("metadata write", response.StatusCode)
	}
	data, err := readBounded(response.Body, 2<<20)
	if err != nil {
		return errors.New("GitHub metadata response exceeds its limit")
	}
	if output != nil && json.Unmarshal(data, output) != nil {
		return errors.New("GitHub metadata response is invalid")
	}
	return nil
}
func (c *Client) WriteMetadata(ctx context.Context, a sourceprovider.Access, r sourceprovider.WriteRequest) (sourceprovider.WriteResult, error) {
	var result sourceprovider.WriteResult
	if err := sourceprovider.ValidateWrite(r); err != nil {
		return result, err
	}
	if !strings.EqualFold(r.Base.Repository, a.Repository) || r.Base.URL != "https://github.com/"+r.Base.Repository {
		return result, errors.New("metadata base belongs to another repository")
	}
	current, err := c.Resolve(ctx, a, r.Base.Reference)
	if err != nil {
		return result, err
	}
	if current.RepositoryID != r.Base.RepositoryID {
		return result, errors.New("repository identity changed")
	}
	a.Write = true
	s, err := c.session(ctx, a)
	if err != nil {
		return result, err
	}
	defer s.close()
	if _, err = s.repository(ctx, sourceprovider.Access{Project: a.Project, Repository: a.Repository, RepositoryID: r.Base.RepositoryID, Credential: a.Credential}); err != nil {
		return result, err
	}
	result.Branch = "teststudio/" + r.Key
	prefix := "/repos/" + a.Repository
	var existing struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	response, err := s.request(ctx, "GET", s.api+prefix+"/git/ref/heads/"+result.Branch, nil, s.token, "application/vnd.github+json")
	if err != nil {
		return result, err
	}
	status := response.StatusCode
	data, readErr := readBounded(response.Body, 64<<10)
	response.Body.Close()
	if readErr != nil {
		return result, readErr
	}
	if status == 200 {
		if json.Unmarshal(data, &existing) != nil || !sourceprovider.ValidCommit(existing.Object.SHA) {
			return result, errors.New("GitHub review branch is invalid")
		}
		result.Commit = existing.Object.SHA
		for name, want := range r.Files {
			var file struct {
				Content  string `json:"content"`
				Encoding string `json:"encoding"`
			}
			if err = s.json(ctx, "GET", prefix+"/contents/"+url.PathEscape(name)+"?ref="+url.QueryEscape(result.Commit), nil, &file); err != nil {
				return result, err
			}
			got, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(file.Content, "\n", ""))
			if err != nil || file.Encoding != "base64" || string(got) != want {
				return result, errors.New("review branch differs from the pending metadata; review the conflict")
			}
		}
	} else if status == 404 {
		if current.Commit != r.Base.Commit {
			return result, errors.New("repository branch changed; refresh the metadata diff before committing")
		}
		var base struct {
			Tree struct {
				SHA string `json:"sha"`
			} `json:"tree"`
		}
		if err = s.json(ctx, "GET", prefix+"/git/commits/"+r.Base.Commit, nil, &base); err != nil {
			return result, err
		}
		names := make([]string, 0, len(r.Files))
		for name := range r.Files {
			names = append(names, name)
		}
		sort.Strings(names)
		tree := []map[string]string{}
		for _, name := range names {
			tree = append(tree, map[string]string{"path": name, "mode": "100644", "type": "blob", "content": r.Files[name]})
		}
		var created struct {
			SHA string `json:"sha"`
		}
		if err = s.json(ctx, "POST", prefix+"/git/trees", map[string]any{"base_tree": base.Tree.SHA, "tree": tree}, &created); err != nil {
			return result, err
		}
		if err = s.json(ctx, "POST", prefix+"/git/commits", map[string]any{"message": r.Message, "tree": created.SHA, "parents": []string{r.Base.Commit}}, &created); err != nil {
			return result, err
		}
		result.Commit = created.SHA
		if !sourceprovider.ValidCommit(result.Commit) {
			return result, errors.New("GitHub returned an invalid metadata commit")
		}
		if err = s.json(ctx, "POST", prefix+"/git/refs", map[string]string{"ref": "refs/heads/" + result.Branch, "sha": result.Commit}, nil); err != nil {
			return result, err
		}
	} else {
		return result, statusError("review branch lookup", status)
	}
	owner, _, _ := strings.Cut(a.Repository, "/")
	target := strings.TrimPrefix(r.Base.Reference, "heads/")
	var pulls []struct {
		URL string `json:"html_url"`
	}
	if err = s.json(ctx, "GET", prefix+"/pulls?state=all&head="+url.QueryEscape(owner+":"+result.Branch)+"&base="+url.QueryEscape(target), nil, &pulls); err != nil {
		return result, err
	}
	if len(pulls) > 0 {
		result.ReviewURL = pulls[0].URL
	} else {
		var pull struct {
			URL string `json:"html_url"`
		}
		if err = s.json(ctx, "POST", prefix+"/pulls", map[string]string{"title": r.Message, "head": result.Branch, "base": target, "body": "TestStudio metadata change. Review the files before merging."}, &pull); err != nil {
			return result, err
		}
		result.ReviewURL = pull.URL
	}
	if !strings.HasPrefix(result.ReviewURL, "https://github.com/"+a.Repository+"/pull/") {
		return result, errors.New("GitHub returned an invalid review URL")
	}
	return result, nil
}
