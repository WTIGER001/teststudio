package isolatedcontainer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/wtiger001/teststudio/adapters/recording/embeddedplaywright"
	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/recordingworker"
)

const SeccompHash = "c4fef485853f4abe05f74b8a5bb8ded0f0df029d463886c1e3b3175976fefb58"

var immutable = regexp.MustCompile(`^(sha256:[a-f0-9]{64}|[A-Za-z0-9][A-Za-z0-9._:/-]*@sha256:[a-f0-9]{64})$`)
var networkDigest = regexp.MustCompile(`^[a-f0-9]{64}$`)

type Runtime struct {
	Docker       string
	Image        string
	GatewayImage string
	Seccomp      string
	Directory    string
}

func (r *Runtime) binary() string {
	if r.Docker != "" {
		return r.Docker
	}
	return "docker"
}

func fingerprint(b recordingworker.Binding, ref recordingworker.Reference) string {
	values := []string{b.Fingerprint(), ref.RuntimeImage, ref.GatewayImage, ref.PolicyHash}
	if ref.NetworkHash != "" {
		values = append(values, ref.NetworkHash)
	}
	data, _ := json.Marshal(values)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (r *Runtime) Plan(b recordingworker.Binding, target browserpolicy.Target) (recordingworker.Reference, error) {
	if err := browserpolicy.ValidateTarget(target); err != nil {
		return recordingworker.Reference{}, err
	}
	if !immutable.MatchString(r.Image) || !immutable.MatchString(r.GatewayImage) {
		return recordingworker.Reference{}, errors.New("isolated recording requires immutable browser and gateway images")
	}
	profile, err := os.ReadFile(r.Seccomp)
	if err != nil {
		return recordingworker.Reference{}, errors.New("recording requires the pinned Chromium sandbox seccomp profile")
	}
	sum := sha256.Sum256(profile)
	if hex.EncodeToString(sum[:]) != SeccompHash {
		return recordingworker.Reference{}, errors.New("recording sandbox profile does not match the pinned policy")
	}
	ref := recordingworker.Reference{Provider: "docker-recording/v1", RuntimeImage: r.Image, GatewayImage: r.GatewayImage, PolicyHash: SeccompHash}
	network, _ := json.Marshal(target)
	networkHash := sha256.Sum256(network)
	ref.NetworkHash = hex.EncodeToString(networkHash[:])
	ref.Fingerprint = fingerprint(b, ref)
	prefix := "ts-recording-" + ref.Fingerprint[:32]
	ref.Container, ref.Gateway, ref.Volume = prefix+"-browser", prefix+"-gateway", prefix+"-output"
	return ref, nil
}

func validReference(b recordingworker.Binding, ref recordingworker.Reference) bool {
	if ref.Provider != "docker-recording/v1" || ref.PolicyHash != SeccompHash || !immutable.MatchString(ref.RuntimeImage) || !immutable.MatchString(ref.GatewayImage) || ref.NetworkHash != "" && !networkDigest.MatchString(ref.NetworkHash) || ref.Fingerprint != fingerprint(b, ref) {
		return false
	}
	prefix := "ts-recording-" + ref.Fingerprint[:32]
	return ref.Container == prefix+"-browser" && ref.Gateway == prefix+"-gateway" && ref.Volume == prefix+"-output"
}

func labels(b recordingworker.Binding, ref recordingworker.Reference) map[string]string {
	return map[string]string{"teststudio.recording": b.ID, "teststudio.recording-binding": b.Fingerprint(), "teststudio.recording-owner": ref.Fingerprint}
}

func labeled(args []string, b recordingworker.Binding, ref recordingworker.Reference) []string {
	for key, value := range labels(b, ref) {
		args = append(args, "--label", key+"="+value)
	}
	return args
}

func (r *Runtime) run(ctx context.Context, args ...string) error {
	command := exec.CommandContext(ctx, r.binary(), args...)
	if command.Run() != nil {
		return errors.New("an owned recording container operation could not finish")
	}
	return nil
}

func (r *Runtime) Start(ctx context.Context, b recordingworker.Binding, ref recordingworker.Reference, target browserpolicy.Target) (recording.InteractiveDriver, error) {
	expected, err := r.Plan(b, target)
	if err != nil || expected != ref || !validReference(b, ref) {
		return nil, errors.New("recording resources do not match their durable reservation")
	}
	if err := browserpolicy.ValidateTarget(target); err != nil {
		return nil, err
	}
	deadline, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for _, item := range []struct{ kind, name string }{{"container", ref.Container}, {"container", ref.Gateway}, {"volume", ref.Volume}} {
		_, exists, err := r.inspect(deadline, item.kind, item.name)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("reserved recording resources already exist; recover cleanup before starting another browser")
		}
	}
	volume := labeled([]string{"volume", "create"}, b, ref)
	if err := r.run(deadline, append(volume, ref.Volume)...); err != nil {
		return nil, err
	}
	args := []string{"create", "--pull=never", "--name", ref.Container, "--init", "--network=none", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--security-opt=seccomp=" + r.Seccomp, "--pids-limit=256", "--memory=2g", "--cpus=2", "--shm-size=256m", "--user=1000:1000", "--log-driver=none", "--tmpfs", "/tmp:rw,exec,nosuid,nodev,size=1g,mode=1777", "--mount", "type=volume,src=" + ref.Volume + ",dst=/output", "--env", "HOME=/tmp", "--env", "TESTSTUDIO_RECORDING_GATEWAY_SOCKET=/output/.browser-egress.sock", "--env", "TESTSTUDIO_RECORDING_DEADLINE=" + b.Deadline.Format(time.RFC3339Nano), "--interactive"}
	args = labeled(args, b, ref)
	args = append(args, ref.RuntimeImage, "/output/source.spec.ts", b.URL)
	if err := r.run(deadline, args...); err != nil {
		return nil, err
	}
	args = []string{"run", "--detach", "--pull=never", "--name", ref.Gateway, "--network=bridge", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=64", "--memory=128m", "--cpus=1", "--user=1000:1000", "--log-driver=none", "--mount", "type=volume,src=" + ref.Volume + ",dst=/output", "--tmpfs", "/run/teststudio:rw,nosuid,noexec,size=1048576,uid=1000,gid=1000,mode=0700", "--env", "TESTSTUDIO_GATEWAY_DEADLINE=" + b.Deadline.Format(time.RFC3339Nano)}
	args = labeled(args, b, ref)
	if err := r.run(deadline, append(args, ref.GatewayImage)...); err != nil {
		return nil, err
	}
	configuration, _ := json.Marshal(browsergateway.Configuration{Target: target})
	command := exec.CommandContext(deadline, r.binary(), "exec", "--interactive", ref.Gateway, "/gateway", "--configure")
	command.Stdin = bytes.NewReader(configuration)
	if command.Run() != nil {
		return nil, errors.New("recording gateway initialization failed")
	}
	for {
		if err := r.run(deadline, "exec", ref.Gateway, "/gateway", "--ready"); err == nil {
			break
		}
		select {
		case <-deadline.Done():
			return nil, errors.New("the recording gateway did not become ready")
		case <-time.After(100 * time.Millisecond):
		}
	}
	directory := filepath.Join(r.Directory, ref.Fingerprint)
	driver, err := embeddedplaywright.NewTransport(directory, func(id, address string) (*exec.Cmd, error) {
		if id != b.ID || address != b.URL {
			return nil, errors.New("recording transport does not match its session")
		}
		return exec.Command(r.binary(), "start", "--attach", "--interactive", ref.Container), nil
	})
	if err != nil {
		return nil, err
	}
	if err := driver.Start(deadline, b.ID, b.URL); err != nil {
		return driver, err
	}
	return driver, nil
}
