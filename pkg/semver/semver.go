// Package semver provides semantic version parsing and constraint matching
// for the Carv package manager.
//
// Supported version formats: "1.0.0", "1.2.3-beta.1", "0.1.0+build.42"
// Supported constraint operators: =, !=, >, <, >=, <=, ^, ~, *
//
// Caret (^) ranges: ^1.2.3 := >=1.2.3, <2.0.0
// Tilde (~) ranges: ~1.2.3 := >=1.2.3, <1.3.0
// Wildcard:         *      := any version
package semver

import (
	"fmt"
	"strconv"
	"strings"
)

// Version represents a parsed semantic version.
type Version struct {
	Major      uint64
	Minor      uint64
	Patch      uint64
	PreRelease string
	Build      string
	raw        string
}

// String returns the version in "major.minor.patch[-pre][+build]" format.
func (v Version) String() string {
	if v.raw != "" {
		return v.raw
	}
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.PreRelease != "" {
		s += "-" + v.PreRelease
	}
	if v.Build != "" {
		s += "+" + v.Build
	}
	return s
}

// Compare returns -1, 0, or 1 if v < other, v == other, or v > other.
// Pre-release versions are compared per semver spec; build metadata is ignored.
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		return cmp64(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return cmp64(v.Minor, other.Minor)
	}
	if v.Patch != other.Patch {
		return cmp64(v.Patch, other.Patch)
	}
	return comparePreRelease(v.PreRelease, other.PreRelease)
}

func cmp64(a, b uint64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// comparePreRelease follows semver 2.0.0 spec:
// pre-release has lower precedence than release.
// Identifiers are compared left to right: numeric < string, numerically for numbers, lexically for strings.
func comparePreRelease(a, b string) int {
	if a == "" && b == "" {
		return 0
	}
	if a == "" {
		return 1 // release > pre-release
	}
	if b == "" {
		return -1
	}

	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		aNum, aErr := strconv.ParseUint(aParts[i], 10, 64)
		bNum, bErr := strconv.ParseUint(bParts[i], 10, 64)

		aIsNum := aErr == nil
		bIsNum := bErr == nil

		switch {
		case aIsNum && bIsNum:
			if aNum != bNum {
				return cmp64(aNum, bNum)
			}
		case aIsNum && !bIsNum:
			return -1 // numeric < string
		case !aIsNum && bIsNum:
			return 1
		default:
			if aParts[i] != bParts[i] {
				if aParts[i] < bParts[i] {
					return -1
				}
				return 1
			}
		}
	}

	return cmp64(uint64(len(aParts)), uint64(len(bParts)))
}

// ParseVersion parses a semver version string.
func ParseVersion(s string) (Version, error) {
	orig := s
	s = strings.TrimSpace(s)
	if s == "" {
		return Version{}, fmt.Errorf("empty version string")
	}

	// Strip leading 'v' or '='
	if len(s) > 0 && (s[0] == 'v' || s[0] == 'V') {
		s = s[1:]
	}

	var build string
	if idx := strings.Index(s, "+"); idx >= 0 {
		build = s[idx+1:]
		s = s[:idx]
	}

	var pre string
	if idx := strings.Index(s, "-"); idx >= 0 {
		pre = s[idx+1:]
		s = s[:idx]
	}

	parts := strings.Split(s, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return Version{}, fmt.Errorf("invalid version %q: expected 1-3 numeric parts", orig)
	}

	var major, minor, patch uint64
	var err error

	major, err = strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return Version{}, fmt.Errorf("invalid version %q: %w", orig, err)
	}

	if len(parts) > 1 {
		minor, err = strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return Version{}, fmt.Errorf("invalid version %q: %w", orig, err)
		}
	}

	if len(parts) > 2 {
		patch, err = strconv.ParseUint(parts[2], 10, 64)
		if err != nil {
			return Version{}, fmt.Errorf("invalid version %q: %w", orig, err)
		}
	}

	return Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		PreRelease: pre,
		Build:      build,
		raw:        orig,
	}, nil
}

// Op is a constraint comparison operator.
type Op int

const (
	OpEqual Op = iota
	OpNotEqual
	OpGreater
	OpLess
	OpGreaterEqual
	OpLessEqual
	OpCaret // ^
	OpTilde // ~
	OpAny   // *
)

func (o Op) String() string {
	switch o {
	case OpEqual:
		return "="
	case OpNotEqual:
		return "!="
	case OpGreater:
		return ">"
	case OpLess:
		return "<"
	case OpGreaterEqual:
		return ">="
	case OpLessEqual:
		return "<="
	case OpCaret:
		return "^"
	case OpTilde:
		return "~"
	case OpAny:
		return "*"
	default:
		return "?"
	}
}

// Constraint represents a single version constraint.
type Constraint struct {
	Op        Op
	Version   Version
	Precision int // number of version parts specified (1=major, 2=minor, 3=patch)
}

