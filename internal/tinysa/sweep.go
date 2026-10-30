// SPDX-License-Identifier: GPL-3.0-or-later

package tinysa

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"tinysa_go/internal/serial"
)

// SweepConfig is one scanraw configuration.
type SweepConfig struct {
	Start  float64 `json:"start"`  // Hz
	Stop   float64 `json:"stop"`   // Hz
	Points int     `json:"points"` // 2..2000
	RBW    float64 `json:"rbw"`    // kHz, 0 = auto
	Atten  int     `json:"atten"`  // dB, -1 = auto
	LNA    bool    `json:"lna"`
	Spur   bool    `json:"spur"`
}

// Validate checks the configuration against the device limits.
func (c SweepConfig) Validate(m Model) error {
	if c.Points < 2 || c.Points > 2000 {
		return fmt.Errorf("points must be 2..2000, got %d", c.Points)
	}
	if c.Start < MinFreq {
		return fmt.Errorf("start must be >= %.0f kHz", MinFreq/1e3)
	}
	if c.Stop <= c.Start {
		return fmt.Errorf("stop must be greater than start")
	}
	if c.Stop > m.MaxFreq {
		return fmt.Errorf("stop %.3f MHz exceeds the %.3f MHz limit of this model",
			c.Stop/1e6, m.MaxFreq/1e6)
	}
	return nil
}

// Command returns the auto-repeating scanraw command (3 = repeat).
func (c SweepConfig) Command() string {
	return fmt.Sprintf("scanraw %d %d %d 3", int64(c.Start), int64(c.Stop), c.Points)
}

// Frame is one completed sweep with accumulated traces and level statistics.
type Frame struct {
	Seq      uint64    `json:"seq"`
	Start    float64   `json:"start"`
	Stop     float64   `json:"stop"`
	Points   int       `json:"points"`
	DBM      []float32 `json:"dbm"`
	Max      []float32 `json:"max,omitempty"`
	Min      []float32 `json:"min,omitempty"`
	Avg      []float32 `json:"avg,omitempty"`
	Noise    float32   `json:"noise"`    // median noise floor, dBm
	Peak     float32   `json:"peak"`     // strongest bin, dBm
	PeakFreq float64   `json:"peakFreq"` // strongest bin, Hz
	SNR      float32   `json:"snr"`      // Peak − Noise, dB
}

// NoiseFloor returns the median level of a sweep. The median is robust against
// narrow-band signals, unlike the mean.
func NoiseFloor(dbm []float32) float32 {
	if len(dbm) == 0 {
		return 0
	}
	buf := make([]float32, len(dbm))
	copy(buf, dbm)
	slices.Sort(buf)
	return buf[len(buf)/2]
}

// PeakStats returns the strongest bin level and its frequency.
func PeakStats(dbm []float32, start, stop float64) (level float32, freq float64) {
	if len(dbm) == 0 {
		return 0, start
	}
	idx := 0
	for i, v := range dbm {
		if v > dbm[idx] {
			idx = i
		}
	}
	level = dbm[idx]
	if len(dbm) > 1 {
		freq = start + (stop-start)*float64(idx)/float64(len(dbm)-1)
	} else {
		freq = start
	}
	return level, freq
}

// CSV renders the frame as a UTF-8 CSV table with one row per point. The same
// format is used by the desktop "Save CSV" dialog and the web UI download.
func (f *Frame) CSV() []byte {
	var sb strings.Builder
	sb.Grow(f.Points * 56)
	sb.WriteString("Frequency_MHz,Live_dBm,Max_dBm,Avg_dBm,Min_dBm,SNR_dB\n")
	for i := 0; i < f.Points && i < len(f.DBM); i++ {
		freq := f.Start
		if f.Points > 1 {
			freq = f.Start + (f.Stop-f.Start)*float64(i)/float64(f.Points-1)
		}
		var max, avg, min float32
		if i < len(f.Max) {
			max = f.Max[i]
		}
		if i < len(f.Avg) {
			avg = f.Avg[i]
		}
		if i < len(f.Min) {
			min = f.Min[i]
		}
		fmt.Fprintf(&sb, "%.6f,%.2f,%.2f,%.2f,%.2f,%.1f\n", freq/1e6, f.DBM[i], max, avg, min, f.DBM[i]-f.Noise)
	}
	return []byte(sb.String())
}

// Traces accumulates max/min/running-average levels since the sweep started.
type Traces struct {
	Max []float32
	Min []float32
	Avg []float32
	n   int
}

