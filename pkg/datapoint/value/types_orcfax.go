package value

import (
	"github.com/orcfax/node-id/pkg/identity"
)

// Orcfax types help us to marshal the data collected into a format
// that is understood by an Orcfax V1 validator.

// OrcfaxMessage wraps the Orcfax data structure.
type OrcfaxMessage struct {
	Message             OrcfaxCollectorData `json:"message"`
	NodeID              string              `json:"node_id"`
	ValidationTimestamp string              `json:"validation_timestamp"`
}

// OrcfaxCollectorData is the primary payload for an Orcfax message
// containing the "collected" and "normalized" data.
type OrcfaxCollectorData struct {
	Timestamp        string              `json:"timestamp"`
	Raw              []OrcfaxRaw         `json:"raw"`
	DataPoints       []string            `json:"data_points"`
	CalculatedValue  string              `json:"calculated_value"`
	Feed             string              `json:"feed"`
	Identity         identity.Identity   `json:"identity"`
	ContentSignature string              `json:"content_signature"`
	Errors           []map[string]string `json:"errors"`
}

// OrcfaxRaw provides a means of storing raw request/response data from
// price-pair sources.
type OrcfaxRaw struct {
	Response         map[string]any `json:"response"`
	RequestURL       string         `json:"request_url"`
	RequestTimestamp string         `json:"request_timestamp"`
	Collector        string         `json:"collector"`
	Error            string         `json:"error"`
}

// buildProperties stores information read from the linker flags
// associated with the binary created from this code. E.g.
//
//	   ```
//	   {
//		      -ldflags -s -w
//	       -X main.version=100.0.0-SNAPSHOT-057f3fc
//	       -X main.commit=057f3fc6318d1824148bf91de5ef674fe8b9a504
//	       -X main.date=2024-01-29T19:14:07Z
//	       -X main.builtBy=goreleaser
//	   }
//	   ```
type BuildProperties struct {
	Commit  string
	Version string
	Date    string
}

func (buildProps *BuildProperties) Initialize() {
	buildProps.Commit = "0000000000000000000000000000000000000000"
	buildProps.Version = "dev-0.0.0"
	buildProps.Date = "1970-01-01T00:00:00Z"
}
