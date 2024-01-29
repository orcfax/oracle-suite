package datapoint

import (
	"testing"

	"runtime/debug"

	"github.com/chronicleprotocol/oracle-suite/pkg/datapoint/value"
)

func TestReadBuildProperties(t *testing.T) {
	// {-ldflags }
	ldFlags := debug.BuildSetting{"-ldflags", "-s -w -X main.version=100.0.0-SNAPSHOT-057f3fc -X main.commit=057f3fc6318d1824148bf91de5ef674fe8b9a504 -X main.date=2024-01-29T19:14:07Z -X main.builtBy=goreleaser"}
	res := parseBuildProperties([]debug.BuildSetting{ldFlags})
	expected := value.BuildProperties{
		"057f3fc6318d1824148bf91de5ef674fe8b9a504",
		"100.0.0-SNAPSHOT-057f3fc",
		"2024-01-29T19:14:07Z",
	}
	if res != expected {
		t.Errorf(
			"build settings incorrectly parsed: '%s' expected '%s'",
			res, expected,
		)
	}
}
