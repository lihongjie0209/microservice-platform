package systemtests

import (
	"slices"
	"testing"
)

func TestAuthorizationBootstrapArgsPassUserIDWithoutShell(t *testing.T) {
	t.Parallel()
	args, err := authorizationBootstrapArgs(" user-1 ")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(args[len(args)-2:], []string{"--user-id", "user-1"}) {
		t.Fatalf("args = %q", args)
	}
	for _, invalid := range []string{"", " ", "--help"} {
		if _, err := authorizationBootstrapArgs(invalid); err == nil {
			t.Fatalf("user id %q accepted", invalid)
		}
	}
}
