package main

import (
	"errors"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func sandboxIdentity() error {
	err := errors.New("sandbox supervision requires the unprivileged Linux identity and process restrictions")
	if os.Geteuid() != 0 && (os.Getuid() != 1000 || os.Geteuid() != 1000 || os.Getgid() != 1000 || os.Getegid() != 1000) {
		return err
	}
	if _, _, code := syscall.AllThreadsSyscall6(syscall.SYS_PRCTL, unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0, 0); code != 0 {
		return err
	}
	if _, _, code := syscall.AllThreadsSyscall6(syscall.SYS_PRCTL, unix.PR_SET_KEEPCAPS, 0, 0, 0, 0, 0); code != 0 {
		return err
	}
	if os.Geteuid() == 0 {
		if syscall.Setgroups([]int{}) != nil || syscall.Setresgid(1000, 1000, 1000) != nil || syscall.Setresuid(1000, 1000, 1000) != nil {
			return err
		}
	}
	if os.Getuid() != 1000 || os.Geteuid() != 1000 || os.Getgid() != 1000 || os.Getegid() != 1000 {
		return err
	}
	groups, groupErr := os.Getgroups()
	if groupErr != nil {
		return err
	}
	for _, group := range groups {
		if group != 1000 {
			return err
		}
	}
	var header unix.CapUserHeader
	var capabilities [2]unix.CapUserData
	header.Version = unix.LINUX_CAPABILITY_VERSION_3
	if unix.Capget(&header, &capabilities[0]) != nil {
		return err
	}
	for _, capability := range capabilities {
		if capability.Effective != 0 || capability.Permitted != 0 || capability.Inheritable != 0 {
			return err
		}
	}
	if unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0) != nil {
		return err
	}
	return nil
}
