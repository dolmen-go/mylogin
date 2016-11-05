package main

import (
	"bufio"
	"io"
	"log"
	"os"

	"github.com/dolmen-go/mylogin"
)

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

	f, err := mylogin.Decode(bufio.NewReader(file))
	if err != nil {
		log.Fatal(err)
	}
	rd := f.PlainText()

	if len(os.Args) == 3 {
		rd = mylogin.FilterSection(rd, os.Args[2])
	}

	_, err = io.Copy(os.Stdout, rd)
	if err != nil {
		log.Fatal(err)
	}
}
