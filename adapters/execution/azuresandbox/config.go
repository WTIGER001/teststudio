package azuresandbox

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
const APIVersion = "2026-02-01-preview"
const TokenScope = "https://dynamicsessions.io/.default"

type Config struct {
	Subscription     string `json:"subscription"`
	ResourceGroup    string `json:"resourceGroup"`
	Region           string `json:"region"`
	Group            string `json:"group"`
	Registry         string `json:"registry"`
	RepositoryPrefix string `json:"repositoryPrefix"`
	ExchangeOrigin   string `json:"exchangeOrigin"`
	VNetConnection   string `json:"vnetConnection"`
	VMMType          string `json:"vmmType"`
	NetworkProfile   string `json:"networkProfile"`
}

var guid = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
var groupName = regexp.MustCompile(`^[A-Za-z0-9_-][A-Za-z0-9_.()-]{0,89}$`)
var resourceName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}[a-z0-9]$`)
var regionName = regexp.MustCompile(`^[a-z][a-z0-9]{1,31}$`)
var registryName = regexp.MustCompile(`^[a-z0-9]{5,50}\.azurecr\.io$`)
var repositoryName = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:/[a-z0-9]+(?:[._-][a-z0-9]+)*)*$`)
var checksum = regexp.MustCompile(`^[a-f0-9]{64}$`)
var ticketToken = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)
var identifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)
var dnsName = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9.-]{0,251}[a-z0-9])?$`)

func hash(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func Validate(c Config) error {
	u, err := url.Parse(c.ExchangeOrigin)
	if err != nil || u.Scheme != "https" || u.Host != u.Hostname() || !dnsName.MatchString(u.Hostname()) || !strings.Contains(u.Hostname(), ".") || strings.Contains(u.Hostname(), "..") || u.User != nil || u.Path != "" || u.RawPath != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || u.String() != c.ExchangeOrigin {
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
	if !guid.MatchString(c.Subscription) || !groupName.MatchString(c.ResourceGroup) || strings.HasSuffix(c.ResourceGroup, ".") || strings.Contains(c.ResourceGroup, "..") || !regionName.MatchString(c.Region) || !resourceName.MatchString(c.Group) || !resourceName.MatchString(c.VNetConnection) || !identifier.MatchString(c.VMMType) || !registryName.MatchString(c.Registry) || !repositoryName.MatchString(c.RepositoryPrefix) || len(c.RepositoryPrefix) > 128 || !checksum.MatchString(c.NetworkProfile) {
		return hostedexecution.ErrConfiguration
	}
	return nil
}

func (c Config) endpoint() string {
	return "https://management." + c.Region + ".azuredevcompute.io"
}

func (c Config) path() string {
	return "/subscriptions/" + c.Subscription + "/resourceGroups/" + url.PathEscape(c.ResourceGroup) + "/sandboxGroups/" + c.Group
}

func (c Config) validImage(image string) bool {
	name, digest, found := strings.Cut(image, "@sha256:")
	return found && len(image) <= 1024 && checksum.MatchString(digest) && strings.HasPrefix(name, c.Registry+"/"+c.RepositoryPrefix+"/") && repositoryName.MatchString(strings.TrimPrefix(name, c.Registry+"/"))
}

func (c Config) validPullIdentity(identity string) bool {
	parts := strings.Split(identity, "/")
	return len(parts) == 9 && parts[0] == "" && parts[1] == "subscriptions" && parts[2] == c.Subscription && parts[3] == "resourceGroups" && groupName.MatchString(parts[4]) && !strings.HasSuffix(parts[4], ".") && !strings.Contains(parts[4], "..") && parts[5] == "providers" && parts[6] == "Microsoft.ManagedIdentity" && parts[7] == "userAssignedIdentities" && resourceName.MatchString(parts[8])
}

func ExecutionProfile(c Config) (string, error) {
	if err := Validate(c); err != nil {
		return "", err
	}
	return "aca-sandbox-v1-" + hash([]any{c, APIVersion, TokenScope, hostedexecution.SandboxSupervisor, "linux-amd64-cpu2-memory4Gi-disk40Gi-uid1000"})[:20], nil
}
