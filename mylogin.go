// package mylogin reads ~/.mylogin.cnf created by mysql_config_editor
//
// See https://dev.mysql.com/doc/refman/5.7/en/mysql-config-editor.html
//
// Example:
//   mysql_config_editor set --login-path=foo --user=bar -p
package mylogin

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const DefaultSection = "client"

const KeySize = 20

// DefaultFile returns the path to the default mylogin.cnf file:
//   Windows: %APPDATA%/MySQL/.mylogin.cnf
//   others: ~/.mylogin.cnf
// If the environment variable MYSQL_TEST_LOGIN_FILE is set
// that path is returned instead.
func DefaultFile() string {
	f := os.Getenv("MYSQL_TEST_LOGIN_FILE")
	if len(f) != 0 {
		return f
	}
	// see defaultfile.go, defaultfile_windows.go
	return platformDefaultFile()
}

// ReadLogin reads a mylogin.cnf file, extracts the requested sections and
// merges them to obtain a single Login (that may be empty)
func ReadLogin(filename string, sectionNames []string) (login *Login, err error) {
	sections, err := ReadSections(filename)
	if err != nil {
		return
	}
	login = sections.Merge(sectionNames)
	return
}

// ReadSections reads all Sections of a mylogin.cnf file
func ReadSections(filename string) (sections Sections, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()

	file, err := Decode(bufio.NewReader(f))
	if err != nil {
		return
	}
	return Parse(file.PlainText())
}

// Parse parses the plaintext content of a mylogin.cnf file
func Parse(rd io.Reader) (sections Sections, err error) {
	var login *Login
	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		line := scanner.Text()
		if line[0] == '[' {
			sections = append(sections,
				Section{Name: line[1 : len(line)-1]})
			login = &sections[len(sections)-1].Login
		} else if login != nil && line != "" {
			if err = login.parseLine(line); err != nil {
				return nil, err
			}
		}
	}
	return
}

type File interface {
	Key() [KeySize]byte
	ByteOrder() binary.ByteOrder
	PlainText() io.Reader
}

type decoder struct {
	key       [KeySize]byte
	byteOrder binary.ByteOrder

	input  io.Reader
	block  cipher.Block
	chunk  [256 * aes.BlockSize]byte
	buffer []byte // Slice pointing to chunk
}

func (d *decoder) Key() [KeySize]byte {
	return d.key
}

func (d *decoder) ByteOrder() binary.ByteOrder {
	return d.byteOrder
}

func (d *decoder) PlainText() io.Reader {
	return d
}

func (d *decoder) Parse() (Sections, error) {
	return Parse(d)
}

// Decode is a filter that returns the plaintext content of a mylogin.cnf file.
// The file is encrypted with AES 128 with the key embeded in the file.
func Decode(input io.Reader) (File, error) {
	// http://ocelot.ca/blog/blog/2015/05/21/decrypt-mylogin-cnf/

	in := bufio.NewReader(input)

	// Skip first 4 bytes
	head4 := make([]byte, 4)
	n, err := in.Read(head4)
	if err != nil {
		return nil, err
	}
	if n != 4 {
		return nil, io.EOF
	}

	var key [KeySize]byte
	n, err = in.Read(key[:])
	if err != nil {
		return nil, err
	}
	if n != len(key) {
		return nil, io.EOF
	}
	// log.Printf("Key: %v\n", key)

	// The following 4 bytes are the length of the first chunk
	// We will use them to detect the byte order
	chunkSize, err := in.Peek(4)
	if err != nil {
		return nil, err
	}
	var byteOrder binary.ByteOrder
	// Assume all chunks have size < 64K
	if chunkSize[0] == 0 && chunkSize[1] == 0 && (chunkSize[2] != 0 || chunkSize[3] != 0) {
		byteOrder = binary.BigEndian
	} else {
		byteOrder = binary.LittleEndian
	}

	// 16 bytes key for AES-128
	var aesKey [16]byte
	// Apply xor
	for i := 0; i < KeySize; i++ {
		aesKey[i%16] ^= key[i]
	}

	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		panic(err.Error())
	}

	return &decoder{key: key, input: in, byteOrder: byteOrder, block: block}, nil
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
		err = binary.Read(d.input, d.byteOrder, &size)
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

/*
// TODO

func Encode(io.Writer, f File) error {
}
*/
