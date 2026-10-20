package devlokerr

import (
	"errors"
	"fmt"
	"testing"
)

func TestExitCodeMapping(t *testing.T) {
	cases := []struct {
		code Code
		want int
	}{
		{CodeInvalidManifest, ExitInvalid},
		{CodePortInUse, ExitPortInUse},
		{CodeElevationRequired, ExitElevation},
		{CodeDependencyMissing, ExitDependency},
		{CodeTimeout, ExitTimeout},
		{CodeCanceled, ExitCanceled},
		{CodeStateInconsistent, ExitStateInconsistent},
		{CodeInternal, ExitGeneral},
	}
	for _, tc := range cases {
		err := New(tc.code, "msg", "hint")
		if got := ExitCode(err); got != tc.want {
			t.Fatalf("%s: got %d want %d", tc.code, got, tc.want)
		}
	}
	if ExitCode(nil) != ExitOK {
		t.Fatal("nil should be ok")
	}
	if ExitCode(errors.New("plain")) != ExitGeneral {
		t.Fatal("plain error should be general")
	}
}

func TestWrapUnwrap(t *testing.T) {
	base := errors.New("disk")
	err := Wrap(CodePermissionDenied, base, "cannot write", "check perms")
	if !errors.Is(err, base) {
		t.Fatal("wrap should preserve cause")
	}
	e, ok := As(err)
	if !ok || e.Hint != "check perms" {
		t.Fatalf("unexpected error: %+v", e)
	}
	if got := err.Error(); got != fmt.Sprintf("%s: cannot write: disk", CodePermissionDenied) {
		t.Fatalf("message = %q", got)
	}
}
