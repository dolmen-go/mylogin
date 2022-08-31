//go:build !windows
// +build !windows

package mylogin

import "os"

func platformDefaultFile() string {
	return os.ExpandEnv(`${HOME}/.mylogin.cnf`)
}
