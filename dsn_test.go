package mylogin_test

import (
	"github.com/dolmen-go/mylogin"

	"testing"

	"github.com/go-sql-driver/mysql"
)

func stringPtr(s string) *string {
	return &s
}

func TestDSN(t *testing.T) {
	l := mylogin.Login{
		User:     stringPtr("dolmen"),
		Password: stringPtr("secret"),
		Host:     stringPtr("localhost"),
		Port:     stringPtr("3306"),
	}
	dsn := l.DSN()
	t.Logf(dsn)
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	dsn2 := cfg.FormatDSN()
	if dsn2 != dsn {
		t.Fatal(dsn2, " != ", dsn)
	}
}
