//  Copyright (C) 2023 Orcfax Ltd.
//
//  This program is free software: you can redistribute it and/or modify
//  it under the terms of the GNU Affero General Public License as
//  published by the Free Software Foundation, either version 3 of the
//  License, or (at your option) any later version.
//
//  This program is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU Affero General Public License for more details.
//
//  You should have received a copy of the GNU Affero General Public License
//  along with this program.  If not, see <http://www.gnu.org/licenses/>.

package datapoint

import (
	"crypto/sha256"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	lg "log"

	"github.com/orcfax/node-id/pkg/identity"

	"github.com/chronicleprotocol/oracle-suite/pkg/datapoint/value"
)

const utcTimeFormat = "2006-01-02T15:04:05Z"

// readAndAttachIdentity will read the local node-id file and attach it
// to the collected record. If the identity hasn't been created yet then
// it is created for reuse later on.
func readAndAttachIdentity() identity.Identity {
	const nodeIdentityLocation = "/tmp/.node-identity.json"

	// Load the identity to enable it to be updated.
	ident, err := identity.LoadCache(nodeIdentityLocation)
	if err != nil {
		// In future implementations we need the handshake to
		// determine whether or not a new identity can just be
		// created.
		lg.Printf("error loading existing identity: '%s' cannot retrieve previous data", err)
	}

	if (ident == identity.Identity{}) {
		lg.Printf("creating a new id: %s", nodeIdentityLocation)
	} else {
		lg.Printf("retrieved id: '%s'", ident.NodeID)
		lg.Printf("first initialized: '%s'", ident.InitializationDate)
	}

	return ident
}

// createContentSignature provides a rudimentary way to create a
// checksum to validate a collector's message against. The message
// helps to prove the origin of a message and helps to protect against
// rounding errors using floating point numbers.
func createContentSignature(timestamp string, values []string, nodeID string) string {
	/* Reference impl:

	   ```golang
		   hash := sha256.New()
		   hash.Write([]byte("2023-09-12T14:08:15Z"))
		   hash.Write([]byte("0.248848"))
		   hash.Write([]byte("0.2489"))
		   hash.Write([]byte("0.2488563207"))
		   hash.Write([]byte("9165f28e-012e-4790-bf38-cce43184bc7d"))
		   hexHash := fmt.Sprintf("%x", bs)
		   hexHash == "6dd329aaba26cf4d1175eafef13e8f49b41d2c36be6832987cb559bd715dcfd2"
	   ```
	*/

	hash := sha256.New()
	hash.Write([]byte(timestamp))
	for _, value := range values {
		hash.Write([]byte(value))
	}
	hash.Write([]byte(nodeID))
	hashBytes := hash.Sum(nil)
	hexHash := fmt.Sprintf("%x", hashBytes)
	return string(hexHash)
}

// priceToString returns a string representation of a price value.
func priceToString(price value.Tick) (string, error) {
	if price.Price != nil {
		return price.Price.String(), nil
	}
	return "", fmt.Errorf("price is nil and cannot be converted")
}

// makeError creates a map representing an error.
func makeError(collector string, message string) map[string]string {
	m := make(map[string]string)
	m[collector] = message
	return m
}

// splitBuildProp provides a rudimentary helper to split build properties.
func splitBuildProp(property string) string {
	return strings.TrimSpace(strings.Split(property, "=")[1])
}

// readBuildProperties populates a buildProperties object so that it
// can be used to output information about this binary.
//
//    E.g.
//    ```
//    {
//	      -ldflags -s -w
//        -X main.version=100.0.0-SNAPSHOT-057f3fc
//        -X main.commit=057f3fc6318d1824148bf91de5ef674fe8b9a504
//        -X main.date=2024-01-29T19:14:07Z
//        -X main.builtBy=goreleaser
//    }
//    ```
//
//
func readBuildProperties() value.BuildProperties {
	buildProps, _ := debug.ReadBuildInfo()
	return parseBuildProperties(buildProps.Settings)
}

