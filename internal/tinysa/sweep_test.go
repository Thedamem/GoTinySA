// SPDX-License-Identifier: GPL-3.0-or-later

package tinysa

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestDetectUltra(t *testing.T) {
	m := Detect("tinySA4_v1.4-252-g2b4aa51")
	if !m.Ultra {
		t.Fatal("tinySA4 must be detected as Ultra")
	}
	if m.Scale != 174 {
		t.Fatalf("scale = %v, want 174", m.Scale)
	}
	if m.MaxFreq != MaxFreqUltra {
		t.Fatalf("max freq = %v", m.MaxFreq)
	}
}

func TestDetectBasic(t *testing.T) {
	m := Detect("tinySA_v1.4-175-g1419a93")
	if m.Ultra {
		t.Fatal("tinySA must not be Ultra")
	}
	if m.Scale != 128 {
		t.Fatalf("scale = %v, want 128", m.Scale)
	}
	if m.MaxFreq != MaxFreqBasic {
		t.Fatalf("max freq = %v", m.MaxFreq)
	}
}

func TestDetectWithHWSuffix(t *testing.T) {
	m := Detect("tinySA4_v1.4-252-g2b4aa51-dirtyHW Version:V0.4.6")
	if !m.Ultra {
		t.Fatal("Ultra must be detected even with trailing HW text")
	}
	if m.MaxFreq != MaxFreqUltra {
		t.Fatalf("max freq = %v", m.MaxFreq)
	}
}

func TestConfigValidate(t *testing.T) {
	ultra := Detect("tinySA4_v1.4")
	ok := SweepConfig{Start: 1e6, Stop: 10e6, Points: 400}
	if err := ok.Validate(ultra); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	if err := (SweepConfig{Start: 10e6, Stop: 1e6, Points: 400}).Validate(ultra); err == nil {
		t.Fatal("reversed range accepted")
	}
	if err := (SweepConfig{Start: 1e6, Stop: 13e9, Points: 400}).Validate(ultra); err == nil {
		t.Fatal("out-of-range stop accepted")
	}
	if err := (SweepConfig{Start: 1e6, Stop: 2e6, Points: 1}).Validate(ultra); err == nil {
		t.Fatal("1 point accepted")
	}
}

func TestCommand(t *testing.T) {
	cfg := SweepConfig{Start: 1e6, Stop: 2e6, Points: 400}
	if got, want := cfg.Command(), "scanraw 1000000 2000000 400 3"; got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
}

// encodeSweep builds one '{data}' block with the given raw levels.
func encodeSweep(levels []uint16) []byte {
	out := make([]byte, 0, 1+len(levels)*3+1)
	out = append(out, '{')
	var b [2]byte
	for _, v := range levels {
		binary.LittleEndian.PutUint16(b[:], v)
		out = append(out, 'c', b[0], b[1])
	}
	out = append(out, '}')
	return out
}

func TestParserFrames(t *testing.T) {
	cfg := SweepConfig{Start: 1e6, Stop: 4e6, Points: 4}
	p := NewParser(cfg, 174)

	levels := []uint16{0, 1024, 2048, 4095}
	stream := append([]byte("scanraw 1000000 4000000 4 3\r\n"), encodeSweep(levels)...)
	stream = append(stream, encodeSweep(levels)...)

	var frames []Frame
	// Feed in odd-sized chunks to exercise the state machine.
	for i := 0; i < len(stream); i += 5 {
		end := i + 5
		if end > len(stream) {
			end = len(stream)
		}
		p.Feed(stream[i:end], func(f Frame) { frames = append(frames, f) })
	}

	if len(frames) != 2 {
		t.Fatalf("got %d frames, want 2", len(frames))
	}
	for _, f := range frames {
		if f.Points != 4 || len(f.DBM) != 4 {
			t.Fatalf("frame shape: %+v", f)
		}
		want := []float32{-174, -142, -110, float32(4095.0/32 - 174)}
		for i := range want {
			if math.Abs(float64(f.DBM[i]-want[i])) > 0.001 {
				t.Fatalf("dbm[%d] = %v, want %v", i, f.DBM[i], want[i])
			}
		}
	}
	if frames[0].Seq != 1 || frames[1].Seq != 2 {
		t.Fatalf("sequence numbers: %d, %d", frames[0].Seq, frames[1].Seq)
	}
}

func TestNoiseFloor(t *testing.T) {
	dbm := []float32{-100, -102, -99, -101, -40, -98}
	nf := NoiseFloor(dbm)
	if nf < -102 || nf > -98 {
		t.Fatalf("noise floor = %v", nf)
	}
	level, freq := PeakStats([]float32{-100, -40, -90}, 100e6, 200e6)
	if level != -40 || freq != 150e6 {
		t.Fatalf("peak = %v @ %v", level, freq)
	}
}

func TestFrameCSV(t *testing.T) {
	f := Frame{
		Start:  100e6,
		Stop:   200e6,
		Points: 3,
		DBM:    []float32{-100, -40, -80},
		Max:    []float32{-90, -30, -70},
		Avg:    []float32{-95, -35, -75},
		Min:    []float32{-110, -50, -90},
		Noise:  -100,
	}
	got := string(f.CSV())
	want := "Frequency_MHz,Live_dBm,Max_dBm,Avg_dBm,Min_dBm,SNR_dB\n" +
		"100.000000,-100.00,-90.00,-95.00,-110.00,0.0\n" +
		"150.000000,-40.00,-30.00,-35.00,-50.00,60.0\n" +
		"200.000000,-80.00,-70.00,-75.00,-90.00,20.0\n"
	if got != want {
		t.Fatalf("CSV mismatch:\n got %q\nwant %q", got, want)
	}
}

func TestTraces(t *testing.T) {
	tr := NewTraces(3)
	tr.Update([]float32{-100, -90, -80})
	tr.Update([]float32{-110, -70, -80})

	if tr.Max[0] != -100 || tr.Max[1] != -70 || tr.Max[2] != -80 {
		t.Fatalf("max = %v", tr.Max)
	}
	if tr.Min[0] != -110 || tr.Min[1] != -90 || tr.Min[2] != -80 {
		t.Fatalf("min = %v", tr.Min)
	}
	wantAvg := []float32{-105, -80, -80}
	for i := range wantAvg {
		if tr.Avg[i] != wantAvg[i] {
			t.Fatalf("avg[%d] = %v, want %v", i, tr.Avg[i], wantAvg[i])
		}
	}
}

func TestParserIgnoresGarbage(t *testing.T) {
	cfg := SweepConfig{Start: 1e6, Stop: 2e6, Points: 2}
	p := NewParser(cfg, 128)
	var frames []Frame
	p.Feed([]byte("noise\r\n} garbage {"), func(f Frame) { frames = append(frames, f) })
	if len(frames) != 0 {
		t.Fatalf("unexpected frames: %d", len(frames))
	}
	p.Feed(encodeSweep([]uint16{100, 200})[1:], func(f Frame) { frames = append(frames, f) })
	if len(frames) != 1 {
		t.Fatalf("got %d frames after incomplete start, want 1", len(frames))
	}
}
