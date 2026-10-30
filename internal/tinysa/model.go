// SPDX-License-Identifier: GPL-3.0-or-later

// Package tinysa implements the tinySA sweep protocol on top of the serial
// client: device model detection, scanraw streaming and frame parsing.
package tinysa

import "strings"

// Frequency limits (Hz).
const (
	MinFreq         = 100e3
	MaxFreqBasic    = 960e6
	MaxFreqUltra    = 12e9  // hard cap; above 7.3 GHz is uncalibrated harmonic mode
	CalibratedUltra = 7.3e9 // level-calibrated range of the Ultra+ ZS407
)

// Model describes the connected device.
type Model struct {
	Ultra    bool    `json:"ultra"`
	Scale    float64 `json:"scale"`
	MaxFreq  float64 `json:"maxFreq"`
	Firmware string  `json:"firmware"`
}

// Detect derives the model from the `version` response, e.g.
// "tinySA4_v1.4-252-g2b4aa51" (Ultra/Ultra+) or "tinySA_v1.4-175-g1419a93".
func Detect(version string) Model {
	v := strings.TrimSpace(version)
	m := Model{Firmware: v, Scale: 128, MaxFreq: MaxFreqBasic}
	if strings.Contains(strings.ToLower(v), "tinysa4") {
		m.Ultra = true
		m.Scale = 174
		m.MaxFreq = MaxFreqUltra
	}
	return m
}
