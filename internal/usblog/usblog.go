// SPDX-License-Identifier: GPL-3.0-or-later

// Package usblog records raw USB traffic for debugging and deterministic replay.
//
// Two files are written per session (see plan §4.1):
//
//	usb_<ts>.log — human readable: time, direction, hex and ASCII dump
//	usb_<ts>.bin — binary records for replay/tests:
//	               [dir:1][len:4 BE][elapsed-us:8 BE][payload]
//
// Files rotate when the current pair exceeds Options.MaxSize bytes; at most
// Options.MaxFiles numbered pairs are kept (index 0 is the first, reused last).
// A nil *Logger is a valid no-op logger.
package usblog

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Direction of a transfer.
type Direction byte

// Transfer directions.
const (
	TX Direction = 'T'
	RX Direction = 'R'
)

// Options configures a Logger.
type Options struct {
	// MaxSize is the rotation threshold for the current file pair, in bytes.
	MaxSize int64
	// MaxFiles is the number of numbered file pairs kept on disk.
	MaxFiles int
	// Timestamp is used in file names; zero means time.Now().
	Timestamp time.Time
}

// DefaultOptions returns the defaults fixed in the plan: 50 MB × 5 pairs.
func DefaultOptions() Options {
	return Options{MaxSize: 50 << 20, MaxFiles: 5}
}

// Logger writes raw traffic logs. Safe for concurrent use.
type Logger struct {
	mu      sync.Mutex
	dir     string
	base    string
	opts    Options
	mono    time.Time
	text    *os.File
	bin     *os.File
	index   int
	written int64
	closed  bool
}

// New creates a logger in dir and opens the first file pair.
func New(dir string, opts Options) (*Logger, error) {
	if opts.MaxSize <= 0 {
		opts.MaxSize = 50 << 20
	}
	if opts.MaxFiles <= 0 {
		opts.MaxFiles = 5
	}
	ts := opts.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("usblog: mkdir: %w", err)
	}
	l := &Logger{
		dir:  dir,
		base: "usb_" + ts.Format("20060102_150405"),
		opts: opts,
		mono: time.Now(),
	}
	if err := l.openFilesLocked(0); err != nil {
		return nil, err
	}
	return l, nil
}

// Paths returns the current text and binary log paths.
func (l *Logger) Paths() (text, bin string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.pathsLocked()
}

func (l *Logger) pathsLocked() (text, bin string) {
	suffix := ""
	if l.index > 0 {
		suffix = fmt.Sprintf(".%d", l.index)
	}
	return filepath.Join(l.dir, l.base+suffix+".log"),
		filepath.Join(l.dir, l.base+suffix+".bin")
}

func (l *Logger) openFilesLocked(index int) error {
	tp, bp := func() (string, string) {
		suffix := ""
		if index > 0 {
			suffix = fmt.Sprintf(".%d", index)
		}
		return filepath.Join(l.dir, l.base+suffix+".log"),
			filepath.Join(l.dir, l.base+suffix+".bin")
	}()

	tf, err := os.Create(tp)
	if err != nil {
		return fmt.Errorf("usblog: create log: %w", err)
	}
	bf, err := os.Create(bp)
	if err != nil {
		_ = tf.Close()
		return fmt.Errorf("usblog: create bin: %w", err)
	}
	l.text, l.bin = tf, bf
	l.index = index
	l.written = 0
	return nil
}

// rotateLocked switches to the next file pair, reusing index 0 after MaxFiles.
func (l *Logger) rotateLocked() {
	next := l.index + 1
	if next >= l.opts.MaxFiles {
		next = 0
	}
	tp, bp := func() (string, string) {
		suffix := ""
		if next > 0 {
			suffix = fmt.Sprintf(".%d", next)
		}
		return filepath.Join(l.dir, l.base+suffix+".log"),
			filepath.Join(l.dir, l.base+suffix+".bin")
	}()
	_ = l.text.Close()
	_ = l.bin.Close()
	_ = os.Remove(tp)
	_ = os.Remove(bp)
	if err := l.openFilesLocked(next); err != nil {
		l.closed = true // fail closed: stop logging rather than corrupt state
	}
}

// Record appends one transfer to both logs. No-op for nil receivers or empty
// payloads. Write errors are ignored: logging must never break the data path.
func (l *Logger) Record(dir Direction, data []byte) {
	if l == nil || len(data) == 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return
	}
	if l.written+int64(len(data)) > l.opts.MaxSize {
		l.rotateLocked()
		if l.closed {
			return
		}
	}

	now := time.Now().UTC()
	elapsed := time.Since(l.mono).Microseconds()

	var sb strings.Builder
	sb.Grow(40 + len(data)*4)
	fmt.Fprintf(&sb, "%s %c %04d:", now.Format("2006-01-02T15:04:05.000000Z"), dir, len(data))
	for _, b := range data {
		fmt.Fprintf(&sb, " %02x", b)
	}
	sb.WriteString(" |")
	for _, b := range data {
		if b >= 0x20 && b < 0x7f {
			sb.WriteByte(b)
		} else {
			sb.WriteByte('.')
		}
	}
	sb.WriteString("|\n")

	if n, err := io.WriteString(l.text, sb.String()); err == nil {
		l.written += int64(n)
	}

	var hdr [13]byte
	hdr[0] = byte(dir)
	binary.BigEndian.PutUint32(hdr[1:5], uint32(len(data)))
	binary.BigEndian.PutUint64(hdr[5:13], uint64(elapsed))
	if _, err := l.bin.Write(hdr[:]); err != nil {
		return
	}
	_, _ = l.bin.Write(data)
}

// Close flushes and closes the current file pair. Idempotent.
func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true
	err1 := l.text.Close()
	err2 := l.bin.Close()
	return errors.Join(err1, err2)
}

// Record is one entry read back from a binary log.
type Record struct {
	Dir       Direction
	ElapsedUS int64
	Payload   []byte
}

// ReadBinary decodes a ".bin" file written by Logger.
func ReadBinary(path string) ([]Record, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	var out []Record
	var hdr [13]byte
	for {
		if _, err := io.ReadFull(r, hdr[:]); err != nil {
			if errors.Is(err, io.EOF) {
				return out, nil
			}
			return out, fmt.Errorf("usblog: read header: %w", err)
		}
		rec := Record{
			Dir:       Direction(hdr[0]),
			ElapsedUS: int64(binary.BigEndian.Uint64(hdr[5:13])),
		}
		n := binary.BigEndian.Uint32(hdr[1:5])
		rec.Payload = make([]byte, n)
		if _, err := io.ReadFull(r, rec.Payload); err != nil {
			return out, fmt.Errorf("usblog: read payload: %w", err)
		}
		out = append(out, rec)
	}
}
