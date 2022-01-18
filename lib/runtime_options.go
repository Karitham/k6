/*
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2018 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package lib

import (
	"fmt"
	"strings"

	"gopkg.in/guregu/null.v3"
)

// CompatibilityMode specifies the JS compatibility mode
// nolint:lll
//go:generate enumer -type=CompatibilityMode -transform=snake -trimprefix CompatibilityMode -output compatibility_mode_gen.go
type CompatibilityMode uint8

const (
	// CompatibilityModeExtended achieves ES6+ compatibility with Babel
	CompatibilityModeExtended CompatibilityMode = iota + 1
	// CompatibilityModeBase is standard goja ES5.1+
	CompatibilityModeBase
)

// RuntimeOptions are settings passed onto the goja JS runtime
type RuntimeOptions struct {
	Env                  map[string]string `json:"env"`
	CompatibilityMode    null.String       `json:"compatibilityMode"`
	SummaryExport        null.String       `json:"summaryExport"`
	IncludeSystemEnvVars null.Bool         `json:"includeSystemEnvVars"`
	NoThresholds         null.Bool         `json:"noThresholds"`
	NoSummary            null.Bool         `json:"noSummary"`
}

// ValidateCompatibilityMode checks if the provided val is a valid compatibility mode
func ValidateCompatibilityMode(val string) (cm CompatibilityMode, err error) {
	if val == "" {
		return CompatibilityModeExtended, nil
	}
	if cm, err = CompatibilityModeString(val); err != nil {
		var compatValues []string
		for _, v := range CompatibilityModeValues() {
			compatValues = append(compatValues, v.String())
		}
		err = fmt.Errorf(`invalid compatibility mode "%s". Use: "%s"`,
			val, strings.Join(compatValues, `", "`))
	}
	return
}
