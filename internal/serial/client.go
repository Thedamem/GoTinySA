// SPDX-License-Identifier: GPL-3.0-or-later

package serial

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"tinysa_go/internal/usblog"
)

// ErrClosed is returned when the client has been closed.
var ErrClosed = errors.New("serial: client closed")

// readSlice is the per-Read timeout inside readUntil; the caller's context
// provides the overall deadline.
const readSlice = 100 * time.Millisecond

type kind int

const (
	kindQuery kind = iota
	kindWrite
	kindStream
	kindAbort
)

type request struct {
	ctx  context.Context
	kind kind
	cmd  string // shell command without trailing CR
	fn   func([]byte) error
	resp chan result
}

type result struct {
	data []byte
	err  error
}

// Client serializes all access to one port: a single goroutine performs every
// read/write, callers talk to it through a channel. Nothing in here blocks the
// caller's UI thread beyond its own context deadline.
type Client struct {
	name string
	port Port
	log  *usblog.Logger

	reqs      chan request
	done      chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

// NewClient starts the command pump on port. log may be nil.
func NewClient(name string, port Port, log *usblog.Logger) *Client {
	c := &Client{
		name: name,
		port: port,
		log:  log,
		reqs: make(chan request),
		done: make(chan struct{}),
	}
	c.wg.Add(1)
	go c.loop()
	return c
}

// Name returns the port name the client was created for.
func (c *Client) Name() string { return c.name }

func (c *Client) loop() {
	defer c.wg.Done()
	for {
		select {
		case <-c.done:
			return
		case req := <-c.reqs:
			req.resp <- c.execute(req)
		}
	}
}

func (c *Client) execute(req request) result {
	switch req.kind {
	case kindQuery:
		data, err := c.doQuery(req.ctx, req.cmd)
		return result{data: data, err: err}
	case kindStream:
		return result{err: c.doStream(req.ctx, req.cmd, req.fn)}
	case kindAbort:
		return result{err: c.doAbort()}
	default:
		return result{err: c.doWrite(req.ctx, req.cmd)}
	}
}

// Query sends cmd and returns the response text with the echoed command, the
// trailing CRLF and the prompt stripped. Multi-line answers keep their inner
// CRLFs.
func (c *Client) Query(ctx context.Context, cmd string) (string, error) {
	data, err := c.roundTrip(ctx, kindQuery, cmd, nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Write sends cmd and waits until the device is ready again (prompt arrives),
// discarding the echo. Use it for commands with no useful response.
func (c *Client) Write(ctx context.Context, cmd string) error {
	_, err := c.roundTrip(ctx, kindWrite, cmd, nil)
	return err
}

// Stream puts the device into a continuous-output mode (e.g. scanraw with
// auto-repeat), delivering every raw chunk to fn until ctx is cancelled or fn
// returns an error. It then sends "abort" and drains the prompt.
//
// The returned error is ctx.Err() when the stream was stopped by the caller.
func (c *Client) Stream(ctx context.Context, cmd string, fn func([]byte) error) error {
	if fn == nil {
		return errors.New("serial: nil stream handler")
	}
	_, err := c.roundTrip(ctx, kindStream, cmd, fn)
	return err
}

// Abort stops any continuous output (e.g. a leftover scanraw from a previous
// session) and resynchronises the shell prompt.
func (c *Client) Abort() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := c.roundTrip(ctx, kindAbort, "", nil)
	return err
}

func (c *Client) roundTrip(ctx context.Context, k kind, cmd string, fn func([]byte) error) ([]byte, error) {
	req := request{ctx: ctx, kind: k, cmd: cmd, fn: fn, resp: make(chan result, 1)}
	select {
	case <-c.done:
		return nil, ErrClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	case c.reqs <- req:
	}
	select {
	case <-c.done:
		return nil, ErrClosed
	case res := <-req.resp:
		return res.data, res.err
	}
}

// Close stops the pump and closes the port. Idempotent and bounded: even a
// driver stuck in Write/Close cannot block the caller for more than ~4 s.
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		close(c.done)

		// Close the port before waiting: a pending Write on Windows waits in
		// GetOverlappedResult indefinitely, so the pump would never return.
		portDone := make(chan struct{})
		go func() {
			_ = c.port.Close()
			close(portDone)
		}()
		select {
		case <-portDone:
		case <-time.After(2 * time.Second):
		}

		pumpDone := make(chan struct{})
		go func() {
			c.wg.Wait()
			close(pumpDone)
		}()
		select {
		case <-pumpDone:
		case <-time.After(2 * time.Second):
		}
	})
	return nil
}

