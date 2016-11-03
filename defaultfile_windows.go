package mylogin

import "os"

func platformDefaultFile() string {
	return os.ExpandEnv(`${APPDATA}\MySQL\.mylogin.cnf`)
}
