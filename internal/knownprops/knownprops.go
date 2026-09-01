// Package knownprops adds Elsevier-specific input validation for custom
// properties whose format GitHub's schema can't express. GitHub only knows
// a property is "string" — not that it should look like an email address,
// a date, or an alphanumeric code. The single_select properties (MigrationReady,
// SystemType, TargetOrg, TechOrg, TechOrgGroup, ...) don't need anything
// here: their allowed values already come live and authoritatively from the
// org's schema (internal/ghclient.GetOrgSchema), so duplicating them here
// would just be a second, staler copy to keep in sync.
package knownprops

import (
	"fmt"
	"net/mail"
	"regexp"
	"time"
)

// Kind identifies the input format a string-typed property should be
// validated against.
type Kind string

const (
	Alphanumeric Kind = "alphanumeric"
	Email        Kind = "email"
	Date         Kind = "date" // YYYY-MM-DD
)

// Validators maps Elsevier's standard custom-property names to the format
// their value must satisfy.
var Validators = map[string]Kind{
	"CostCode":           Alphanumeric,
	"SystemID":           Alphanumeric,
	"SystemName":         Alphanumeric,
	"owner":              Email,
	"MigrationReadyDate": Date,
}

var alphanumericPattern = regexp.MustCompile(`^[A-Za-z0-9]+$`)

// Validate reports whether value satisfies kind, returning a human-readable
// error describing the expected format when it doesn't.
func Validate(kind Kind, value string) error {
	switch kind {
	case Alphanumeric:
		if !alphanumericPattern.MatchString(value) {
			return fmt.Errorf("must be alphanumeric (letters and digits only)")
		}
	case Email:
		if _, err := mail.ParseAddress(value); err != nil {
			return fmt.Errorf("must be a valid email address")
		}
	case Date:
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return fmt.Errorf("must be a date in YYYY-MM-DD format")
		}
	}
	return nil
}
