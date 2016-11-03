package main

import (
	"bufio"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/dolmen-go/mylogin"
)

func main() {
	//mylogin.ReadConfig(os.Args[1], os.Args[2])
	file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	rd, err := mylogin.Decode(bufio.NewReader(file))
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) == 3 {
		rd = mylogin.FilterSection(rd, os.Args[2])
	}

	_, err = io.Copy(rd, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}
