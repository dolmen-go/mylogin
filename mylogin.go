// package mylogin reads ~/.mylogin.cnf created by mysql_config_editor
//
// See https://dev.mysql.com/doc/refman/5.7/en/mysql-config-editor.html
//
// Example:
//   mysql_config_editor set --login-path=foo --user=bar -p
package mylogin

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

type Config struct {
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     string `json:"port,omitempty"`
	Socket   string `json:"socket,omitempty"`
	Warn     bool   `json:"warn"`
}

func (c *Config) parseLine(line string) error {
	s := strings.SplitN(line, " = ", 2)
	switch s[0] {
	case "user":
		c.User = s[1]
	case "password":
		c.Password = s[1]
	case "host":
		c.Host = s[1]
	case "port":
		c.Port = s[1]
	case "socket":
		c.Socket = s[1]
	case "warn":
		c.Warn = strings.EqualFold(s[1], "TRUE")
	default:
		return fmt.Errorf("Unknown option '%s'", s[0])
	}
	return nil
}

type ConfigSection struct {
	Name   string `json:"name"`
	Config Config `json:"config"`
}

/*
// TODO Load ~/.mylogin.cnf
func ReadMyLogin(section string) (c *Config, err error) {

}
*/

func ReadConfig(file string, section string) (c *Config, err error) {
	if section == "" {
		section = "client"
	}
	sections, err := ReadAll(file)
	if err != nil {
		return
	}
	for _, s := range sections {
		if s.Name == section {
			c = &s.Config
			break
		}
	}
	return
}

func ReadAll(filename string) (sections []ConfigSection, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	rd, err := Decode(bufio.NewReader(file))
	if err != nil {
		return
	}

	var config *Config
	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		line := scanner.Text()
		if line[0] == '[' {
			sections = append(sections,
				ConfigSection{Name: line[1 : len(line)-2]})
			config := &sections[len(sections)-1].Config
			config.Warn = true // default
		} else if config != nil && line != "" {
			if err = config.parseLine(line); err != nil {
				return nil, err
			}
		}
	}
	return
}

// AES 128
func Decode(in io.Reader) (io.Reader, error) {
	// http://ocelot.ca/blog/blog/2015/05/21/decrypt-mylogin-cnf/

	// Skip first 4 bytes
	head4 := make([]byte, 4)
	n, err := in.Read(head4)
	if err != nil {
		return nil, err
	}
	if n != 4 {
		return nil, io.EOF
	}

	fileKey := make([]byte, 20)
	n, err = in.Read(fileKey)
	if err != nil {
		return nil, err
	}
	if n != 20 {
		return nil, io.EOF
	}

	// Apply xor
	key := make([]byte, 16)
	for i := 0; i < 20; i++ {
		key[i%16] ^= fileKey[i]
	}

	// log.Printf("Key: %v\n", key)

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err.Error())
	}

	return &decoder{input: in, block: block}, nil
}

// FIXME use the platform byte order as default
var ByteOrder = binary.LittleEndian

type decoder struct {
	input  io.Reader
	block  cipher.Block
	chunk  [4096]byte
	buffer []byte // Slice pointing to chunk
}

func (d *decoder) Read(buf []byte) (n int, err error) {
	if len(buf) == 0 {
		return
	}
	if len(d.buffer) > 0 {
		n = copy(buf, d.buffer)
		d.buffer = d.buffer[n:]
		return
	}
	var size int32
	for {
		// Read a new chunk
		err = binary.Read(d.input, ByteOrder, &size)
		if err != nil {
			return 0, err
		}
		if size != 0 {
			break
		}
	}
	if size < 0 || int(size) > len(d.chunk) || size%aes.BlockSize != 0 {
		return 0, fmt.Errorf("invalid block size: %d", size)
	}
	n, err = d.input.Read(d.chunk[:size])
	if n != int(size) {
		if err == nil {
			err = io.EOF
		}
		return 0, err
	}

	// Each 16-bytes block is encoded with a null IV
	d.buffer = d.chunk[:size]
	for i := 0; i < int(size); i += aes.BlockSize {
		cbc := cipher.NewCBCDecrypter(d.block, make([]byte, aes.BlockSize))
		b := d.chunk[i : i+aes.BlockSize]
		cbc.CryptBlocks(b, b)

	}

	// Remove PKCS#7 padding
	// last byte value gives the number of padding byte
	// each padding byte has that value
	padding := d.buffer[len(d.buffer)-1]
	// Note that mysql_config_editor generates up to 16 bytes of padding
	// which is a full AES block, so 16 encrypted bytes just to be drop when
	// reading.
	// Is it a bug or some nasty redundancy to reveal the encryption key?
	if padding > 0 && padding <= aes.BlockSize {
		//log.Printf("Padding: %d\n", padding)
		for _, c := range d.buffer[len(d.buffer)-int(padding):] {
			if c != padding {
				padding = 0
				break
			}
		}
		d.buffer = d.buffer[:len(d.buffer)-int(padding)]
	}

	n = copy(buf, d.buffer)
	d.buffer = d.buffer[n:]
	return
}

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
