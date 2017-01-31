package mylogin

import (
	"bytes"
	"fmt"
	"net"
	"strings"
)

// Login is the content of a section of mylogin.cnf.
type Login struct {
	User     *string `json:"user,omitempty"`
	Password *string `json:"password,omitempty"`
	Host     *string `json:"host,omitempty"`   // TCP hostname
	Port     *string `json:"port,omitempty"`   // TCP port
	Socket   *string `json:"socket,omitempty"` // Unix socket path
}

// IsEmpty is true if l is nil or none of the fields are set.
func (l *Login) IsEmpty() bool {
	return l == nil ||
		(l.User == nil &&
			l.Password == nil &&
			l.Host == nil &&
			l.Port == nil &&
			l.Socket == nil)
}

// DSN builds a DSN for github.com/go-sql-driver/mysql
//
// The DSN returned always has a '/' at the end.
// The DSN for an empty Login is just "/".
func (l *Login) DSN() string {
	// Handles the case where login is nil
	if l.IsEmpty() {
		return "/"
	}

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
		} else {
			port = "3306" // MySQL default port
		}
		b.WriteString("tcp(")
		b.WriteString(net.JoinHostPort(host, port))
		b.WriteByte(')')
	}

	// The separator with the database name
	b.WriteByte('/')

	return b.String()
}

// String returns DSN().
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

func (l *Login) parseLine(line string) error {
	s := strings.SplitN(line, " = ", 2)

	s[1] = unescape(s[1])

	switch s[0] {
	case "user":
		l.User = &s[1]
	case "password":
		l.Password = &s[1]
	case "host":
		l.Host = &s[1]
	case "port":
		l.Port = &s[1]
	case "socket":
		l.Socket = &s[1]
	default:
		return fmt.Errorf("Unknown option '%s'", s[0])
	}
	return nil
}

// Merge merges l into login: options set in l take precedence over
// options set in login.
func (l *Login) Merge(other *Login) {
	if other.User != nil {
		l.User = other.User
	}
	if other.Password != nil {
		l.Password = other.Password
	}
	if other.Host != nil {
		l.Host = other.Host
	}
	if other.Port != nil {
		l.Port = other.Port
	}
	if other.Socket != nil {
		l.Socket = other.Socket
	}
}
