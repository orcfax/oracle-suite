package value

import (
	"github.com/chronicleprotocol/oracle-suite/internal/identity"
	"github.com/chronicleprotocol/oracle-suite/pkg/util/bn"
)

// Value is a data point value.
//
// A value can be anything, e.g. a number, a string, a struct, etc.
//
// The interface must be implemented by using non-pointer receivers.
type Value interface {
	// Print returns a human-readable representation of the value.
	Print() string
}

// ValidatableValue is a data point value which can be validated.
//
// The interface must be implemented by using non-pointer receivers.
type ValidatableValue interface {
	Validate() error
}

// NumericValue is a data point value which is a number.
//
// The interface must be implemented by using non-pointer receivers.
type NumericValue interface {
	Number() *bn.FloatNumber
}

// Orcfax types help us to marshal the data collected into a format
// that is understood by an Orcfax V1 validator.

// OrcfaxMessage wraps the Orcfax data structure.
type OrcfaxMessage struct {
	Message OrcfaxCollectorData `json:"message"`
}

// OrcfaxCollectorData is the primary payload for an Orcfax message
// containing the "collected" and "normalized" data.
type OrcfaxCollectorData struct {
	Timestamp        string            `json:"timestamp"`
	Raw              []OrcfaxRaw       `json:"raw"`
	DataPoints       []string          `json:"data_points"`
	CalculatedValue  string            `json:"calculated_value"`
	Feed             string            `json:"feed"`
	Identity         identity.Identity `json:"identity"`
	ContentSignature string            `json:"content_signature"`
	Errors           []string          `json:"errors"`
}

// OrcfaxRaw provides a means of storing raw request/response data from
// price-pair sources.
type OrcfaxRaw struct {
	Response         string `json:"response"`
	RequestURL       string `json:"request_url"`
	RequestTimestamp string `json:"request_timestamp"`
	Collector        string `json:"collector"`
	Error            string `json:"error"`
}

// OrcfaxCollectorIdentity provides a means of identifying the collector
// node.
type OrcfaxCollectorIdentity struct {
	NodeID   string            `json:"node_id"`
	Location CollectorLocation `json:"location"`
}

// CollectorLocation is a more granular breakdown of the information
// available about a collecting node.
type CollectorLocation struct {
	IP       string `json:"ip"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Loc      string `json:"loc"`
	Org      string `json:"org"`
	Postal   string `json:"postal"`
	Timezone string `json:"timezone"`
	Readme   string `json:"readme"`
}
