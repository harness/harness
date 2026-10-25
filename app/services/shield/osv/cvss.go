// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package osv

import (
	"fmt"
	"math"
	"strings"
)

// CVSSV3BaseScore computes the CVSS v3.0/v3.1 base score of a vector such
// as "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H". Temporal and
// environmental metrics are ignored. CVSS v4 is not supported.
func CVSSV3BaseScore(vector string) (float64, error) {
	parts := strings.Split(strings.TrimSpace(vector), "/")
	if len(parts) < 9 || (parts[0] != "CVSS:3.1" && parts[0] != "CVSS:3.0") {
		return 0, fmt.Errorf("not a CVSS v3 vector: %q", vector)
	}
	m := map[string]string{}
	for _, p := range parts[1:] {
		k, v, ok := strings.Cut(p, ":")
		if !ok {
			return 0, fmt.Errorf("bad CVSS metric %q", p)
		}
		m[k] = v
	}
	scopeChanged := m["S"] == "C"
	if m["S"] != "C" && m["S"] != "U" {
		return 0, fmt.Errorf("bad CVSS scope in %q", vector)
	}
	av, ok1 := map[string]float64{"N": 0.85, "A": 0.62, "L": 0.55, "P": 0.2}[m["AV"]]
	ac, ok2 := map[string]float64{"L": 0.77, "H": 0.44}[m["AC"]]
	ui, ok3 := map[string]float64{"N": 0.85, "R": 0.62}[m["UI"]]
	prTab := map[string]float64{"N": 0.85, "L": 0.62, "H": 0.27}
	if scopeChanged {
		prTab = map[string]float64{"N": 0.85, "L": 0.68, "H": 0.5}
	}
	pr, ok4 := prTab[m["PR"]]
	cia := map[string]float64{"H": 0.56, "L": 0.22, "N": 0}
	c, ok5 := cia[m["C"]]
	i, ok6 := cia[m["I"]]
	a, ok7 := cia[m["A"]]
	if !(ok1 && ok2 && ok3 && ok4 && ok5 && ok6 && ok7) {
		return 0, fmt.Errorf("bad CVSS metrics in %q", vector)
	}
	iss := 1 - (1-c)*(1-i)*(1-a)
	var impact float64
	if scopeChanged {
		impact = 7.52*(iss-0.029) - 3.25*math.Pow(iss-0.02, 15)
	} else {
		impact = 6.42 * iss
	}
	if impact <= 0 {
		return 0, nil
	}
	expl := 8.22 * av * ac * pr * ui
	round := roundUp31
	if parts[0] == "CVSS:3.0" {
		round = func(x float64) float64 { return math.Ceil(x*10) / 10 }
	}
	if scopeChanged {
		return round(math.Min(1.08*(impact+expl), 10)), nil
	}
	return round(math.Min(impact+expl, 10)), nil
}

func roundUp31(x float64) float64 {
	n := int(math.Round(x * 100000))
	if n%10000 == 0 {
		return float64(n) / 100000
	}
	return (math.Floor(float64(n)/10000) + 1) / 10
}

// SeverityLabel maps a CVSS base score to its qualitative rating.
func SeverityLabel(score float64) string {
	switch {
	case score <= 0:
		return "NONE"
	case score < 4:
		return "LOW"
	case score < 7:
		return "MEDIUM"
	case score < 9:
		return "HIGH"
	}
	return "CRITICAL"
}
