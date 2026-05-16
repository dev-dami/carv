package semver

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input   string
		want    Version
		wantErr bool
	}{
		{"1.0.0", Version{Major: 1, Minor: 0, Patch: 0, raw: "1.0.0"}, false},
		{"0.1.0", Version{Major: 0, Minor: 1, Patch: 0, raw: "0.1.0"}, false},
		{"1.2.3", Version{Major: 1, Minor: 2, Patch: 3, raw: "1.2.3"}, false},
		{"v1.2.3", Version{Major: 1, Minor: 2, Patch: 3, raw: "v1.2.3"}, false},
		{"1.2.3-beta.1", Version{Major: 1, Minor: 2, Patch: 3, PreRelease: "beta.1", raw: "1.2.3-beta.1"}, false},
		{"1.2.3+build.42", Version{Major: 1, Minor: 2, Patch: 3, Build: "build.42", raw: "1.2.3+build.42"}, false},
		{"1.2.3-rc.1+build", Version{Major: 1, Minor: 2, Patch: 3, PreRelease: "rc.1", Build: "build", raw: "1.2.3-rc.1+build"}, false},
		{"1.2", Version{Major: 1, Minor: 2, Patch: 0, raw: "1.2"}, false},
		{"1", Version{Major: 1, Minor: 0, Patch: 0, raw: "1"}, false},
		{"", Version{}, true},
		{"abc", Version{}, true},
		{"1.2.3.4", Version{}, true},
	}

	for _, tt := range tests {
		got, err := ParseVersion(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseVersion(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("ParseVersion(%q) = %+v, want %+v", tt.input, got, tt.want)
		}
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"1.0.0", "1.1.0", -1},
		{"1.1.0", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0-alpha", "1.0.0", -1},
		{"1.0.0", "1.0.0-alpha", 1},
		{"1.0.0-alpha", "1.0.0-beta", -1},
		{"1.0.0-beta.2", "1.0.0-beta.11", -1},
		{"1.0.0-rc.1", "1.0.0-rc.2", -1},
		{"1.0.0+build1", "1.0.0+build2", 0},
		{"1.0.0-1", "1.0.0-2", -1},
		{"1.0.0-2", "1.0.0-10", -1},
	}

	for _, tt := range tests {
		a := mustVer(tt.a)
		b := mustVer(tt.b)
		got := a.Compare(b)
		if got != tt.want {
			t.Errorf("%s vs %s: got %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestParseConstraint(t *testing.T) {
	tests := []struct {
		input   string
		op      Op
		ver     string
		wantErr bool
	}{
		{"1.0.0", OpEqual, "1.0.0", false},
		{"=1.0.0", OpEqual, "1.0.0", false},
		{"!=1.0.0", OpNotEqual, "1.0.0", false},
		{">1.0.0", OpGreater, "1.0.0", false},
		{"<1.0.0", OpLess, "1.0.0", false},
		{">=1.0.0", OpGreaterEqual, "1.0.0", false},
		{"<=1.0.0", OpLessEqual, "1.0.0", false},
		{"^1.2.3", OpCaret, "1.2.3", false},
		{"~1.2.3", OpTilde, "1.2.3", false},
		{"*", OpAny, "", false},
		{"", OpAny, "", false},
	}

	for _, tt := range tests {
		got, err := ParseConstraint(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseConstraint(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr {
			if got.Op != tt.op {
				t.Errorf("ParseConstraint(%q) op = %v, want %v", tt.input, got.Op, tt.op)
			}
			if tt.ver != "" && got.Version.String() != tt.ver {
				t.Errorf("ParseConstraint(%q) version = %s, want %s", tt.input, got.Version.String(), tt.ver)
			}
		}
	}
}

func TestConstraintMatches(t *testing.T) {
	tests := []struct {
		constraint string
		version    string
		want       bool
	}{
		// Exact
		{"1.0.0", "1.0.0", true},
		{"1.0.0", "1.0.1", false},
		{"=1.0.0", "1.0.0", true},

		// Not equal
		{"!=1.0.0", "1.0.0", false},
		{"!=1.0.0", "1.0.1", true},

		// Greater/Less
		{">1.0.0", "1.0.1", true},
		{">1.0.0", "1.0.0", false},
		{"<2.0.0", "1.9.9", true},
		{"<2.0.0", "2.0.0", false},
		{">=1.0.0", "1.0.0", true},
		{">=1.0.0", "0.9.9", false},
		{"<=1.0.0", "1.0.0", true},
		{"<=1.0.0", "1.0.1", false},

		// Caret
		{"^1.2.3", "1.2.3", true},
		{"^1.2.3", "1.5.0", true},
		{"^1.2.3", "1.9.9", true},
		{"^1.2.3", "2.0.0", false},
		{"^1.2.3", "1.2.2", false},
		{"^0.2.3", "0.2.3", true},
		{"^0.2.3", "0.3.0", false},
		{"^0.2.3", "0.2.5", true},
		{"^0.0.3", "0.0.3", true},
		{"^0.0.3", "0.0.4", false},

		// Tilde
		{"~1.2.3", "1.2.3", true},
		{"~1.2.3", "1.2.9", true},
		{"~1.2.3", "1.3.0", false},
		{"~1.2.3", "1.2.2", false},
		{"~1.2", "1.2.0", true},
		{"~1.2", "1.2.9", true},
		{"~1.2", "1.3.0", false},
		{"~1", "1.0.0", true},
		{"~1", "1.9.9", true},
		{"~1", "2.0.0", false},

		// Wildcard
		{"*", "0.0.0", true},
		{"*", "99.99.99", true},
	}

	for _, tt := range tests {
		c := mustCon(tt.constraint)
		v := mustVer(tt.version)
		got := c.Matches(v)
		if got != tt.want {
			t.Errorf("Constraint(%q).Matches(%q) = %v, want %v", tt.constraint, tt.version, got, tt.want)
		}
	}
}

func TestParseConstraints(t *testing.T) {
	tests := []struct {
		input   string
		version string
		want    bool
	}{
		{">=1.0.0, <2.0.0", "1.5.0", true},
		{">=1.0.0, <2.0.0", "2.0.0", false},
		{">=1.0.0, <2.0.0", "0.9.9", false},
		{">=1.0 <2.0", "1.5.0", true},
		{"^1.0.0", "1.5.0", true},
		{"^1.0.0", "2.0.0", false},
		{"*", "1.0.0", true},
		{">=1.0.0, !=1.5.0", "1.5.0", false},
		{">=1.0.0, !=1.5.0", "1.6.0", true},
	}

	for _, tt := range tests {
		cs, err := ParseConstraints(tt.input)
		if err != nil {
			t.Errorf("ParseConstraints(%q) error: %v", tt.input, err)
			continue
		}
		v := mustVer(tt.version)
		got := cs.Matches(v)
		if got != tt.want {
			t.Errorf("Constraints(%q).Matches(%q) = %v, want %v", tt.input, tt.version, got, tt.want)
		}
	}
}

func TestSatisfies(t *testing.T) {
	ok, err := Satisfies("1.2.3", "^1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected 1.2.3 to satisfy ^1.0.0")
	}

	ok, err = Satisfies("2.0.0", "^1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected 2.0.0 to NOT satisfy ^1.0.0")
	}
}

func mustVer(s string) Version {
	v, err := ParseVersion(s)
	if err != nil {
		panic("bad test version: " + s)
	}
	return v
}

func mustCon(s string) Constraint {
	c, err := ParseConstraint(s)
	if err != nil {
		panic("bad test constraint: " + s)
	}
	return c
}
