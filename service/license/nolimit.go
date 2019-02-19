// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build nolimit

package license

// DefaultLicense is an empty license with no restrictions.
var DefaultLicense = &License{Kind: LicenseFoss}

func Trial(string) *License         { return nil }
func Load(string) (*License, error) { return DefaultLicense, nil }
