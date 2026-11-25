package store

import "testing"

func TestNormalizeDriver(t *testing.T) {
	cases := map[string]string{
		"": "sqlite", "sqlite3": "sqlite", "MySQL": "mysql", "mariadb": "mysql",
		"postgres": "pgsql", "postgresql": "pgsql", "pg": "pgsql", "pgsql": "pgsql",
	}
	for in, want := range cases {
		if got := NormalizeDriver(in); got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
	if validDriver("oracle") {
		t.Fatal("oracle must be invalid")
	}
}

func TestIdentAndQuote(t *testing.T) {
	if Ident("ay_label") != true || Ident("ay-label") || Ident("1x") {
		t.Fatal("Ident")
	}
	d := &DB{Driver: "mysql"}
	q, err := d.QuoteIdent("ay_label")
	if err != nil || q != "`ay_label`" {
		t.Fatalf("mysql quote %q %v", q, err)
	}
	d.Driver = "pgsql"
	q, err = d.QuoteIdent("ay_label")
	if err != nil || q != `"ay_label"` {
		t.Fatalf("pgsql quote %q %v", q, err)
	}
	if _, err := d.QuoteIdent("bad-name"); err == nil {
		t.Fatal("expected bad ident error")
	}
}

func TestToPostgresCreate(t *testing.T) {
	in := "CREATE TABLE `ay_x` (\n  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,\n  `name` varchar(50) NOT NULL,\n  PRIMARY KEY (`id`)\n) ENGINE=MyISAM DEFAULT CHARSET=utf8;"
	out := toPostgresCreate(in)
	if contains(out, "AUTO_INCREMENT") || contains(out, "ENGINE") || contains(out, "`") {
		t.Fatalf("bad pg ddl: %s", out)
	}
	if !contains(out, "SERIAL") || !contains(out, "TEXT") {
		t.Fatalf("missing pg types: %s", out)
	}
}

func TestRandomIntCast(t *testing.T) {
	d := &DB{Driver: "mysql"}
	if d.Random() != "RAND()" || d.IntCast("scode") != "CAST(scode AS SIGNED)" {
		t.Fatalf("mysql dialect %s %s", d.Random(), d.IntCast("scode"))
	}
	d.Driver = "pgsql"
	if d.Random() != "RANDOM()" || !contains(d.IntCast("scode"), "INTEGER") {
		t.Fatalf("pgsql dialect %s %s", d.Random(), d.IntCast("scode"))
	}
	d.Driver = "sqlite"
	if d.IntCast("scode") != "CAST(scode AS INTEGER)" {
		t.Fatal(d.IntCast("scode"))
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
