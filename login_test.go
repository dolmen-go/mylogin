package mylogin

import "testing"

func TestParseLine(t *testing.T) {
	for _, test := range []struct {
		line string
		user string
	}{
		{`user = toto`, `toto`},
		{`user = toto titi`, `toto titi`},
		{`user = "toto"`, `toto`},
		{`user = "toto titi"`, `toto titi`},
		{`user = "toto = titi"`, `toto = titi`},
		{`user = "toto \" titi"`, `toto " titi`},
	} {
		var l Login
		err := l.parseLine(test.line)
		if err != nil {
			t.Errorf("%q: %v", test.line, err)
			continue
		}
		if *l.User != test.user {
			t.Errorf("%q: got %q, expected %q", test.line, *l.User, test.user)
		}
	}
}
