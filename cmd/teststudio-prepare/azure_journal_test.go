package main

import (
	"os"
	"strings"
	"testing"
)

func TestAzureAssemblyJournalRequiresTheWritableConfiguredSMBMount(t *testing.T) {
	path, account, share := "/private/source-journals", "studiorecovery", "source-journals"
	valid := "101 90 0:50 / /private/source-journals rw,nosuid,nodev,relatime shared:1 - cifs //studiorecovery.file.core.windows.net/source-journals rw,uid=65532,gid=65532,dir_mode=0700,file_mode=0600\n"
	if !azureJournalMount([]byte("1 0 0:1 / / rw - overlay overlay rw\n"+valid), path, account, share) {
		t.Fatal("configured private SMB mount was rejected")
	}
	directory := t.TempDir()
	if err := probeAzureJournalDirectory(directory); err != nil {
		t.Fatal("local journal write and synchronization probe failed", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 0 {
		t.Fatal("journal probe left private process files behind", err)
	}
	for name, data := range map[string]string{
		"missing":       "1 0 0:1 / / rw - overlay overlay rw\n",
		"ephemeral":     strings.Replace(valid, "cifs //studiorecovery.file.core.windows.net/source-journals", "tmpfs tmpfs", 1),
		"wrong-account": strings.Replace(valid, "//studiorecovery.", "//foreign.", 1),
		"wrong-share":   strings.Replace(valid, ".net/source-journals", ".net/other-journals", 1),
		"read-only":     strings.Replace(valid, " rw,nosuid", " ro,nosuid", 1),
		"subpath":       strings.Replace(valid, "0:50 / ", "0:50 /other ", 1),
		"duplicate":     valid + valid,
		"incomplete":    "101 90 0:50 / /private/source-journals rw\n",
	} {
		t.Run(name, func(t *testing.T) {
			if azureJournalMount([]byte(data), path, account, share) {
				t.Fatal("incompatible journal mount was accepted")
			}
		})
	}
}