func parseBuildProperties(buildProperties []debug.BuildSetting) value.BuildProperties {
	buildProps := value.BuildProperties{}
	for _, settings := range buildProperties {
		if settings.Key != "-ldflags" {
			continue
		}
		setting := strings.Split(settings.Value, "-X")
		for _, property := range setting {
			if strings.Contains(property, "main.version") {
				buildProps.Version = splitBuildProp(property)
			}
			if strings.Contains(property, "main.commit") {
				buildProps.Commit = splitBuildProp(property)
			}
			if strings.Contains(property, "main.date") {
				buildProps.Date = splitBuildProp(property)
			}
		}
	}
	return buildProps
}

// generateMessageObject provides a helper function to generate the
// remainder of the Orcfax message at whatever point in the validation
// the procedure ends.
func generateMessageObject(collectorData value.OrcfaxCollectorData) value.OrcfaxMessage {
	nodeIdentity := readAndAttachIdentity()
	collectorData.ContentSignature = createContentSignature(
		collectorData.Timestamp,
		collectorData.DataPoints,
		nodeIdentity.NodeID,
	)
	msg := value.OrcfaxMessage{}
	msg.Message = collectorData
	msg.Message.Identity = nodeIdentity
	msg.NodeID = nodeIdentity.NodeID
	msg.ValidationTimestamp = time.Now().UTC().Format(utcTimeFormat)
	return msg
}

// MarshalOrcfax returns an Orcfax validator collector profile,
func (p Point) MarshalOrcfax() (value.OrcfaxMessage, error) {

	if p.Error != nil {
		// p.Error acts like a global error and affects all the feeds.
		// we simply need to return here
		collectorData := value.OrcfaxCollectorData{}
		msg := value.OrcfaxMessage{}
		m := makeError("ALL", fmt.Sprintf("%s", p.Error))
		collectorData.Errors = append(collectorData.Errors, m)
		msg.Message = collectorData
		msg = generateMessageObject(collectorData)
		return msg, nil
	}

	medianPrice := p.Value.(value.Tick)
	feedPair := medianPrice.Pair
	collectorData := value.OrcfaxCollectorData{}
	calculated, _ := priceToString(medianPrice)
	collectorData.CalculatedValue = calculated
	collectorData.Timestamp = time.Now().UTC().Format(utcTimeFormat)
	var dataPoints []string
	var rawData []value.OrcfaxRaw

	buildProperties := readBuildProperties()

	for _, t := range p.SubPoints {
		for _, tt := range t.SubPoints {
			origin := tt.Meta["origin"]
			collector := tt.Meta["collector"]
			subPointTick, ok := tt.Value.(value.Tick)
			if !ok {
				collectorData.Errors = append(
					collectorData.Errors,
					makeError(feedPair.String(), fmt.Sprintf(
						"%s: error with type casting header",
						origin,
					)))
				// continue onto the next collector.
				continue
			}
			priceConverted, err := priceToString(subPointTick)
			if err != nil {
				collectorData.Errors = append(
					collectorData.Errors,
					makeError(
						feedPair.String(),
						fmt.Sprintf("%s: %s", origin, err),
					))
				// continue onto the next collector.
				continue
			}
			val, ok := tt.Meta["headers"].(string)
			if !ok {
				collectorData.Errors = append(
					collectorData.Errors,
					makeError(feedPair.String(), fmt.Sprintf(
						"%s: error with type casting header",
						origin,
					)),
				)
				// continue onto the next collector.
				continue
			}
			// continue to add to the raw data output.
			raw := value.OrcfaxRaw{}

			raw.Collector = fmt.Sprintf(
				"%s.%s.%s.%s",
				origin,
				collector,
				buildProperties.Version,
				buildProperties.Commit,
			)

			raw.Response = val
			raw.RequestURL = tt.Meta["request_url"].(string)
			raw.RequestTimestamp = tt.Time.UTC().Format(utcTimeFormat)
			rawData = append(rawData, raw)
			dataPoints = append(dataPoints, priceConverted)
		}
	}

	collectorData.DataPoints = dataPoints
	collectorData.Raw = rawData
	collectorData.Feed = strings.Replace(feedPair.String(), "/", "-", 1)

	// NB. This is the Chronicle Labs concept of validation. We need to
	// verify this and also augment it with ours.
	if err := p.Validate(); err != nil {
		collectorData.Errors = append(collectorData.Errors, (makeError("ALL", err.Error())))
	}

	msg := generateMessageObject(collectorData)
	return msg, nil
}
