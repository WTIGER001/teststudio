package gitlab

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

func (c *Client) json(ctx context.Context, a sourceprovider.Access, method, path string, body, output any) error {
	var content *strings.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return errors.New("GitLab metadata request is invalid")
		}
		content = strings.NewReader(string(data))
	} else {
		content = strings.NewReader("")
	}
	data, err := c.request(ctx, a, method, path, content, 2<<20)
	if err != nil {
		return err
	}
	if output != nil && json.Unmarshal(data, output) != nil {
		return errors.New("GitLab metadata response is invalid")
	}
	return nil
}
func (c *Client) WriteMetadata(ctx context.Context, a sourceprovider.Access, r sourceprovider.WriteRequest) (sourceprovider.WriteResult, error) {
	var result sourceprovider.WriteResult
	if err := sourceprovider.ValidateWrite(r); err != nil {
		return result, err
	}
	origin, _ := c.origin()
	if r.Base.Repository != a.Repository || r.Base.URL != origin+"/"+a.Repository {
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
	a.RepositoryID = r.Base.RepositoryID
	prefix := "/projects/" + a.RepositoryID
	result.Branch = "teststudio/" + r.Key
	var branches []struct {
		Name   string `json:"name"`
		Commit struct {
			ID string `json:"id"`
		} `json:"commit"`
	}
	if err = c.json(ctx, a, "GET", prefix+"/repository/branches?search="+url.QueryEscape("^"+result.Branch+"$")+"&per_page=100", nil, &branches); err != nil {
		return result, err
	}
	for _, branch := range branches {
		if branch.Name == result.Branch {
			result.Commit = branch.Commit.ID
		}
	}
	if result.Commit != "" {
		for name, want := range r.Files {
			var file struct {
				Content  string `json:"content"`
				Encoding string `json:"encoding"`
			}
			if err = c.json(ctx, a, "GET", prefix+"/repository/files/"+url.PathEscape(name)+"?ref="+url.QueryEscape(result.Commit), nil, &file); err != nil {
				return result, err
			}
			got, err := base64.StdEncoding.DecodeString(file.Content)
			if err != nil || file.Encoding != "base64" || string(got) != want {
				return result, errors.New("review branch differs from pending metadata; review the conflict")
			}
		}
	} else {
		if current.Commit != r.Base.Commit {
			return result, errors.New("repository branch changed; refresh the metadata diff before committing")
		}
		names := make([]string, 0, len(r.Files))
		for name := range r.Files {
			names = append(names, name)
		}
		sort.Strings(names)
		actions := []map[string]string{}
		for _, name := range names {
			action := "create"
			var file struct {
				FilePath string `json:"file_path"`
			}
			data, lookupErr := c.request(ctx, a, "GET", prefix+"/repository/files/"+url.PathEscape(name)+"?ref="+r.Base.Commit, nil, 2<<20)
			if lookupErr == nil {
				if json.Unmarshal(data, &file) != nil || file.FilePath != name {
					return result, errors.New("GitLab metadata file lookup is invalid")
				}
				action = "update"
			} else if !strings.Contains(lookupErr.Error(), "(HTTP 404)") {
				return result, lookupErr
			}
			actions = append(actions, map[string]string{"action": action, "file_path": name, "content": r.Files[name]})
		}
		var commit struct {
			ID string `json:"id"`
		}
		if err = c.json(ctx, a, "POST", prefix+"/repository/commits", map[string]any{"branch": result.Branch, "start_sha": r.Base.Commit, "commit_message": r.Message, "actions": actions}, &commit); err != nil {
			return result, err
		}
		result.Commit = commit.ID
	}
	if !sourceprovider.ValidCommit(result.Commit) {
		return result, errors.New("GitLab returned an invalid metadata commit")
	}
	target := strings.TrimPrefix(r.Base.Reference, "heads/")
	var reviews []struct {
		URL string `json:"web_url"`
	}
	if err = c.json(ctx, a, "GET", prefix+"/merge_requests?scope=all&source_branch="+url.QueryEscape(result.Branch)+"&target_branch="+url.QueryEscape(target), nil, &reviews); err != nil {
		return result, err
	}
	if len(reviews) > 0 {
		result.ReviewURL = reviews[0].URL
	} else {
		var review struct {
			URL string `json:"web_url"`
		}
		if err = c.json(ctx, a, "POST", prefix+"/merge_requests", map[string]any{"source_branch": result.Branch, "target_branch": target, "title": r.Message, "remove_source_branch": true}, &review); err != nil {
			return result, err
		}
		result.ReviewURL = review.URL
	}
	if !strings.HasPrefix(result.ReviewURL, origin+"/"+a.Repository+"/-/merge_requests/") {
		return result, errors.New("GitLab returned an invalid review URL")
	}
	return result, nil
}
