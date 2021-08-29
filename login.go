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

var unquote = strings.NewReplacer(
	`\"`, `"`,
	`\\`, `\`,
).Replace

func (l *Login) parseLine(line string) error {
	// Reference code:
	// https://github.com/mysql/mysql-shell/blob/master/mysql-secret-store/login-path/login_path_helper.cc#L52

	s := strings.SplitN(line, " = ", 2)

	v := s[1]

	// mysql_config_editor quotes strings since 8.0.24
	// https://github.com/mysql/mysql-server/commit/7d8028ac99730d4ccbe42d6edc11cc4f6d0cddca#diff-f8995fe51ada555169245803572ae5bd33a1793f6c027a39f8475c9156068ee5L518
	// shcore::unquote_string: https://github.com/mysql/mysql-shell/blob/master/mysqlshdk/libs/utils/utils_string.cc#L225
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		v = unquote(v[1 : len(v)-1])
	} else {
		v = strings.ReplaceAll(v, `\\`, `\`)
	}

	v = unescape(v)

	switch s[0] {
	case "user":
		l.User = &v
	case "password":
		l.Password = &v
	case "host":
		l.Host = &v
	case "port":
		l.Port = &v
	case "socket":
		l.Socket = &v
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
