// Command mylogin allows to dump the content of ~/.my.cnf.

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/dolmen-go/mylogin"
)

type outputFormat interface {
	Help() (string, string)
	flag.Getter
	Print(w io.Writer, section *mylogin.Section) error
}

// outputFormatBool is an abstract base format for formats defined as a bool CLI flag.
type outputFormatBool struct {
	bool
}

func (outputFormatBool) IsBoolFlag() bool {
	return true
}

func (f *outputFormatBool) String() string {
	return strconv.FormatBool(f.bool)
}

func (f *outputFormatBool) Set(s string) error {
	ok, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	f.bool = ok
	return nil
}

func (f *outputFormatBool) Get() interface{} {
	if !f.bool {
		return nil
	}
	return true
}

type formatReplay struct {
	outputFormatBool
}

func (formatReplay) Help() (string, string) {
	return "replay", "mysql_config_editor 'set' command format (note: password is not exported)"
}

func (formatReplay) Print(w io.Writer, section *mylogin.Section) error {
	args := make([]string, 5, 5+5*2)
	args[0] = `mysql_config_editor`
	args[1] = `set`
	args[2] = `--skip-warn`
	args[3] = `-G`
	args[4] = section.Name
	if section.Login.User != nil {
		args = append(args, `-u`, *section.Login.User)
	}
	if section.Login.Password != nil {
		args = append(args, `-p`)
	}
	if section.Login.Host != nil {
		args = append(args, `-h`, *section.Login.Host)
	}
	if section.Login.Port != nil {
		args = append(args, `-P`, *section.Login.Port)
	}
	if section.Login.Socket != nil {
		args = append(args, `-S`, *section.Login.Socket)
	}
	_, err := fmt.Fprintln(w, strings.Join(args, " "))
	return err
}

type formatRemove struct {
	outputFormatBool
}

func (formatRemove) Help() (string, string) {
	return "remove", "mysql_config_editor 'remove' command format"
}

func (formatRemove) Print(w io.Writer, section *mylogin.Section) error {
	_, err := fmt.Fprintln(w, "mysql_config_editor remove -G", section.Name)
	return err
}

func loginAsMap(login *mylogin.Login) map[string]interface{} {
	// The login struct contains *string
	// This is not convenient to use in templates
	// So we remap it to a map, skipping nil values
	m := make(map[string]interface{})
	for _, x := range []struct {
		key   string
		value *string
	}{
		{"user", login.User},
		{"password", login.Password},
		{"host", login.Host},
		{"socket", login.Socket},
		{"port", login.Port},
	} {
		if x.value != nil {
			m[x.key] = *x.value
		}
	}

	return m
}

type formatJSON struct {
	outputFormatBool
}

func (formatJSON) Help() (string, string) {
	return "json", "JSON format"
}

func (formatJSON) Print(w io.Writer, section *mylogin.Section) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(loginAsMap(&section.Login))
}

type formatTemplate struct {
	tmpl *template.Template
}

func (formatTemplate) Help() (string, string) {
	return "template", "text/template format"
}

func (f *formatTemplate) String() string {
	if f.tmpl == nil {
		return ""
	}
	return "<template>"
}

func (f *formatTemplate) Set(s string) error {
	tmpl, err := template.New("user-template").Parse(s)
	if err != nil {
		return err
	}
	(*f).tmpl = tmpl
	return nil
}

func (f *formatTemplate) Get() interface{} {
	if f.tmpl == nil {
		return nil
	}
	return f
}

func (f *formatTemplate) Print(w io.Writer, section *mylogin.Section) error {
	err := f.tmpl.Execute(os.Stdout, loginAsMap(&section.Login))
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w)
	return err
}

func main() {
	var filename string
	flag.StringVar(&filename, "file", mylogin.DefaultFile(), "mylogin.cnf path")

	formats := []outputFormat{
		&formatReplay{},
		&formatRemove{},
		&formatJSON{},
		&formatTemplate{},
	}

	for _, fmt := range formats {
		name, usage := fmt.Help()
		flag.Var(fmt, name, usage)
	}

	flag.Parse()

	var selectedFormat outputFormat
	for _, ft := range formats {
		f := ft.Get()
		if f == nil {
			continue
		}
		if selectedFormat != nil {
			h1, _ := ft.Help()
			h2, _ := selectedFormat.Help()
			fmt.Fprintf(os.Stderr, "options -%s and -%s are exclusive.\n", h1, h2)
			flag.Usage()
			os.Exit(1)
		}
		selectedFormat = ft
	}

	if selectedFormat != nil {

		if flag.NArg() != 0 {

			for _, name := range flag.Args() {
				login, err := mylogin.ReadLogin(filename, []string{name})
				if err != nil {
					log.Fatal(err)
				}
				if login == nil {
					log.Fatal("section doesn't exists")
				}

				err = selectedFormat.Print(os.Stdout, &mylogin.Section{Name: name, Login: *login})
				if err != nil {
					log.Fatal(err)
				}
			}
		} else {
			sections, err := mylogin.ReadSections(filename)
			if err != nil {
				log.Fatal(err)
			}

			for i := range sections {
				err = selectedFormat.Print(os.Stdout, &sections[i])
				if err != nil {
					log.Fatal(err)
				}
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
