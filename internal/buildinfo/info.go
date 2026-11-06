package buildinfo

import (
	"encoding/base64"
	"encoding/json"
)

var Manifest string

type Info struct {
	Version        string `json:"version"`
	DisplayVersion string `json:"displayVersion"`
	Major          int    `json:"major"`
	Minor          int    `json:"minor"`
	Patch          int    `json:"patch"`
	Tag            string `json:"tag"`
	BuildID        string `json:"buildId"`
	BuiltAt        string `json:"builtAt"`
	Commit         string `json:"commit"`
	SourceHash     string `json:"sourceHash"`
	Modified       bool   `json:"modified"`
}

var current = decode(Manifest)

func Current() Info { return current }

func decode(value string) Info {
	var result Info
	data, err := base64.StdEncoding.DecodeString(value)
	if err == nil && len(data) > 0 && json.Unmarshal(data, &result) == nil && result.Version != "" && result.BuildID != "" {
		return result
	}
	return Info{Version: "development", DisplayVersion: "development", Tag: "unversioned"}
}
