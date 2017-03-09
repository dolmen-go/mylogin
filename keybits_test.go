package mylogin_test

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/dolmen-go/mylogin"
)

// mysql_config_editor generates files with a key where the high 3 bits
// of each byte are always cleared. Let's check by fuzzing.
//
// See also cmd/mylogin-key
func TestKeyBits(t *testing.T) {
	mysql_config_editor, err := exec.LookPath("mysql_config_editor")
	if err != nil {
		t.Logf("mysql_config_editor not found in PATH")
		return
	}

	tempDir, err := ioutil.TempDir("", "keybits-")
	if err != nil {
		t.Fatalf("ioutil.TempDir: %s", err)
	}
	defer os.RemoveAll(tempDir)

	timeout := time.NewTimer(800 * time.Millisecond)
Loop:
	for i := 0; i < 100000; i++ {
		testFileKeyBits(t, mysql_config_editor, filepath.Join(tempDir, fmt.Sprintf("%08d.cnf", i)))

		select {
		case <-timeout.C:
			break Loop
		default:
		}
	}
}

func testFileKeyBits(t *testing.T, mysql_config_editor string, filename string) {
	cmd := exec.Command(mysql_config_editor, `set`, `--login-path=toto`)
	cmd.Env = append(
		os.Environ(),
		"MYSQL_TEST_LOGIN_FILE="+filename,
	)
	var err error
	cmd.Stdout, err = os.OpenFile(os.DevNull, os.O_WRONLY, 0700)
	if err != nil {
		t.Fatalf("%s: %s", os.DevNull, err)
	}
	cmd.Stderr = os.Stderr
	defer os.Remove(filename)

	err = cmd.Run()
	if err != nil {
		t.Fatalf("%s: %s", filename, err)
	}

	f, err := os.Open(filename)
	if err != nil {
		t.Fatalf("%s: %s", filename, err)
	}
	defer f.Close()

	file, err := mylogin.Decode(bufio.NewReader(f))
	if err != nil {
		t.Fatalf("%s: %s", filename, err)
	}
	key := file.Key()

	// Check that each byte of the key has the 3 high bits clear
	for _, b := range key {
		if b >= 32 {
			t.Errorf("%s: %X (more than 5 bits in key)", filename, key)
			return
		}
	}

	t.Logf("%s: %X (3 high bits always cleared)", filename, key)
}
