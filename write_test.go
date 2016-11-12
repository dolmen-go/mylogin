package mylogin_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/dolmen-go/mylogin"
)

type fileInfoByName []os.FileInfo

func (s fileInfoByName) Len() int           { return len(s) }
func (s fileInfoByName) Less(i, j int) bool { return s[i].Name() < s[j].Name() }
func (s fileInfoByName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func iterDir(path string, filter func(os.FileInfo) bool) (chan string, error) {
	if filter == nil {
		filter = func(os.FileInfo) bool { return true }
	}
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	files, err := dir.Readdir(-1)
	if err != nil {
		return nil, err
	}
	c := make(chan string)
	go func() {
		sort.Sort(fileInfoByName(files))
		// Ignore write on closed channel
		defer func() {
			// FIXME
			_ = recover()
		}()
		for _, fileinfo := range files {
			if !filter(fileinfo) {
				continue
			}
			c <- fmt.Sprintf("%s%c%s", path, os.PathSeparator, fileinfo.Name())
		}
		close(c)
	}()
	return c, nil
}

func TestReadWrite(t *testing.T) {
	files, err := iterDir("testdata", func(f os.FileInfo) bool {
		return f.Mode().IsRegular() && strings.HasSuffix(f.Name(), ".cnf")
	})
	if err != nil {
		t.Fatal(err)
	}

	var orig bytes.Buffer
	var out bytes.Buffer

	for path := range files {
		t.Logf(path)
		f, err := os.Open(path)
		if err != nil {
			t.Errorf("%s: %s", err)
			continue
		}
		func() {
			defer f.Close()
			orig.Reset()
			out.Reset()
			io.Copy(&orig, f)

			origBytes := orig.Bytes()
			content, err := mylogin.Decode(bytes.NewBuffer(orig.Bytes()))
			if err != nil {
				t.Errorf("%s: %s", path, err)
				return
			}
			err = mylogin.Encode(&out, content)
			if err != nil {
				t.Errorf("%s: %s", path, err)
				return
			}
			outBytes := out.Bytes()
			if bytes.Compare(origBytes, outBytes) == 0 {
				t.Logf("%s: OK", path)
				return
			}
			t.Errorf("%s: content differ", path)
			if len(outBytes) != len(origBytes) {
				t.Logf("orig: %d bytes", len(origBytes))
				t.Logf("out:  %d bytes", len(outBytes))
			}

		}()
	}
}
