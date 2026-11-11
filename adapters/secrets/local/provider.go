package local

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/networkguard"
	"github.com/wtiger001/teststudio/internal/secrets"
)

type Binding struct {
	Origins     []string           `json:"allowedOrigins,omitempty"`
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Project     string             `json:"projectId"`
	Purpose     string             `json:"purpose"`
	Environment string             `json:"environmentVariable,omitempty"`
	File        string             `json:"file,omitempty"`
	Network     networkguard.Scope `json:"network"`
}
type Provider struct {
	bindings map[string]Binding
	lookup   func(string) (string, bool)
}

var variable = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,127}$`)

func New(bindings []Binding) (*Provider, error) {
	if len(bindings) > 1000 {
		return nil, errors.New("secret binding catalog exceeds 1000 entries")
	}
	p := &Provider{bindings: map[string]Binding{}, lookup: os.LookupEnv}
	for _, b := range bindings {
		if b.Network == "" {
			b.Network = networkguard.Public
		}
		if b.Name == "" {
			b.Name = b.ID
		}
		if b.Purpose == secrets.APICredential {
			if len(b.Origins) < 1 || len(b.Origins) > 16 || b.Network == networkguard.Loopback {
				return nil, errors.New("API credentials require 1 through 16 exact origins and a public or private network scope")
			}
			seen := map[string]bool{}
			for _, origin := range b.Origins {
				canonical, err := browserpolicy.Origin(origin)
				if err != nil || canonical != origin || seen[origin] || strings.Contains(origin, "*") {
					return nil, errors.New("API credential origins must be unique and canonical")
				}
				seen[origin] = true
			}
		} else if len(b.Origins) > 0 {
			return nil, errors.New("origin restrictions apply only to API credentials")
		}
		if !domain.ValidID(b.Project) || !domain.ValidID(b.ID) || len(b.Name) > 256 || !domain.ValidID(b.Purpose) || (b.Environment == "") == (b.File == "") || (b.Environment != "" && !variable.MatchString(b.Environment)) || (b.File != "" && (!filepath.IsAbs(b.File) || len(b.File) > 4096)) || (b.Network != networkguard.Public && b.Network != networkguard.Private && b.Network != networkguard.Loopback) {
			return nil, errors.New("secret binding requires valid scope, purpose, source, and network settings")
		}
		key := b.Project + "/" + b.ID
		if _, exists := p.bindings[key]; exists {
			return nil, errors.New("secret binding identifiers must be unique within a project")
		}
		p.bindings[key] = b
	}
	return p, nil
}
func Load(path string) (*Provider, error) {
	if path == "" {
		return New(nil)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.New("secret binding configuration is unavailable")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return nil, errors.New("secret binding configuration exceeds its read limit")
	}
	return decodeBindings(data)
}

func decodeBindings(data []byte) (*Provider, error) {
	var bindings []Binding
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&bindings) != nil {
		return nil, errors.New("secret binding configuration is invalid")
	}
	var extra any
	if !errors.Is(decoder.Decode(&extra), io.EOF) {
		return nil, errors.New("secret binding configuration contains trailing data")
	}
	return New(bindings)
}
func LoadConfiguration(path, environment string) (*Provider, error) {
	if path != "" && environment != "" {
		return nil, errors.New("choose a secret binding file or environment configuration")
	}
	if environment == "" {
		return Load(path)
	}
	if !variable.MatchString(environment) {
		return nil, errors.New("secret binding environment reference is invalid")
	}
	value, ok := os.LookupEnv(environment)
	if !ok || len(value) == 0 || len(value) > 1<<20 {
		return nil, errors.New("secret binding environment configuration is unavailable or exceeds its limit")
	}
	return decodeBindings([]byte(value))
}

func (p *Provider) References(ctx context.Context, project, purpose string) ([]secrets.Reference, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	refs := []secrets.Reference{}
	for _, b := range p.bindings {
		if b.Project == project && b.Purpose == purpose {
			refs = append(refs, secrets.Reference{ID: b.ID, Name: b.Name, Origins: slices.Clone(b.Origins)})
		}
	}
	slices.SortFunc(refs, func(a, b secrets.Reference) int { return strings.Compare(a.ID, b.ID) })
	return refs, nil
}
func (p *Provider) Resolve(ctx context.Context, project, id, purpose string) (secrets.Value, error) {
	if err := ctx.Err(); err != nil {
		return secrets.Value{}, secrets.ErrUnavailable
	}
	b, ok := p.bindings[project+"/"+id]
	if !ok || b.Purpose != purpose {
		return secrets.Value{}, secrets.ErrUnavailable
	}
	var data []byte
	if b.Environment != "" {
		value, ok := p.lookup(b.Environment)
		if !ok {
			return secrets.Value{}, secrets.ErrUnavailable
		}
		data = []byte(value)
	} else {
		info, err := os.Stat(b.File)
		if err != nil || !info.Mode().IsRegular() || info.Size() > 65536 {
			return secrets.Value{}, secrets.ErrUnavailable
		}
		f, err := os.Open(b.File)
		if err != nil {
			return secrets.Value{}, secrets.ErrUnavailable
		}
		defer f.Close()
		data, err = io.ReadAll(io.LimitReader(f, 65537))
		if err != nil {
			return secrets.Value{}, secrets.ErrUnavailable
		}
	}
	if len(data) == 0 || len(data) > 65536 {
		return secrets.Value{}, secrets.ErrUnavailable
	}
	return secrets.Value{Bytes: data, Network: b.Network, Origins: slices.Clone(b.Origins)}, nil
}
