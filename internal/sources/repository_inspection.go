package sources

import (
	"context"
	"errors"
	"path"
	"slices"
	"strings"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/secrets"
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
	config := registered.Configuration
	return s.inspectRepositoryFiles(ctx, sourceprovider.Access{Project: project, Repository: config.Repository, RepositoryID: registered.RepositoryID, Credential: config.Credential}, config.Provider, config.Reference)
}

type RepositoryInspectionRequest struct {
	Provider   string `json:"provider"`
	Repository string `json:"repository"`
	Reference  string `json:"reference"`
	Credential string `json:"credentialReference,omitempty"`
	Mode       string `json:"accessMode,omitempty"`
}

func (s *Service) PreviewRepository(ctx context.Context, user, project, source string, input RepositoryInspectionRequest) (RepositoryInspection, error) {
	if err := s.checkRepository(ctx, user, project, source, "manage"); err != nil {
		return RepositoryInspection{}, err
	}
	if !domain.ValidID(input.Provider) || input.Mode != "" && input.Mode != "read-only" && input.Mode != "read-write" {
		return RepositoryInspection{}, errors.New("choose an approved provider and metadata ownership mode")
	}
	if input.Mode == "read-write" && (!strings.HasPrefix(input.Reference, "heads/") || input.Credential == "") {
		return RepositoryInspection{}, errors.New("read/write metadata requires a branch and an approved write credential")
	}
	if input.Credential != "" {
		if s.Secrets == nil {
			return RepositoryInspection{}, errors.New("repository credentials are unavailable")
		}
		refs, err := s.Secrets.References(ctx, project, sourceprovider.CredentialPurpose)
		if err != nil {
			return RepositoryInspection{}, err
		}
		if !slices.ContainsFunc(refs, func(ref secrets.Reference) bool { return ref.ID == input.Credential }) {
			return RepositoryInspection{}, errors.New("choose a repository credential approved for this project")
		}
	}
	result, err := s.inspectRepositoryFiles(ctx, sourceprovider.Access{Project: project, Repository: input.Repository, Credential: input.Credential, Write: input.Mode == "read-write"}, input.Provider, input.Reference)
	if err != nil {
		return RepositoryInspection{}, err
	}
	if err = s.checkRepository(ctx, user, project, source, "manage"); err != nil {
		return RepositoryInspection{}, err
	}
	return result, nil
}

func (s *Service) inspectRepositoryFiles(ctx context.Context, access sourceprovider.Access, providerID, reference string) (RepositoryInspection, error) {
	provider := s.Providers[providerID]
	if provider == nil {
		return RepositoryInspection{}, sourceprovider.ErrUnavailable
	}
	if err := provider.Validate(access, reference); err != nil {
		return RepositoryInspection{}, err
	}
	revision, err := provider.Resolve(ctx, access, reference)
	if err != nil {
		return RepositoryInspection{}, errors.New("repository inspection could not resolve source; check its connection, branch, credential, and server trust")
	}
	if !sourceprovider.ValidCommit(revision.Commit) || revision.Reference != reference || revision.RepositoryID == "" || access.RepositoryID != "" && access.RepositoryID != revision.RepositoryID {
		return RepositoryInspection{}, errors.New("repository inspection returned an invalid source identity")
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
