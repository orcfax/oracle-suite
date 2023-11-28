/*
Package identity provides helper functions for returning information
about a system's version to help identify it as a node within the
Orcfax ecosystem.
*/
package identity

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/ipinfo/go/v2/ipinfo"

	"github.com/google/uuid"
)

/*
Identity is an object that provides as much information about the

hardware running this code.

	{
	  "node_id": "6bf28344-01e5-4aab-825b-846153fa6db5",
	  "location": {
	      // ipInfoSimpleSummary (below)
	  },
	  "initialization": "2023-11-23T08:04:29Z",
	  "init_version": null,,c
	  "validator_web_socket": null,
	  "validator_certificate": null
	}
*/
type Identity struct {
	NodeID             string              `json:"node_id"`
	Location           IPInfoSimpleSummary `json:"location"`
	InitializationDate string              `json:"initialization"`
	ValidatorWebSocket string              `json:"validator_web_socket"`
}

/*
IPInfoSimpleSummary helps us to construct a simplified object that
looks as follows:

	"location": {
		"ip": "170.64.189.192",
		"city": "Sydney",
		"region": "New South Wales",
		"country": "AU",
		"loc": "-33.9092,151.1940",
		"org": "AS14061 DigitalOcean, LLC",
		"postal": "2015",
		"timezone": "Australia/Sydney",
		"readme": "https://ipinfo.io/missingauth"
	}
*/
type IPInfoSimpleSummary struct {
	IP       string `json:"ip"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Location string `json:"loc"`
	Org      string `json:"org"`
	Postal   string `json:"postal"`
	Timezone string `json:"timezone"`
	Readme   string `json:"readme"`
}

// Exists returns true if the identity file exists, otherwise
// false.
func Exists(existingID string) bool {
	_, err := os.Stat(existingID)
	return !os.IsNotExist(err)
}

// LoadCache loads the cached summary from the default location on the
// filesystem. If it cannot find the identity we need to create a new
// one from scratch.
func LoadCache(existingID string) (Identity, error) {
	f, err := os.ReadFile(existingID)
	if err != nil {
		return Identity{}, err
	}
	var ident Identity
	err = json.Unmarshal(f, &ident)
	return ident, err
}

// IPInfoDefault returns the default core structure from IP Info.
func IPInfoDefault() *ipinfo.Core {
	client := ipinfo.DefaultClient
	core, err := client.GetIPInfo(nil)
	if err != nil {
		log.Println(err)
	}
	return core
}

// IPInfoSimple provides a simplified summary of the IP Info properties.
func IPInfoSimple() IPInfoSimpleSummary {
	const ipInfoURL string = "https://ipinfo.io/"
	summary := IPInfoSimpleSummary{}
	client := ipinfo.DefaultClient
	core, err := client.GetIPInfo(nil)
	if err != nil {
		log.Println(err)
	}

	summary.IP = core.IP.String()
	summary.City = core.City
	summary.Region = core.Region
	summary.Country = core.Country
	summary.Location = core.Location
	summary.Org = core.Org
	summary.Postal = core.Postal
	summary.Timezone = core.Timezone
	summary.Readme = ipInfoURL
	return summary
}

// GetIdentity is a convenience function to help construct an identity
// object.
func GetIdentity(nodeID string, websocket string) Identity {

	if nodeID == "" {
		uuidV4, _ := uuid.NewRandom()
		nodeID = uuidV4.String()
	}

	const utcTimeFormat = "2006-01-02T15:04:05Z"

	info := IPInfoSimple()
	ident := Identity{}
	ident.NodeID = nodeID
	ident.ValidatorWebSocket = websocket
	ident.Location = info
	ident.InitializationDate = time.Now().UTC().Format(utcTimeFormat)
	return ident
}
