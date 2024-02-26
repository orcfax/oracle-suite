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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	lg "log"

	"github.com/orcfax/node-id/pkg/identity"

	"github.com/orcfax/oracle-suite/pkg/datapoint/value"
)

const utcTimeFormat = "2006-01-02T15:04:05Z"

// generateNewIdentity allows us to initialize an identity on this
// collector when one doesn't already exist.
func generateNewIdentity(idLoc string) {
	lg.Printf("creating a new identity: '%s'", idLoc)
	ident := identity.GetIdentity("", "", "ws://")
	identJS, _ := json.MarshalIndent(ident, "", "   ")
	err := os.WriteFile(idLoc, identJS, 0644)
	if err != nil {
		lg.Printf(
			"identity hasn't been created, ensure '%s' can be written to, to prevent ipinfo API request volume failures",
			idLoc,
		)
	}
}

// readAndAttachIdentity will read the local node-id file and attach it
// to the collected record. If the identity hasn't been created yet then
// it is created for reuse later on.
func readAndAttachIdentity() identity.Identity {
	tmp := os.TempDir()
	idLoc := filepath.Join(tmp, ".node-identity.json")

	// Load the identity to enable it to be updated.
	ident, err := identity.LoadCache(idLoc)
	if err != nil {
		// In future implementations we need the handshake to
		// determine whether or not a new identity can just be
		// created.I.e.
		lg.Printf("error loading existing identity: '%s' cannot retrieve previous data", err)
		generateNewIdentity(idLoc)
	}

	ident, _ = identity.LoadCache(idLoc)
	lg.Printf("retrieved id: '%s'", ident.NodeID)
	lg.Printf("first initialized: '%s'", ident.InitializationDate)

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
//	   E.g.
//	   ```
//	   {
//		      -ldflags -s -w
//	       -X main.version=100.0.0-SNAPSHOT-057f3fc
//	       -X main.commit=057f3fc6318d1824148bf91de5ef674fe8b9a504
//	       -X main.date=2024-01-29T19:14:07Z
//	       -X main.builtBy=goreleaser
//	   }
//	   ```
func readBuildProperties() value.BuildProperties {
	buildProps, _ := debug.ReadBuildInfo()
	return parseBuildProperties(buildProps.Settings)
}

// parseBuildProperties handles the logic required to work through the
// build settings key-values and extract the values we need.
func parseBuildProperties(buildProperties []debug.BuildSetting) value.BuildProperties {
	buildProps := value.BuildProperties{}
	buildProps.Initialize()
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

// generateGlobalErrorStateMessage returns a blank message object.
func generateGlobalErrorStateMessage(point Point) (value.OrcfaxMessage, error) {
	collectorData := value.OrcfaxCollectorData{}
	msg := value.OrcfaxMessage{}
	errorEntry := makeError("GLOBAL", fmt.Sprintf("%s", point.Error))
	collectorData.Errors = append(collectorData.Errors, errorEntry)
	msg.Message = collectorData
	msg = generateMessageObject(collectorData)
	return msg, nil
}

func appendError(feedPair value.Pair, collectorData value.OrcfaxCollectorData, err string) value.OrcfaxCollectorData {
	collectorData.Errors = append(
		collectorData.Errors,
		makeError(feedPair.String(), err))
	return collectorData
}

// processExchangeData processes each individual exchange output
// formatting it so that it can be received by the Orcfax collector.
func processExchangeData(
	feedPair value.Pair,
	point Point,
	collectorData value.OrcfaxCollectorData) value.OrcfaxCollectorData {
	var dataPoints []string
	var rawData []value.OrcfaxRaw

	// We need to augment the exchange data information with the
	// executable's build properties so we read these here.
	buildProperties := readBuildProperties()

	for _, globalSubPoints := range point.SubPoints {
		for _, collectorSubPoint := range globalSubPoints.SubPoints {
			origin := collectorSubPoint.Meta["origin"]
			collector := collectorSubPoint.Meta["collector"]
			// Cast the subPoint to a value that can be processed more
			// granularly.
			subPointTick, ok := collectorSubPoint.Value.(value.Tick)
			if !ok {
				collectorData = appendError(feedPair, collectorData, fmt.Sprintf(
					"%s: error with type casting header, collector value cannot be parsed",
					origin,
				))
				// continue onto the next collector.
				continue
			}
			priceConverted, err := priceToString(subPointTick)
			if err != nil {
				collectorData = appendError(feedPair, collectorData, fmt.Sprintf("%s: %s", origin, err))
				// continue onto the next collector.
				continue
			}
			val, ok := collectorSubPoint.Meta["response"].([]byte)
			if !ok {
				collectorData = appendError(feedPair, collectorData, fmt.Sprintf(
					"%s: error with type casting original http responses",
					origin,
				))
				// continue onto the next collector.
				continue
			}

			// Build the raw output associated with the Orcfax message object.
			raw := value.OrcfaxRaw{}
			raw.Collector = fmt.Sprintf(
				"%s.%s.%s.%s",
				origin,
				collector,
				buildProperties.Version,
				buildProperties.Commit,
			)
			responseJSON := make(map[string]any)
			json.Unmarshal(val, &responseJSON)
			raw.Response = responseJSON
			raw.RequestURL = collectorSubPoint.Meta["request_url"].(string)
			raw.RequestTimestamp = collectorSubPoint.Time.UTC().Format(utcTimeFormat)
			rawData = append(rawData, raw)
			dataPoints = append(dataPoints, priceConverted)
		}
	}
	collectorData.DataPoints = dataPoints
	collectorData.Raw = rawData
	return collectorData
}

// generateCollectorObject creates the outline collector object
// that contains global information about what was returned by the
// collector as well as granular output form individual exchanges.
func generateCollectorObject(point Point) value.OrcfaxCollectorData {

	// capture global information.
	medianPrice := point.Value.(value.Tick)
	feedPair := medianPrice.Pair

	lg.Printf("processing feed pair: '%s'", feedPair)

	collectorData := value.OrcfaxCollectorData{}
	collectorData.Feed = strings.Replace(feedPair.String(), "/", "-", 1)
	calculated, _ := priceToString(medianPrice)
	collectorData.CalculatedValue = calculated
	collectorData.Timestamp = time.Now().UTC().Format(utcTimeFormat)

	collectorData = processExchangeData(feedPair, point, collectorData)

	// We perform this nearly last so that we have the granular output
	// above.
	collectorData = finalizeValidation(point, collectorData)
	return collectorData
}

// finalizeValidation performs Chronicle-labs own validation on the
// data here.
func finalizeValidation(point Point, collectorData value.OrcfaxCollectorData) value.OrcfaxCollectorData {
	// Chronicle-labs Oracle suite has its own concept of validation
	// that is performed on each data point before it arrives here.
	// Doubly it can also validate the global object which is its function
	// here to check for:
	//
	//   * global errors.
	//   * calculated value not being set.
	//   * time not being set.
	//
	// In the short-term there isn't a huge drawback having this final
	// check as we understand the suite. We might move our own validation
	// into the same function.
	//
	if err := point.Validate(); err != nil {
		collectorData.Errors = append(collectorData.Errors, (makeError("GLOBAL", err.Error())))
	}
	return collectorData
}

// MarshalOrcfax returns an Orcfax validator collector profile given
// the successful retrieval of price-pair data from the configured
// sources.
func (point Point) MarshalOrcfax() (value.OrcfaxMessage, error) {

	if point.Error != nil {
		// Global error across the collectors and so data cannot be
		// retrieved.
		return generateGlobalErrorStateMessage(point)
	}

	collectorData := generateCollectorObject(point)

	// Add the collectorData to the final Orcfax message object.
	msg := generateMessageObject(collectorData)

	return msg, nil
}
