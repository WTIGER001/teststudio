package sources

import (
	"context"
	"errors"
	"path"
	"slices"
	"strings"

	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
	"github.com/wtiger001/teststudio/internal/storage"
)

type RepositoryCandidate struct {
	ModuleRoot string `json:"moduleRoot"`
	Config     string `json:"config"`
}
type RepositoryInspection struct {
	Revision   sourceprovider.Revision `json:"revision"`
	Candidates []RepositoryCandidate   `json:"candidates"`
	Manifests  []string                `json:"manifests"`
}

func (s *Service) InspectRepository(ctx context.Context, user, project, source string) (RepositoryInspection, error) {
	row, err := s.ReadRepository(ctx, user, project, source)
	if err != nil {
		return RepositoryInspection{}, err
	}
	if err = s.checkRepository(ctx, user, project, source, "manage"); err != nil {
		return RepositoryInspection{}, err
	}
	registered, err := storage.Decode[RepositorySource](row)
	if err != nil {
		return RepositoryInspection{}, err
	}
	provider := s.Providers[registered.Configuration.Provider]
	if provider == nil {
		return RepositoryInspection{}, sourceprovider.ErrUnavailable
	}
	access := sourceprovider.Access{Project: project, Repository: registered.Configuration.Repository, RepositoryID: registered.RepositoryID, Credential: registered.Configuration.Credential}
	revision, err := provider.Resolve(ctx, access, registered.Configuration.Reference)
	if err != nil {
		return RepositoryInspection{}, errors.New("repository inspection could not resolve source; check its connection")
	}
	archive, err := provider.Archive(ctx, access, revision)
	if err != nil {
		return RepositoryInspection{}, errors.New("repository inspection could not retrieve the resolved source")
	}
	raw, err := sourcebundle.ReadRoot(archive.Content, archive.Root)
	if err != nil {
		return RepositoryInspection{}, err
	}
	result := RepositoryInspection{Revision: revision, Candidates: []RepositoryCandidate{}, Manifests: []string{}}
	for name, body := range raw.Files {
		base := path.Base(name)
		if strings.HasPrefix(base, "playwright.config.") && slices.Contains([]string{".ts", ".js", ".mts", ".mjs", ".cts", ".cjs"}, path.Ext(base)) {
			result.Candidates = append(result.Candidates, RepositoryCandidate{ModuleRoot: path.Dir(name), Config: base})
		}
		if strings.HasSuffix(name, ".json") && strings.Contains(string(body), `"suiteId"`) && strings.Contains(string(body), `"buildProfile"`) {
			result.Manifests = append(result.Manifests, name)
		}
	}
	slices.SortFunc(result.Candidates, func(a, b RepositoryCandidate) int {
		return strings.Compare(a.ModuleRoot+"/"+a.Config, b.ModuleRoot+"/"+b.Config)
	})
	slices.Sort(result.Manifests)
	return result, nil
}
