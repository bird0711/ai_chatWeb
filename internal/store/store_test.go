package store

import (
	"context"
	"testing"
)

func TestSplitModels(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"a\nb\n", []string{"a", "b"}},
		{"a, b, c", []string{"a", "b", "c"}},
		{"a\r\nb, a", []string{"a", "b"}},
		{"\n\n", []string{}},
		{" model-1 , model-2\nmodel-3", []string{"model-1", "model-2", "model-3"}},
	}
	for _, c := range cases {
		got := splitModels(c.in)
		if len(got) != len(c.want) {
			t.Fatalf("splitModels(%q) = %#v, want %#v", c.in, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("splitModels(%q) = %#v, want %#v", c.in, got, c.want)
			}
		}
	}
}

func TestNullInt64(t *testing.T) {
	if v := nullInt64(0); v != nil {
		t.Fatalf("nullInt64(0) = %#v, want nil", v)
	}
	if v := nullInt64(-5); v != nil {
		t.Fatalf("nullInt64(-5) = %#v, want nil", v)
	}
	if v := nullInt64(7); v == nil {
		t.Fatalf("nullInt64(7) = nil, want non-nil")
	} else {
		if got, ok := v.(int64); !ok || got != 7 {
			t.Fatalf("nullInt64(7) = %#v (type %T), want int64(7)", v, v)
		}
	}
}

func TestValidateDatabaseName(t *testing.T) {
	good := []string{"testdb", "my_db1", "A1"}
	bad := []string{"", "inv@lid", "with-hyphen", "space name"}
	for _, s := range good {
		if err := validateDatabaseName(s); err != nil {
			t.Fatalf("validateDatabaseName(%q) returned error: %v", s, err)
		}
	}
	for _, s := range bad {
		if err := validateDatabaseName(s); err == nil {
			t.Fatalf("validateDatabaseName(%q) expected error, got nil", s)
		}
	}
}

func TestMySQLConfigStringHelpers(t *testing.T) {
	cfg := MySQLConfig{User: "u", Password: "p", Host: "h", Port: "3306", Database: "db"}
	dsn := cfg.DSN()
	if dsn == "" {
		t.Fatal("DSN should not be empty")
	}
	if cfg.AdminDSN() == "" {
		t.Fatal("AdminDSN should not be empty")
	}
	if cfg.SafeAddr() == "" {
		t.Fatal("SafeAddr should not be empty")
	}
	// MySQLDSN helper should match cfg.DSN()
	if MySQLDSN(cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database) != cfg.DSN() {
		t.Fatalf("MySQLDSN mismatch: %q vs %q", MySQLDSN(cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database), cfg.DSN())
	}
}

func TestOpenRedisInvalidAddr(t *testing.T) {
	ctx := context.Background()
	// use an invalid address to ensure OpenRedis returns an error instead of hanging
	if _, err := OpenRedis(ctx, "invalid:6379", "", 0); err == nil {
		t.Fatalf("OpenRedis(invalid:6379) expected error, got nil")
	}
}
