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
	"testing"
)

// TestContentSignature makes sure that our content signature is
// created consistently for Orcfax v1's needs.
func TestContentSignature(t *testing.T) {
	timestamp := "2023-09-12T14:08:15Z"
	dataPoints := []string{
		"0.248848",
		"0.2489",
		"0.2488563207",
	}
	nodeID := "9165f28e-012e-4790-bf38-cce43184bc7d"
	contentSignature := createContentSignature(timestamp, dataPoints, nodeID)
	expected := "6dd329aaba26cf4d1175eafef13e8f49b41d2c36be6832987cb559bd715dcfd2"
	if contentSignature != expected {
		t.Errorf("content signature doesn't match: %s (expected: %s)", contentSignature, expected)
	}
}
