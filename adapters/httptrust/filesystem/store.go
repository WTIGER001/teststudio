package filesystem

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
)

type Binding struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Project string `json:"projectId"`
	Origin  string `json:"origin"`
	File    string `json:"caFile"`
}

type entry struct {
	roots  *x509.CertPool
	bundle execution.HTTPTrustBundle
}

type Store struct{ entries map[string]entry }

func Load(filename string) (*Store, error) {
	if filename == "" {
		return New(nil)
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, errors.New("HTTP trust bundle configuration is unavailable")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return nil, errors.New("HTTP trust bundle configuration exceeds its read limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var bindings []Binding
	if decoder.Decode(&bindings) != nil {
		return nil, errors.New("HTTP trust bundle configuration is invalid")
	}
	var extra any
	if !errors.Is(decoder.Decode(&extra), io.EOF) {
		return nil, errors.New("HTTP trust bundle configuration contains trailing data")
	}
	return New(bindings)
}

func New(bindings []Binding) (*Store, error) {
	if len(bindings) > 1000 {
		return nil, errors.New("HTTP trust configuration exceeds 1000 bindings")
	}
	store := &Store{entries: map[string]entry{}}
	identifiers := map[string]bool{}
	for _, binding := range bindings {
		u, err := url.Parse(binding.Origin)
		if err != nil || len(binding.Origin) > 4096 || u.Scheme != "https" || u.RawQuery != "" || u.ForceQuery || u.Path != "" && u.Path != "/" || !domain.ValidID(binding.ID) || !domain.ValidID(binding.Project) || strings.TrimSpace(binding.Name) == "" || len(binding.Name) > 200 || !filepath.IsAbs(binding.File) || len(binding.File) > 4096 {
			return nil, errors.New("HTTP trust binding requires a valid identifier, project, name, exact HTTPS origin, and absolute CA file")
		}
		origin, err := execution.HTTPOrigin(binding.Origin)
		if err != nil {
			return nil, errors.New("HTTP trust binding origin is invalid")
		}
		key, identifier := binding.Project+"/"+origin, binding.Project+"/"+binding.ID
		if _, duplicate := store.entries[key]; duplicate || identifiers[identifier] {
			return nil, errors.New("HTTP trust bindings require unique project origins and identifiers")
		}
		roots, digest, err := loadRoots(binding.File)
		if err != nil {
			return nil, errors.New("HTTP trust CA file must contain only valid PEM certificate authorities within 64 KiB")
		}
		store.entries[key] = entry{roots, execution.HTTPTrustBundle{ID: binding.ID, Name: binding.Name, Digest: digest}}
		identifiers[identifier] = true
	}
	return store, nil
}

func loadRoots(filename string) (*x509.CertPool, string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 65536 {
		return nil, "", errors.New("invalid CA file")
	}
	data, err := io.ReadAll(io.LimitReader(file, 65537))
	if err != nil || len(data) > 65536 {
		return nil, "", errors.New("invalid CA file size")
	}
	roots, err := x509.SystemCertPool()
	if err != nil {
		return nil, "", err
	}
	fingerprints := []string{}
	for len(bytes.TrimSpace(data)) > 0 {
		if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("-----BEGIN CERTIFICATE-----")) {
			return nil, "", errors.New("invalid CA PEM block")
		}
		block, remaining := pem.Decode(data)
		if block == nil || block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			return nil, "", errors.New("invalid CA PEM certificate")
		}
		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil || !certificate.IsCA || !certificate.BasicConstraintsValid || certificate.KeyUsage != 0 && certificate.KeyUsage&x509.KeyUsageCertSign == 0 {
			return nil, "", errors.New("invalid certificate authority")
		}
		roots.AddCert(certificate)
		digest := sha256.Sum256(block.Bytes)
		fingerprints = append(fingerprints, hex.EncodeToString(digest[:]))
		data = remaining
	}
	if len(fingerprints) == 0 {
		return nil, "", errors.New("empty CA bundle")
	}
	slices.Sort(fingerprints)
	fingerprints = slices.Compact(fingerprints)
	digest := sha256.Sum256([]byte(strings.Join(fingerprints, "\n")))
	return roots, hex.EncodeToString(digest[:]), nil
}

func (s *Store) Resolve(ctx context.Context, project, origin string) (*x509.CertPool, *execution.HTTPTrustBundle, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if !domain.ValidID(project) {
		return nil, nil, errors.New("HTTP trust requires an explicit project")
	}
	value, ok := s.entries[project+"/"+origin]
	if !ok {
		return nil, nil, nil
	}
	bundle := value.bundle
	return value.roots.Clone(), &bundle, nil
}
