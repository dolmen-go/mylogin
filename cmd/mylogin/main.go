package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"text/template"

	"github.com/dolmen-go/mylogin"
)

func main() {
	var filename string
	var formatJSON bool
	var formatTemplate string

	flag.StringVar(&filename, "file", mylogin.DefaultFile(), "mylogin.cnf path")
	flag.BoolVar(&formatJSON, "json", false, "JSON output")
	flag.StringVar(&formatTemplate, "template", "", "text/template format")
	flag.Parse()

	if formatJSON || formatTemplate != "" {
		if flag.NArg() == 0 {
			log.Fatal("missing section name")
		}
		login, err := mylogin.ReadLogin(filename, []string{flag.Arg(0)})
		if err != nil {
			log.Fatal(err)
		}
		if login == nil {
			log.Fatal("section doesn't exists")
		}
		if formatJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetEscapeHTML(false)
			enc.SetIndent("", "  ")
			enc.Encode(login)
		} else {
			tmpl, err := template.New("user-template").Parse(formatTemplate)
			if err != nil {
				log.Fatal(err)
			}
			err = tmpl.Execute(os.Stdout, login)
			if err != nil {
				log.Fatal(err)
			}
		}
	} else {
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
}
