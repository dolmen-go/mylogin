package main

import (
	"bufio"
	"encoding/binary"
	"io"
	"log"
	"os"
	"unsafe"

	"github.com/dolmen-go/mylogin"
)

var nativeByteOrder binary.ByteOrder

func init() {
	// http://grokbase.com/t/gg/golang-nuts/129jhmdb3d/go-nuts-how-to-tell-endian-ness-of-machine#20120918nttlyywfpl7ughnsys6pm4pgpe
	var x int32 = 0x01020304
	switch *(*byte)(unsafe.Pointer(&x)) {
	case 1:
		nativeByteOrder = binary.BigEndian
	case 4:
		nativeByteOrder = binary.LittleEndian
	}
}

func main() {
	var filename string

	if len(os.Args) > 1 {
		filename = os.Args[1]
	} else {
		filename = mylogin.DefaultFile()
	}

	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	rd, err := mylogin.Decode(bufio.NewReader(file), nativeByteOrder)
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) == 3 {
		rd = mylogin.FilterSection(rd, os.Args[2])
	}

	_, err = io.Copy(os.Stdout, rd)
	if err != nil {
		log.Fatal(err)
	}
}
