package timestamp

import "testing"

func TestParse_Microseconds(t *testing.T) {
	if _, err := Parse("2026-04-07T05:53:28.348188Z"); err != nil {
		t.Errorf("expected ok; got %v", err)
	}
}

func TestParse_NoFraction(t *testing.T) {
	if _, err := Parse("2026-04-07T05:53:28Z"); err != nil {
		t.Errorf("expected ok; got %v", err)
	}
}

func TestParse_Invalid(t *testing.T) {
	if _, err := Parse("not a time"); err == nil {
		t.Errorf("expected error")
	}
}

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z", 0},
		{"2026-02-01T00:00:00Z", "2026-01-01T00:00:00Z", 1},
		{"2026-01-01T00:00:00Z", "2026-02-01T00:00:00Z", -1},
		{"2026-04-07T05:53:28.348188Z", "2026-04-07T05:53:28.348187Z", 1},
	}
	for _, c := range cases {
		got, err := Compare(c.a, c.b)
		if err != nil {
			t.Fatal(err)
		}
		if got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestCompare_InvalidFirst(t *testing.T) {
	if _, err := Compare("bad", "2026-01-01T00:00:00Z"); err == nil {
		t.Errorf("expected error")
	}
}

func TestCompare_InvalidSecond(t *testing.T) {
	if _, err := Compare("2026-01-01T00:00:00Z", "bad"); err == nil {
		t.Errorf("expected error")
	}
}