func (c *Client) doQuery(ctx context.Context, cmd string) ([]byte, error) {
	if err := c.writeLine(cmd); err != nil {
		return nil, err
	}
	raw, err := c.readUntil(ctx, Prompt)
	if err != nil {
		return nil, err
	}
	data := bytes.TrimSuffix(raw, []byte(Prompt))
	return trimEcho(data, cmd), nil
}

// trimEcho drops everything through the last echo of cmd ("cmd\r\n"), so stale
// bytes left over from an interrupted sweep cannot pollute the response.
func trimEcho(data []byte, cmd string) []byte {
	if i := bytes.LastIndex(data, []byte(cmd+"\r\n")); i >= 0 {
		data = data[i+len(cmd)+2:]
	} else if i := bytes.LastIndex(data, []byte(cmd+"\n")); i >= 0 {
		data = data[i+len(cmd)+1:]
	} else {
		data = dropEcho(data, cmd)
	}
	return bytes.TrimRight(data, "\r\n")
}

// doAbort runs on the pump goroutine.
func (c *Client) doAbort() error {
	_ = c.port.ResetInputBuffer()
	if err := c.writeLine("abort"); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _ = c.readUntil(ctx, Prompt)
	_ = c.port.ResetInputBuffer()
	return nil
}

func (c *Client) doWrite(ctx context.Context, cmd string) error {
	if err := c.writeLine(cmd); err != nil {
		return err
	}
	_, err := c.readUntil(ctx, Prompt)
	return err
}

// doStream reads raw bytes until ctx ends or the handler fails, then aborts
// the device and swallows the abort echo/prompt.
func (c *Client) doStream(ctx context.Context, cmd string, fn func([]byte) error) error {
	_ = c.port.ResetInputBuffer()
	if err := c.writeLine(cmd); err != nil {
		return err
	}

	tmp := make([]byte, 4096)
	var streamErr error
	for {
		if err := ctx.Err(); err != nil {
			streamErr = err
			break
		}
		if err := c.port.SetReadTimeout(readSlice); err != nil {
			streamErr = fmt.Errorf("serial: set read timeout: %w", err)
			break
		}
		n, err := c.port.Read(tmp)
		if n > 0 {
			chunk := append([]byte(nil), tmp[:n]...)
			c.log.Record(usblog.RX, chunk)
			if err := fn(chunk); err != nil {
				streamErr = err
				break
			}
		}
		if err != nil && !isTimeout(err) {
			streamErr = fmt.Errorf("serial: read: %w", err)
			break
		}
	}

	// Stop continuous output and resynchronise with the shell prompt.
	abortCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = c.writeLine("abort")
	_, _ = c.readUntil(abortCtx, Prompt)
	_ = c.port.ResetInputBuffer()

	return streamErr
}

// dropEcho removes the leading "cmd\r\n" line the device echoes back.
func dropEcho(data []byte, cmd string) []byte {
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		if bytes.Equal(bytes.TrimRight(data[:i], "\r"), []byte(cmd)) {
			return data[i+1:]
		}
		return data
	}
	if bytes.Equal(bytes.TrimRight(data, "\r"), []byte(cmd)) {
		return nil
	}
	return data
}

func (c *Client) writeLine(cmd string) error {
	data := []byte(cmd + "\r")
	c.log.Record(usblog.TX, data)
	if _, err := c.port.Write(data); err != nil {
		return fmt.Errorf("serial: write: %w", err)
	}
	return nil
}

func (c *Client) readUntil(ctx context.Context, term string) ([]byte, error) {
	var buf bytes.Buffer
	tmp := make([]byte, 1024)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := c.port.SetReadTimeout(readSlice); err != nil {
			return nil, fmt.Errorf("serial: set read timeout: %w", err)
		}
		n, err := c.port.Read(tmp)
		if n > 0 {
			chunk := append([]byte(nil), tmp[:n]...)
			c.log.Record(usblog.RX, chunk)
			buf.Write(chunk)
			if i := bytes.Index(buf.Bytes(), []byte(term)); i >= 0 {
				return buf.Bytes()[:i+len(term)], nil
			}
		}
		if err != nil && !isTimeout(err) {
			return nil, fmt.Errorf("serial: read: %w", err)
		}
	}
}

func isTimeout(err error) bool {
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}
