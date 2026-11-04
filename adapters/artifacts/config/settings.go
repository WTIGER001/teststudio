package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/wtiger001/teststudio/internal/artifacts"
)

type Settings struct {
	Driver    string `json:"driver"`
	Directory string `json:"directory,omitempty"`
	S3        *S3    `json:"s3,omitempty"`
	Azure     *Azure `json:"azure,omitempty"`
}

type S3 struct {
	Region        string `json:"region"`
	Bucket        string `json:"bucket"`
	Prefix        string `json:"prefix,omitempty"`
	ExpectedOwner string `json:"expectedOwner,omitempty"`
	KMSKeyID      string `json:"kmsKeyId,omitempty"`
}

type Azure struct {
	ServiceURL              string `json:"serviceUrl"`
	Container               string `json:"container"`
	Prefix                  string `json:"prefix,omitempty"`
	ManagedIdentityClientID string `json:"managedIdentityClientId,omitempty"`
}

var bucketName = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)
var containerName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)
var ownerID = regexp.MustCompile(`^[0-9]{12}$`)

func Load(filename, directory string) (Settings, error) {
	settings := Settings{Driver: "filesystem", Directory: directory}
	if filename != "" {
		file, err := os.Open(filename)
		if err != nil {
			return Settings{}, errors.New("artifact store configuration cannot be read")
		}
		defer file.Close()
		data, err := io.ReadAll(io.LimitReader(file, (64<<10)+1))
		if err != nil || len(data) > 64<<10 {
			return Settings{}, errors.New("artifact store configuration exceeds its read limit")
		}
		settings = Settings{}
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&settings); err != nil {
			return Settings{}, errors.New("artifact store configuration must be valid JSON with supported fields")
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			return Settings{}, errors.New("artifact store configuration must contain one JSON object")
		}
		if settings.Driver == "filesystem" && settings.Directory != "" && !filepath.IsAbs(settings.Directory) {
			settings.Directory = filepath.Join(filepath.Dir(filename), settings.Directory)
		}
	}
	if err := settings.Validate(); err != nil {
		return Settings{}, err
	}
	if settings.Driver == "filesystem" {
		absolute, err := filepath.Abs(settings.Directory)
		if err != nil {
			return Settings{}, err
		}
		settings.Directory = absolute
	}
	return settings, nil
}

func (s Settings) Validate() error {
	switch s.Driver {
	case "filesystem":
		if s.Directory == "" || s.S3 != nil || s.Azure != nil {
			return errors.New("filesystem artifact storage requires only a directory")
		}
	case "s3":
		if s.S3 == nil || s.Azure != nil || s.Directory != "" {
			return errors.New("S3 artifact storage requires only S3 settings")
		}
		v := s.S3
		if strings.TrimSpace(v.Region) != v.Region || v.Region == "" || !bucketName.MatchString(v.Bucket) || strings.Contains(v.Bucket, "..") || !artifacts.ValidPrefix(v.Prefix) || (v.ExpectedOwner != "" && !ownerID.MatchString(v.ExpectedOwner)) {
			return errors.New("S3 artifact storage requires a region, bucket, canonical prefix, and valid optional owner")
		}
	case "azure":
		if s.Azure == nil || s.S3 != nil || s.Directory != "" {
			return errors.New("Azure artifact storage requires only Azure settings")
		}
		v := s.Azure
		u, err := url.Parse(v.ServiceURL)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || !containerName.MatchString(v.Container) || strings.Contains(v.Container, "--") || !artifacts.ValidPrefix(v.Prefix) {
			return errors.New("Azure artifact storage requires an HTTPS service origin, container, and canonical prefix")
		}
	default:
		return errors.New("artifact driver must be filesystem, s3, or azure")
	}
	return nil
}
