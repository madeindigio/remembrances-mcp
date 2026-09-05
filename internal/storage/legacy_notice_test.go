package storage

import (
	"os"
	"strings"
	"testing"
)

// TestTheLegacyLineNoticeReachesALegacyDeployment pins RM-10's notice to
// behaviour rather than to prose.
//
// The acceptance criterion is that a deployment still on the pre-principal line
// is told, in time, what stops working and what to do about it. A README says
// that to whoever opens the README; the warning below says it to whoever starts
// the process, which is the operator the deadline will actually surprise. The
// second half of the pin is that a migrated deployment is not nagged: a notice
// that is always printed is a notice nobody reads.
func TestTheLegacyLineNoticeReachesALegacyDeployment(t *testing.T) {
	restore := RequirePrincipal()
	t.Cleanup(func() { SetRequirePrincipal(restore) })

	SetRequirePrincipal(false)
	notice := LegacyLineNotice()
	if notice == "" {
		t.Fatal("a deployment on the legacy path was told nothing")
	}
	for _, want := range []string{LegacyLineEndOfSupport, "--require-principal", "GOMEM_REQUIRE_PRINCIPAL"} {
		if !strings.Contains(notice, want) {
			t.Errorf("the notice does not name %q: %s", want, notice)
		}
	}

	SetRequirePrincipal(true)
	if got := LegacyLineNotice(); got != "" {
		t.Errorf("a migrated deployment was still warned: %s", got)
	}
}

// TestTheEndOfSupportDateIsTheSameEverywhere keeps the running binary and the
// two documents from drifting apart. An operator who reads one date and is
// warned about another has been told nothing usable, and the date is exactly
// the kind of value that gets edited in one place.
func TestTheEndOfSupportDateIsTheSameEverywhere(t *testing.T) {
	for _, path := range []string{"../../README.md", "../../CHANGELOG.md"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		if !strings.Contains(string(raw), LegacyLineEndOfSupport) {
			t.Errorf("%s does not name the end-of-support date %q", path, LegacyLineEndOfSupport)
		}
	}
}
