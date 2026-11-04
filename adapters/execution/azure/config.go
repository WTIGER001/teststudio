package azureworker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/netip"
	"net/url"
	"regexp"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

const Location = "azure-container-apps"

type Config struct {
	Subscription          string `json:"subscription"`
	ResourceGroup         string `json:"resourceGroup"`
	Region                string `json:"region"`
	WorkerEnvironment     string `json:"workerEnvironment"`
	ControllerEnvironment string `json:"controllerEnvironment"`
	PullIdentity          string `json:"pullIdentity"`
	Registry              string `json:"registry"`
	RepositoryPrefix      string `json:"repositoryPrefix"`
	NamePrefix            string `json:"namePrefix"`
	NetworkProfile        string `json:"networkProfile"`
	ExchangeOrigin        string `json:"exchangeOrigin"`
}

var guid = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
var groupName = regexp.MustCompile(`^[A-Za-z0-9_-][A-Za-z0-9_.()-]{0,89}$`)
var resourceName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}[a-z0-9]$`)
var namePrefix = regexp.MustCompile(`^[a-z][a-z0-9-]{0,4}[a-z0-9]$`)
var region = regexp.MustCompile(`^[a-z][a-z0-9]{1,31}$`)
var registry = regexp.MustCompile(`^[a-z0-9]{5,50}\.azurecr\.io$`)
var repository = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:/[a-z0-9]+(?:[._-][a-z0-9]+)*)*$`)
var checksum = regexp.MustCompile(`^[a-f0-9]{64}$`)
var token = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)
var hostname = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9.-]{0,251}[a-z0-9])?$`)

func fingerprint(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (c Config) resourceID(provider, kind, name string) string {
	return "/subscriptions/" + c.Subscription + "/resourceGroups/" + c.ResourceGroup + "/providers/" + provider + "/" + kind + "/" + name
}

func (c Config) ownsResource(value, provider, kind string) bool {
	parts := strings.Split(value, "/")
	return len(parts) == 9 && parts[0] == "" && parts[1] == "subscriptions" && parts[2] == c.Subscription && parts[3] == "resourceGroups" && groupName.MatchString(parts[4]) && !strings.HasSuffix(parts[4], ".") && parts[5] == "providers" && parts[6] == provider && parts[7] == kind && resourceName.MatchString(parts[8])
}

func Validate(c Config) error {
	u, err := url.Parse(c.ExchangeOrigin)
	if err != nil || u.Scheme != "https" || u.Host != u.Hostname() || !hostname.MatchString(u.Hostname()) || !strings.Contains(u.Hostname(), ".") || strings.Contains(u.Hostname(), "..") || u.User != nil || u.Path != "" || u.RawPath != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || u.String() != c.ExchangeOrigin {
		return hostedexecution.ErrConfiguration
	}
	for _, label := range strings.Split(u.Hostname(), ".") {
		if len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return hostedexecution.ErrConfiguration
		}
	}
	if _, err := netip.ParseAddr(u.Hostname()); err == nil {
		return hostedexecution.ErrConfiguration
	}
	if !guid.MatchString(c.Subscription) || !groupName.MatchString(c.ResourceGroup) || strings.HasSuffix(c.ResourceGroup, ".") || !region.MatchString(c.Region) || !c.ownsResource(c.WorkerEnvironment, "Microsoft.App", "managedEnvironments") || !c.ownsResource(c.ControllerEnvironment, "Microsoft.App", "managedEnvironments") || strings.EqualFold(c.WorkerEnvironment, c.ControllerEnvironment) || !c.ownsResource(c.PullIdentity, "Microsoft.ManagedIdentity", "userAssignedIdentities") || !registry.MatchString(c.Registry) || !repository.MatchString(c.RepositoryPrefix) || len(c.RepositoryPrefix) > 128 || !namePrefix.MatchString(c.NamePrefix) || strings.Contains(c.NamePrefix, "--") || !checksum.MatchString(c.NetworkProfile) {
		return hostedexecution.ErrConfiguration
	}
	return nil
}

func ExecutionProfile(c Config) (string, error) {
	if err := Validate(c); err != nil {
		return "", err
	}
	return "aca-job-v1-" + fingerprint([]any{c, "linux-amd64-cpu2-memory4Gi-container-storage-v1"})[:20], nil
}
