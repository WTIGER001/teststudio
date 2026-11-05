package main

import (
	"errors"
	"io"
	"os"
	"strings"
)

func validateAzureJournal(directory, account, share string) error {
	failure := errors.New("Azure assembly requires its configured writable SMB journal mount with file and directory synchronization")
	mounts, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return failure
	}
	data, err := io.ReadAll(io.LimitReader(mounts, (1<<20)+1))
	closeErr := mounts.Close()
	if err != nil || closeErr != nil || !azureJournalMount(data, directory, account, share) {
		return failure
	}
	if err := probeAzureJournalDirectory(directory); err != nil {
		return failure
	}
	return nil
}

func probeAzureJournalDirectory(directory string) error {
	probe, err := os.CreateTemp(directory, ".teststudio-journal-probe-")
	if err != nil {
		return err
	}
	name := probe.Name()
	defer os.Remove(name)
	_, writeErr := probe.WriteString("teststudio-journal-probe\n")
	syncErr, fileCloseErr := probe.Sync(), probe.Close()
	parent, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer parent.Close()
	if err := errors.Join(writeErr, syncErr, fileCloseErr, parent.Sync()); err != nil {
		return err
	}
	if err := os.Remove(name); err != nil {
		return err
	}
	return parent.Sync()
}

func azureJournalMount(data []byte, directory, account, share string) bool {
	if len(data) > 1<<20 {
		return false
	}
	matched := false
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, " - ", 2)
		fields := strings.Fields(parts[0])
		if len(fields) < 5 || fields[4] != directory {
			continue
		}
		if matched || len(fields) < 6 || len(parts) != 2 || fields[3] != "/" {
			return false
		}
		volume := strings.Fields(parts[1])
		if len(volume) != 3 || volume[0] != "cifs" || volume[1] != "//"+account+".file.core.windows.net/"+share {
			return false
		}
		options := strings.Split(fields[5], ",")
		writable := false
		for _, option := range options {
			if option == "ro" {
				return false
			}
			writable = writable || option == "rw"
		}
		if !writable {
			return false
		}
		matched = true
	}
	return matched
}
