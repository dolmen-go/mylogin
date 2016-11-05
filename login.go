package mylogin

import (
	"bytes"
	"fmt"
	"net"
	"strings"
)

type Login struct {
	User     *string `json:"user,omitempty"`
	Password *string `json:"password,omitempty"`
	Host     *string `json:"host,omitempty"`
	Port     *string `json:"port,omitempty"`
	Socket   *string `json:"socket,omitempty"`
}

// IsEmpty is true if l is nil or none of the fields are set
func (l *Login) IsEmpty() bool {
	return l == nil ||
		(l.User == nil &&
			l.Password == nil &&
			l.Host == nil &&
			l.Port == nil &&
			l.Socket == nil)
}

// DSN builds a DSN for github.com/go-sql-driver/mysql
func (l *Login) DSN() string {
	var b bytes.Buffer
	if l.User != nil {
		b.WriteString(*l.User)
		if l.Password != nil {
			b.WriteByte(':')
			b.WriteString(*l.Password)
		}
		b.WriteByte('@')
	}
	if l.Socket != nil {
		b.WriteString("unix(")
		b.WriteString(*l.Socket)
		b.WriteByte(')')
	} else if l.Host != nil || l.Port != nil {
		var host, port string
		if l.Host != nil {
			host = *l.Host
		}
		if l.Port != nil {
			port = *l.Port
		}
		b.WriteString("tcp(")
		b.WriteString(net.JoinHostPort(host, port))
		b.WriteByte(')')
	}
	if b.Len() > 0 {
		b.WriteByte('/')
	}
	return b.String()
}

// String returns DSN()
func (l *Login) String() string {
	return l.DSN()
}

var unescape = strings.NewReplacer(
	`\b`, "\b",
	`\t`, "\t",
	`\n`, "\n",
	`\r`, "\r",
	`\\`, `\`,
	`\s`, ` `,
).Replace

func (c *Login) parseLine(line string) error {
	s := strings.SplitN(line, " = ", 2)

	s[1] = unescape(s[1])

	switch s[0] {
	case "user":
		c.User = &s[1]
	case "password":
		c.Password = &s[1]
	case "host":
		c.Host = &s[1]
	case "port":
		c.Port = &s[1]
	case "socket":
		c.Socket = &s[1]
	default:
		return fmt.Errorf("Unknown option '%s'", s[0])
	}
	return nil
}

func (login *Login) Merge(l *Login) {
	if l.User != nil {
		login.User = l.User
	}
	if l.Password != nil {
		login.Password = l.Password
	}
	if l.Host != nil {
		login.Host = l.Host
	}
	if l.Port != nil {
		login.Port = l.Port
	}
	if l.Socket != nil {
		login.Socket = l.Socket
	}
}
