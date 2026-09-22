package secret

import (
	"fmt"
	"regexp"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
)

// MaxNameLength is the longest secret name the platform accepts.
const MaxNameLength = 128

// namePattern is the platform's rule for secret names. Names are also
// lower-cased before storage, and the "sec_" prefix is reserved for identifiers.
var namePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ValidateName reports whether name is usable as a secret name, returning an
// *errdefs.InvalidArgumentError describing the rule it broke.
//
// Checking locally turns what would be an opaque 400 into an error that names
// the problem. It also matters for Fill: a name containing "}" would close the
// placeholder early and silently reference a different secret.
func ValidateName(name string) error {
	if name == "" {
		return invalidName(name, "a secret name must not be empty")
	}
	if len(name) > MaxNameLength {
		return invalidName(name, fmt.Sprintf("a secret name must be at most %d bytes", MaxNameLength))
	}
	if !namePattern.MatchString(name) {
		return invalidName(name, "a secret name may contain only letters, digits, underscores and hyphens")
	}
	return nil
}

func invalidName(name, reason string) error {
	return &errdefs.InvalidArgumentError{SandboxError: errdefs.SandboxError{
		Message: fmt.Sprintf("secret name %q is not usable: %s", name, reason),
	}}
}

// Fill returns the placeholder that resolves to the named secret's value.
//
// This is local string formatting and makes no network call: it does not check
// that the secret exists. A reference to an unknown secret fails inside the
// platform when the placeholder is resolved.
//
// Put the result wherever a sandbox should see the value, typically an
// environment variable:
//
//	ref, err := secret.Fill("openai-key")   // "${e2b.secrets.openai-key}"
func Fill(name string) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}
	return "${e2b.secrets." + name + "}", nil
}

// MustFill is Fill for names known at compile time, panicking instead of
// returning an error so the result can be used directly in a composite literal.
//
//	EnvVars: map[string]string{
//	    "OPENAI_API_KEY": secret.MustFill("openai-key"),
//	}
func MustFill(name string) string {
	ref, err := Fill(name)
	if err != nil {
		panic(err)
	}
	return ref
}
