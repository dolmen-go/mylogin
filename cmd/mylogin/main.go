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
	Print(w io.Writer, login *mylogin.Login) error
}

type formatReplay bool

func (formatReplay) Help() (string, string) {
	return "replay", "mysql_config_editor commands format"
}

func (formatReplay) IsBoolFlag() bool {
	return true
}

func (f *formatReplay) String() string {
	return strconv.FormatBool(bool(*f))
}

func (f *formatReplay) Set(s string) error {
	ok, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	*f = formatReplay(ok)
	return nil
}

func (f *formatReplay) Get() interface{} {
	if !*f {
		return nil
	}
	return f
}

func (formatReplay) Print(w io.Writer, login *mylogin.Login) error {
	args := make([]string, 5, 5+5*2)
	args[0] = `mysql_config_editor`
	args[1] = `set`
	args[2] = `--skip-warn`
	args[3] = `-G`
	args[4] = flag.Arg(0)
	if login.User != nil {
		args = append(args, `-u`, *login.User)
	}
	if login.Password != nil {
		args = append(args, `-p`)
	}
	if login.Host != nil {
		args = append(args, `-h`, *login.Host)
	}
	if login.Port != nil {
		args = append(args, `-P`, *login.Port)
	}
	if login.Socket != nil {
		args = append(args, `-S`, *login.Socket)
	}
	_, err := fmt.Fprintln(w, strings.Join(args, " "))
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

type formatJSON bool

func (formatJSON) Help() (string, string) {
	return "json", "JSON format"
}

func (formatJSON) IsBoolFlag() bool {
	return true
}

func (f *formatJSON) String() string {
	return strconv.FormatBool(bool(*f))
}

func (f *formatJSON) Set(s string) error {
	ok, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	*f = formatJSON(ok)
	return nil
}

func (f *formatJSON) Get() interface{} {
	if !*f {
		return nil
	}
	return f
}

func (formatJSON) Print(w io.Writer, login *mylogin.Login) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(loginAsMap(login))
}

type formatTemplate struct {
	tmpl *template.Template
}

func (formatTemplate) Help() (string, string) {
	return "template", "text/template format"
}

func (f *formatTemplate) String() string {
	if (*f).tmpl == nil {
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

func (f *formatTemplate) Print(w io.Writer, login *mylogin.Login) error {
	err := f.tmpl.Execute(os.Stdout, loginAsMap(login))
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w)
	return err
}

func main() {
	var filename string
	flag.StringVar(&filename, "file", mylogin.DefaultFile(), "mylogin.cnf path")
	//var formatJSON, formatReplay bool
	//var formatTemplate string

	fmtReplay := formatReplay(false)
	fmtJSON := formatJSON(false)
	formats := []outputFormat{
		&fmtReplay,
		&fmtJSON,
		&formatTemplate{},
	}

	for _, fmt := range formats {
		name, usage := fmt.Help()
		flag.Var(fmt, name, usage)
	}

	flag.Parse()

	var selectedFormat outputFormat
	for _, fmt := range formats {
		f := fmt.Get()
		if f == nil {
			continue
		}
		if selectedFormat != nil {
			flag.Usage()
		}
		selectedFormat = f.(outputFormat)
	}

	if selectedFormat != nil {

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

		err = selectedFormat.Print(os.Stdout, login)
		if err != nil {
			log.Fatal(err)
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
