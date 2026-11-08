package recordingworker

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sort"

	"github.com/wtiger001/teststudio/internal/browserpolicy"
)

type OriginPolicy map[string][]string

func LoadOriginPolicy(filename string) (OriginPolicy, error) {
	if filename == "" {
		return nil, nil
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, errors.New("recording origin policy is unavailable")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 65537))
	if err != nil || len(data) > 64<<10 {
		return nil, errors.New("recording origin policy exceeds its limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	var policy OriginPolicy
	var extra any
	if decoder.Decode(&policy) != nil || policy == nil || decoder.Decode(&extra) != io.EOF {
		return nil, errors.New("recording origin policy must contain one bounded JSON origin map")
	}
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	return policy, nil
}

func (p OriginPolicy) Validate() error {
	if len(p) > 64 {
		return errors.New("recording origin policy permits at most 64 starting origins")
	}
	for origin, additional := range p {
		if err := browserpolicy.ValidateTarget(browserpolicy.Target{Origin: origin, AdditionalOrigins: additional}); err != nil {
			return errors.New("recording origin policy requires distinct exact HTTP or HTTPS origins without paths, wildcards, or credentials")
		}
	}
	return nil
}

func (p OriginPolicy) additional(origin string) []string {
	result := append([]string(nil), p[origin]...)
	sort.Strings(result)
	return result
}
