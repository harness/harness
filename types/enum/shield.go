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

package enum

// ShieldScanTrigger defines what triggered a Shield scan.
type ShieldScanTrigger string

// ShieldScanTrigger enumeration.
const (
	ShieldScanTriggerPush     ShieldScanTrigger = "push"
	ShieldScanTriggerPullReq  ShieldScanTrigger = "pullreq"
	ShieldScanTriggerSchedule ShieldScanTrigger = "schedule"
	ShieldScanTriggerManual   ShieldScanTrigger = "manual"
)

var shieldScanTriggers = sortEnum([]ShieldScanTrigger{
	ShieldScanTriggerPush,
	ShieldScanTriggerPullReq,
	ShieldScanTriggerSchedule,
	ShieldScanTriggerManual,
})

func (ShieldScanTrigger) Enum() []any { return toInterfaceSlice(shieldScanTriggers) }
func (t ShieldScanTrigger) Sanitize() (ShieldScanTrigger, bool) {
	return Sanitize(t, GetAllShieldScanTriggers)
}
func GetAllShieldScanTriggers() ([]ShieldScanTrigger, ShieldScanTrigger) {
	return shieldScanTriggers, ShieldScanTriggerManual
}

// ShieldScanner defines a Shield scanner.
type ShieldScanner string

// ShieldScanner enumeration.
const (
	ShieldScannerOSV ShieldScanner = "osv"
)

var shieldScanners = sortEnum([]ShieldScanner{ShieldScannerOSV})

func (ShieldScanner) Enum() []any { return toInterfaceSlice(shieldScanners) }
func (s ShieldScanner) Sanitize() (ShieldScanner, bool) {
	return Sanitize(s, GetAllShieldScanners)
}
func GetAllShieldScanners() ([]ShieldScanner, ShieldScanner) {
	return shieldScanners, ShieldScannerOSV
}

// ShieldScanStatus defines the status of a Shield scan.
type ShieldScanStatus string

// ShieldScanStatus enumeration.
const (
	ShieldScanStatusPending ShieldScanStatus = "pending"
	ShieldScanStatusRunning ShieldScanStatus = "running"
	ShieldScanStatusDone    ShieldScanStatus = "done"
	ShieldScanStatusFailed  ShieldScanStatus = "failed"
)

var shieldScanStatuses = sortEnum([]ShieldScanStatus{
	ShieldScanStatusPending,
	ShieldScanStatusRunning,
	ShieldScanStatusDone,
	ShieldScanStatusFailed,
})

func (ShieldScanStatus) Enum() []any { return toInterfaceSlice(shieldScanStatuses) }
func (s ShieldScanStatus) Sanitize() (ShieldScanStatus, bool) {
	return Sanitize(s, GetAllShieldScanStatuses)
}
func GetAllShieldScanStatuses() ([]ShieldScanStatus, ShieldScanStatus) {
	return shieldScanStatuses, ShieldScanStatusPending
}

// ShieldFindingKind defines the kind of a Shield finding.
type ShieldFindingKind string

// ShieldFindingKind enumeration.
const (
	ShieldFindingKindDependency ShieldFindingKind = "dependency"
	ShieldFindingKindSAST       ShieldFindingKind = "sast"
	ShieldFindingKindSecret     ShieldFindingKind = "secret"
	ShieldFindingKindImage      ShieldFindingKind = "image"
	ShieldFindingKindIaC        ShieldFindingKind = "iac"
)

var shieldFindingKinds = sortEnum([]ShieldFindingKind{
	ShieldFindingKindDependency,
	ShieldFindingKindSAST,
	ShieldFindingKindSecret,
	ShieldFindingKindImage,
	ShieldFindingKindIaC,
})

func (ShieldFindingKind) Enum() []any { return toInterfaceSlice(shieldFindingKinds) }
func (k ShieldFindingKind) Sanitize() (ShieldFindingKind, bool) {
	return Sanitize(k, GetAllShieldFindingKinds)
}
func GetAllShieldFindingKinds() ([]ShieldFindingKind, ShieldFindingKind) {
	return shieldFindingKinds, ShieldFindingKindDependency
}

// ShieldFindingState defines the lifecycle state of a Shield finding.
type ShieldFindingState string

// ShieldFindingState enumeration.
const (
	ShieldFindingStateOpen         ShieldFindingState = "open"
	ShieldFindingStateFixed        ShieldFindingState = "fixed"
	ShieldFindingStateIgnored      ShieldFindingState = "ignored"
	ShieldFindingStateAcceptedRisk ShieldFindingState = "accepted_risk"
)

var shieldFindingStates = sortEnum([]ShieldFindingState{
	ShieldFindingStateOpen,
	ShieldFindingStateFixed,
	ShieldFindingStateIgnored,
	ShieldFindingStateAcceptedRisk,
})

func (ShieldFindingState) Enum() []any { return toInterfaceSlice(shieldFindingStates) }
func (s ShieldFindingState) Sanitize() (ShieldFindingState, bool) {
	return Sanitize(s, GetAllShieldFindingStates)
}
func GetAllShieldFindingStates() ([]ShieldFindingState, ShieldFindingState) {
	return shieldFindingStates, ShieldFindingStateOpen
}

// ShieldReachability defines whether a vulnerable dependency is reachable from the code.
type ShieldReachability string

// ShieldReachability enumeration.
const (
	ShieldReachabilityUnknown      ShieldReachability = "unknown"
	ShieldReachabilityReachable    ShieldReachability = "reachable"
	ShieldReachabilityNotReachable ShieldReachability = "not_reachable"
)

var shieldReachabilities = sortEnum([]ShieldReachability{
	ShieldReachabilityUnknown,
	ShieldReachabilityReachable,
	ShieldReachabilityNotReachable,
})

func (ShieldReachability) Enum() []any { return toInterfaceSlice(shieldReachabilities) }
func (r ShieldReachability) Sanitize() (ShieldReachability, bool) {
	return Sanitize(r, GetAllShieldReachabilities)
}
func GetAllShieldReachabilities() ([]ShieldReachability, ShieldReachability) {
	return shieldReachabilities, ShieldReachabilityUnknown
}

// ShieldFixStrategy defines how a finding is expected to be fixed.
type ShieldFixStrategy string

// ShieldFixStrategy enumeration.
const (
	ShieldFixStrategyBump    ShieldFixStrategy = "bump"
	ShieldFixStrategyPatch   ShieldFixStrategy = "patch"
	ShieldFixStrategyReplace ShieldFixStrategy = "replace"
	ShieldFixStrategyManual  ShieldFixStrategy = "manual"
	ShieldFixStrategyNoFix   ShieldFixStrategy = "no_fix"
)

var shieldFixStrategies = sortEnum([]ShieldFixStrategy{
	ShieldFixStrategyBump,
	ShieldFixStrategyPatch,
	ShieldFixStrategyReplace,
	ShieldFixStrategyManual,
	ShieldFixStrategyNoFix,
})

func (ShieldFixStrategy) Enum() []any { return toInterfaceSlice(shieldFixStrategies) }
func (s ShieldFixStrategy) Sanitize() (ShieldFixStrategy, bool) {
	return Sanitize(s, GetAllShieldFixStrategies)
}
func GetAllShieldFixStrategies() ([]ShieldFixStrategy, ShieldFixStrategy) {
	return shieldFixStrategies, ShieldFixStrategyBump
}
