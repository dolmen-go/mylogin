package mylogin

import (
	"bufio"
	"bytes"
	"io"
)

// FilterSection reads an INI-style content and filter out any section
// except the given one
func FilterSection(rd io.Reader, section string) io.Reader {
	header := make([]byte, 1, 2+len(section))
	header[0] = '['
	header = append(header, section...)
	header = append(header, ']')
	return &filterSection{header: header, scanner: bufio.NewScanner(rd)}
}

type filterSection struct {
	header  []byte
	show    bool
	scanner *bufio.Scanner
	buffer  bytes.Buffer
}

func (f *filterSection) Read(buf []byte) (n int, err error) {
	if len(buf) == 0 {
		return
	}
	for f.buffer.Len() == 0 {
		if !f.scanner.Scan() {
			err = f.scanner.Err()
			if err == nil {
				err = io.EOF
			}
			return
		}
		line := f.scanner.Bytes()
		if line[0] == '[' {
			f.show = bytes.Equal(f.header, line)
		}
		if f.show {
			f.buffer.Write(line)
			f.buffer.WriteByte('\n')
		}
	}

	n, err = f.buffer.Read(buf)
	if err == io.EOF {
		err = nil
	}
	return
}