// NewTraces creates empty traces for the given number of points.
func NewTraces(points int) *Traces {
	t := &Traces{
		Max: make([]float32, points),
		Min: make([]float32, points),
		Avg: make([]float32, points),
	}
	for i := range t.Max {
		t.Max[i] = -200
		t.Min[i] = 50
	}
	return t
}

// Update folds one sweep into the traces.
func (t *Traces) Update(dbm []float32) {
	t.n++
	for i, v := range dbm {
		if i >= len(t.Max) {
			break
		}
		if v > t.Max[i] {
			t.Max[i] = v
		}
		if v < t.Min[i] {
			t.Min[i] = v
		}
		t.Avg[i] += (v - t.Avg[i]) / float32(t.n)
	}
}

func cloneF32(v []float32) []float32 { return append([]float32(nil), v...) }

// Parser assembles scanraw frames from the raw byte stream.
//
// Wire format per sweep: '{' + points × (tag:1 + level:uint16 LE) + '}'.
// Sweeps are concatenated, so the stream looks like "{data}{data}{data}".
type Parser struct {
	cfg   SweepConfig
	scale float64
	buf   []byte
	in    bool
	seq   uint64
}

// NewParser creates a parser for cfg; scale is 174 (Ultra) or 128 (basic).
func NewParser(cfg SweepConfig, scale float64) *Parser {
	return &Parser{cfg: cfg, scale: scale, buf: make([]byte, 0, cfg.Points*3)}
}

// Feed consumes a chunk and calls emit for every complete sweep.
func (p *Parser) Feed(chunk []byte, emit func(Frame)) {
	need := p.cfg.Points * 3
	for _, b := range chunk {
		if !p.in {
			if b == '{' {
				p.in = true
				p.buf = p.buf[:0]
			}
			continue
		}
		p.buf = append(p.buf, b)
		if len(p.buf) < need {
			continue
		}
		p.seq++
		f := Frame{
			Seq:    p.seq,
			Start:  p.cfg.Start,
			Stop:   p.cfg.Stop,
			Points: p.cfg.Points,
			DBM:    make([]float32, p.cfg.Points),
		}
		for i := range f.DBM {
			v := uint16(p.buf[i*3+1]) | uint16(p.buf[i*3+2])<<8
			f.DBM[i] = float32(float64(v)/32 - p.scale)
		}
		p.in = false
		emit(f)
	}
}

// Sweeper runs scanraw sessions on one device.
type Sweeper struct {
	client *serial.Client
	model  Model
}

// NewSweeper creates a sweeper bound to an open device.
func NewSweeper(client *serial.Client, model Model) *Sweeper {
	return &Sweeper{client: client, model: model}
}

// Apply pushes atten/LNA/spur/RBW settings to the device.
func (s *Sweeper) Apply(ctx context.Context, cfg SweepConfig) error {
	cmds := make([]string, 0, 4)
	if cfg.Atten < 0 {
		cmds = append(cmds, "attenuate auto")
	} else {
		cmds = append(cmds, fmt.Sprintf("attenuate %d", cfg.Atten))
	}
	if s.model.Ultra {
		if cfg.LNA {
			cmds = append(cmds, "lna on")
		} else {
			cmds = append(cmds, "lna off")
		}
	}
	if cfg.Spur {
		cmds = append(cmds, "spur on")
	} else {
		cmds = append(cmds, "spur off")
	}
	if cfg.RBW <= 0 {
		cmds = append(cmds, "rbw auto")
	} else {
		cmds = append(cmds, fmt.Sprintf("rbw %g", cfg.RBW))
	}
	for _, cmd := range cmds {
		if _, err := s.client.Query(ctx, cmd); err != nil {
			return fmt.Errorf("%s: %w", cmd, err)
		}
	}
	return nil
}

// Run streams sweeps until ctx is cancelled, calling emit for each frame.
// Frames carry max/min/average traces accumulated since the start.
func (s *Sweeper) Run(ctx context.Context, cfg SweepConfig, emit func(Frame)) error {
	p := NewParser(cfg, s.model.Scale)
	tr := NewTraces(cfg.Points)
	return s.client.Stream(ctx, cfg.Command(), func(chunk []byte) error {
		p.Feed(chunk, func(f Frame) {
			tr.Update(f.DBM)
			f.Max = cloneF32(tr.Max)
			f.Min = cloneF32(tr.Min)
			f.Avg = cloneF32(tr.Avg)
			f.Noise = NoiseFloor(f.DBM)
			f.Peak, f.PeakFreq = PeakStats(f.DBM, f.Start, f.Stop)
			f.SNR = f.Peak - f.Noise
			emit(f)
		})
		return nil
	})
}