// ParseConstraint parses a version constraint string.
// Supports: "1.0.0", "^1.2.3", "~1.2.3", ">=1.0.0", "<2.0.0", "!=1.5.0", "*"
func ParseConstraint(s string) (Constraint, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "*" {
		return Constraint{Op: OpAny}, nil
	}

	var op Op
	switch {
	case strings.HasPrefix(s, "!="):
		op = OpNotEqual
		s = s[2:]
	case strings.HasPrefix(s, ">="):
		op = OpGreaterEqual
		s = s[2:]
	case strings.HasPrefix(s, "<="):
		op = OpLessEqual
		s = s[2:]
	case strings.HasPrefix(s, ">"):
		op = OpGreater
		s = s[1:]
	case strings.HasPrefix(s, "<"):
		op = OpLess
		s = s[1:]
	case strings.HasPrefix(s, "^"):
		op = OpCaret
		s = s[1:]
	case strings.HasPrefix(s, "~"):
		op = OpTilde
		s = s[1:]
	case strings.HasPrefix(s, "="):
		op = OpEqual
		s = s[1:]
	default:
		op = OpEqual
	}

	s = strings.TrimSpace(s)
	if s == "" || s == "*" {
		if op == OpEqual {
			return Constraint{Op: OpAny}, nil
		}
		return Constraint{}, fmt.Errorf("operator %q requires a version", op)
	}

	// Determine precision (number of version parts) before stripping pre/build.
	verStr := s
	if idx := strings.Index(verStr, "+"); idx >= 0 {
		verStr = verStr[:idx]
	}
	if idx := strings.Index(verStr, "-"); idx >= 0 {
		verStr = verStr[:idx]
	}
	prec := 1 + strings.Count(verStr, ".")

	ver, err := ParseVersion(s)
	if err != nil {
		return Constraint{}, err
	}

	return Constraint{Op: op, Version: ver, Precision: prec}, nil
}

// Matches checks if a version satisfies the constraint.
func (c Constraint) Matches(v Version) bool {
	switch c.Op {
	case OpAny:
		return true
	case OpEqual:
		return v.Compare(c.Version) == 0
	case OpNotEqual:
		return v.Compare(c.Version) != 0
	case OpGreater:
		return v.Compare(c.Version) > 0
	case OpLess:
		return v.Compare(c.Version) < 0
	case OpGreaterEqual:
		return v.Compare(c.Version) >= 0
	case OpLessEqual:
		return v.Compare(c.Version) <= 0
	case OpCaret:
		return matchesCaret(v, c.Version)
	case OpTilde:
		return matchesTilde(v, c.Version)
	default:
		return false
	}
}

// matchesCaret: ^X.Y.Z := >=X.Y.Z, <(X+1).0.0  (if X>0)
// ^0.Y.Z := >=0.Y.Z, <0.(Y+1).0  (if Y>0)
// ^0.0.Z := >=0.0.Z, <0.0.(Z+1)
func matchesCaret(v, base Version) bool {
	// Must be >= base
	if v.Compare(base) < 0 {
		return false
	}

	var upper Version
	if base.Major > 0 {
		upper = Version{Major: base.Major + 1}
	} else if base.Minor > 0 {
		upper = Version{Major: 0, Minor: base.Minor + 1}
	} else {
		upper = Version{Major: 0, Minor: 0, Patch: base.Patch + 1}
	}

	return v.Compare(upper) < 0
}

// matchesTilde: ~X.Y.Z := >=X.Y.Z, <X.(Y+1).0
// ~X.Y := >=X.Y.0, <X.(Y+1).0
// ~X := >=X.0.0, <(X+1).0.0
func matchesTilde(v, base Version) bool {
	if v.Compare(base) < 0 {
		return false
	}

	var upper Version
	// For ~X (precision 1), upper bound is (X+1).0.0
	// For ~X.Y or ~X.Y.Z (precision 2+), upper bound is X.(Y+1).0
	if base.Minor == 0 && base.Patch == 0 {
		upper = Version{Major: base.Major + 1}
	} else {
		upper = Version{Major: base.Major, Minor: base.Minor + 1}
	}
	return v.Compare(upper) < 0
}

// Constraints is a set of constraints that must ALL be satisfied (AND logic).
type Constraints []Constraint

// ParseConstraints parses a comma- or space-separated list of constraints.
// Examples: ">=1.0.0, <2.0.0", "^1.2.3", ">=1.0 <2.0"
func ParseConstraints(s string) (Constraints, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "*" {
		return Constraints{Constraint{Op: OpAny}}, nil
	}

	// Split on comma or space (handling multiple spaces)
	var parts []string
	for _, token := range strings.Fields(s) {
		token = strings.Trim(token, ",")
		if token == "" {
			continue
		}
		// If token contains comma inside, split further
		sub := strings.Split(token, ",")
		for _, subToken := range sub {
			subToken = strings.TrimSpace(subToken)
			if subToken != "" {
				parts = append(parts, subToken)
			}
		}
	}

	if len(parts) == 0 {
		return Constraints{Constraint{Op: OpAny}}, nil
	}

	var cs Constraints
	for _, p := range parts {
		c, err := ParseConstraint(p)
		if err != nil {
			return nil, fmt.Errorf("invalid constraint %q: %w", p, err)
		}
		cs = append(cs, c)
	}
	return cs, nil
}

// Matches checks if a version satisfies ALL constraints.
func (cs Constraints) Matches(v Version) bool {
	for _, c := range cs {
		if !c.Matches(v) {
			return false
		}
	}
	return true
}

// Satisfies checks if a version string satisfies a constraint string.
// Convenience wrapper: Satisfies("1.2.3", "^1.0.0") -> true
func Satisfies(versionStr, constraintStr string) (bool, error) {
	v, err := ParseVersion(versionStr)
	if err != nil {
		return false, err
	}
	cs, err := ParseConstraints(constraintStr)
	if err != nil {
		return false, err
	}
	return cs.Matches(v), nil
}
