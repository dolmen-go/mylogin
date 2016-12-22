package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"os"

	"github.com/dolmen-go/mylogin"
)

func main() {
	var filename string

	flag.StringVar(&filename, "file", mylogin.DefaultFile(), "mylogin.cnf path")
	flag.Parse()

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

	if flag.NArg() > 0 {
		rd = mylogin.FilterSection(rd, flag.Arg(0))
	}

	_, err = io.Copy(os.Stdout, rd)
	if err != nil {
		log.Fatal(err)
	}
}
