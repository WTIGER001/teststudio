package oci

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

var ErrConfiguration = errors.New("registry inspection requires an explicit HTTPS registry, repository prefix, and supported platform")
var ErrUnavailable = errors.New("approved registry metadata is unavailable")
var ErrInvalid = errors.New("registry content does not prove a matching immutable supervised runtime")
var ErrDenied = errors.New("registry metadata access was denied")

type Credential struct {
	Username string `json:"-"`
	Password string `json:"-"`
	Token    string `json:"-"`
}

type Options struct {
	Origin              string
	RepositoryPrefix    string
	Architecture        string
	TokenURL            string
	TokenService        string
	Credentials         func(context.Context) (Credential, error)
	Transport           http.RoundTripper
	BlobRedirectOrigins []string
}

type Metadata struct {
	template      *runtimeTemplate
	Image         hostedexecution.Image
	Manifest      string
	Configuration string
	Project       string
	Installation  string
	Preparation   string
	Source        string
}

type Client struct {
	origin, repositoryPrefix, architecture, tokenURL, tokenService string
	credentials                                                    func(context.Context) (Credential, error)
	http                                                           *http.Client
	blobRedirectOrigins                                            map[string]bool
}

type authorization struct{ base, current Credential }

func (c Credential) String() string   { return "private registry credential" }
func (c Credential) GoString() string { return c.String() }

const ociManifest = "application/vnd.oci.image.manifest.v1+json"
const ociIndex = "application/vnd.oci.image.index.v1+json"
const dockerManifest = "application/vnd.docker.distribution.manifest.v2+json"
const dockerIndex = "application/vnd.docker.distribution.manifest.list.v2+json"
const metadataLimit = 2 << 20

type platform struct {
	OS           string   `json:"os"`
	Architecture string   `json:"architecture"`
	Variant      string   `json:"variant,omitempty"`
	Features     []string `json:"os.features,omitempty"`
}

type descriptor struct {
	MediaType string    `json:"mediaType"`
	Digest    string    `json:"digest"`
	Size      int64     `json:"size"`
	URLs      []string  `json:"urls,omitempty"`
	Platform  *platform `json:"platform,omitempty"`
}

type manifest struct {
	Schema       int          `json:"schemaVersion"`
	MediaType    string       `json:"mediaType"`
	ArtifactType string       `json:"artifactType,omitempty"`
	Subject      *descriptor  `json:"subject,omitempty"`
	Config       descriptor   `json:"config"`
	Layers       []descriptor `json:"layers"`
	Manifests    []descriptor `json:"manifests,omitempty"`
}

type configuration struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	Variant      string `json:"variant,omitempty"`
	Config       struct {
		Labels     map[string]string          `json:"Labels"`
		Volumes    map[string]json.RawMessage `json:"Volumes"`
		User       string                     `json:"User"`
		WorkingDir string                     `json:"WorkingDir"`
		Entrypoint []string                   `json:"Entrypoint"`
		OnBuild    []string                   `json:"OnBuild"`
	} `json:"config"`
	RootFS struct {
		Type    string   `json:"type"`
		DiffIDs []string `json:"diff_ids"`
	} `json:"rootfs"`
}
