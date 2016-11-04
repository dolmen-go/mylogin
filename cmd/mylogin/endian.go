// +build !amd64

package main

import (
	"encoding/binary"
	"unsafe"
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
