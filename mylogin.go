// Package mylogin reads and writes ~/.mylogin.cnf created by mysql_config_editor.
//
// Reference documentation:
//  - https://dev.mysql.com/doc/refman/8.0/en/mysql-config-editor.html
//  - https://dev.mysql.com/doc/refman/8.0/en/option-file-options.html#option_general_login-path
//
// Example:
//
//	mysql_config_editor set --login-path=foo --user=bar -p
//
// For usage examples, see the utilies in the same repo.
package mylogin

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

// DefaultSection is the name of the base section used by all MySQL client tools.
const DefaultSection = "client"

// Key is a key used for encryption of mylogin.cnf files.
type Key [20]byte

func (k Key) IsZero() bool {
	return k[0] == 0 && k == Key{}
}

func (k *Key) cipher() cipher.Block {
	// 16 bytes key for AES-128
	var aesKey [16]byte
	// Apply xor
	for i := 0; i < len(k); i++ {
		aesKey[i%16] ^= k[i]
	}

	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		panic(err.Error())
	}

	return block
}

// NewKey creates a new key from a source of random bytes.
// See [math/rand.Read] and [crypto/rand.Read] as possible sources.
//
// The generated key has the 3 high bits cleared so each byte is non-printable.
func NewKey(readRandom func([]byte) (int, error)) (Key, error) {
	var key Key
	// FIXME We will finally use only 5 bits of each byte.
	//       We should take much less bytes and spread them.
	_, err := readRandom(key[:])
	if err != nil {
		return Key{}, nil
	}
	for i := range key {
		// Clear the high bits
		key[i] = key[i] & 0x1F
	}
	return key, nil
}

// DefaultFile returns the path to the default mylogin.cnf file:
//
//	Windows: %APPDATA%/MySQL/.mylogin.cnf
//	others: ~/.mylogin.cnf
//
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
// merges them to obtain a single Login (that may be empty).
func ReadLogin(filename string, sectionNames []string) (login *Login, err error) {
	sections, err := ReadSections(filename)
	if err != nil {
		return
	}
	login = sections.Merge(sectionNames)
	return
}

// ReadSections reads all Sections of a mylogin.cnf file.
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
// and returns the structured content.
func Parse(rd io.Reader) (sections Sections, err error) {
	// Reference code: https://github.com/mysql/mysql-shell/blob/master/mysql-secret-store/login-path/login_path_helper.cc#L52
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

// File is the full structure of a mylogin.cnf file.
type File interface {
	// The key used for encrypting the file
	Key() Key
	// Byte ordering for saving int32 chunk sizes
	ByteOrder() binary.ByteOrder
	// The plaintext content of the file
	PlainText() io.Reader
}

type decoder struct {
	key       Key
	byteOrder binary.ByteOrder

	input  io.Reader
	chunk  [256 * aes.BlockSize]byte
	buffer []byte // Slice pointing to chunk
}

func (d *decoder) Key() Key {
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

// Decode is a filter that returns the plaintext content of a mylogin.cnf
// file.
// The file is encrypted with AES 128 CBC with the key embedded in the file.
func Decode(input io.Reader) (File, error) {
	// http://ocelot.ca/blog/blog/2015/05/21/decrypt-mylogin-cnf/

	in := bufio.NewReader(input)

	// Skip first 4 bytes
	head4 := make([]byte, 4)
	n, err := io.ReadFull(in, head4)
	if err != nil {
		return nil, err
	}
	if n != 4 {
		return nil, io.EOF
	}

	var key Key
	if n, err = io.ReadFull(in, key[:]); err != nil {
		return nil, err
	}
	if n != cap(key) {
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

	return &decoder{key: key, input: in, byteOrder: byteOrder}, nil
}

// Read is the PlainText reader.
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
		if err = binary.Read(d.input, d.byteOrder, &size); err != nil {
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

	blockCipher := d.key.cipher()

	// Each 16-bytes block is encoded with a null IV
	d.buffer = d.chunk[:size]
	for i := 0; i < int(size); i += aes.BlockSize {
		cbc := cipher.NewCBCDecrypter(blockCipher, make([]byte, aes.BlockSize))
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

// Encode writes a mylogin.cnf content encrypted
func Encode(w io.Writer, f File) (err error) {
	key := f.Key()
	if key.IsZero() {
		return errors.New("key is not initialized")
	}

	//  Header
	if _, err = w.Write([]byte{0, 0, 0, 0}); err != nil {
		return
	}
	if _, err = w.Write(key[:]); err != nil {
		return
	}

	blockCipher := key.cipher()

	scanner := bufio.NewScanner(f.PlainText())

	// TODO scan strictly on \n (not \r\n)
	scanner.Split(bufio.ScanLines)

	var chunk [4096]byte
	byteOrder := f.ByteOrder()

	for scanner.Scan() {
		l := copy(chunk[:cap(chunk)-aes.BlockSize], scanner.Bytes())
		// ScanLines does not return the EOL, so we add it back
		chunk[l] = '\n'
		l++

		// There can be up to 16 (really, not 15) bytes of padding
		// in files generated by mysql_config_editor
		n := ((l + aes.BlockSize) / aes.BlockSize) * aes.BlockSize
		//fmt.Println("padding:", n-l)
		paddingChar := byte(n - l)
		for i := l; i < n; i++ {
			chunk[i] = paddingChar
		}

		for i := 0; i < n; i += aes.BlockSize {
			cbc := cipher.NewCBCEncrypter(blockCipher, make([]byte, aes.BlockSize))
			b := chunk[i : i+aes.BlockSize]
			cbc.CryptBlocks(b, b)
		}

		if err = binary.Write(w, byteOrder, int32(n)); err != nil {
			return
		}
		if _, err = w.Write(chunk[:n]); err != nil {
			return
		}
	}

	return
}

// TODO find a good name
type file struct {
	K  Key
	B  binary.ByteOrder
	PT []byte
}

func (f *file) Key() Key {
	return f.K
}

func (f *file) ByteOrder() binary.ByteOrder {
	return f.B
}

func (f *file) PlainText() io.Reader {
	return bytes.NewBuffer(f.PT)
}
