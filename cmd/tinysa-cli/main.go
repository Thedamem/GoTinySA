// SPDX-License-Identifier: GPL-3.0-or-later

// Command tinysa-cli is a diagnostic shell for tinySA devices. It exercises
// the serial layer (and the raw USB logger) without the GUI and is the base
// for hardware acceptance of milestone M1.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"tinysa_go/internal/serial"
	"tinysa_go/internal/usblog"
	ver "tinysa_go/internal/version"
)

func main() {
	portFlag := flag.String("port", "", "serial port (default: first tinySA found)")
	baud := flag.Int("baud", serial.DefaultBaud, "baud rate")
	list := flag.Bool("list", false, "list serial ports and exit")
	logDir := flag.String("log", "", "enable raw USB log into this directory")
	timeout := flag.Duration("timeout", 3*time.Second, "per-command timeout")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "tinysa-cli %s\n\nusage: tinysa-cli [flags] [command ...]\n\nflags:\n", ver.String())
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nwith no command an interactive console is started.\n")
	}
	flag.Parse()

	if *list {
		if err := printPorts(); err != nil {
			fatal(err)
		}
		return
	}

	name := *portFlag
	if name == "" {
		devs, err := serial.FindTinySA()
		if err != nil {
			fatal(err)
		}
		if len(devs) == 0 {
			fatal(fmt.Errorf("no tinySA found; use -list or -port"))
		}
		if len(devs) > 1 {
			fmt.Fprintf(os.Stderr, "multiple tinySA devices: %s; using %s\n", devNames(devs), devs[0].Name)
		}
		name = devs[0].Name
	}

	port, err := serial.Open(name, *baud)
	if err != nil {
		fatal(err)
	}

	var log *usblog.Logger
	if *logDir != "" {
		log, err = usblog.New(*logDir, usblog.DefaultOptions())
		if err != nil {
			fatal(err)
		}
		tp, bp := log.Paths()
		fmt.Fprintf(os.Stderr, "usb log: %s / %s\n", tp, bp)
	}

	client := serial.NewClient(name, port, log)
	defer func() {
		_ = client.Close()
		_ = log.Close()
	}()

	fmt.Printf("connected: %s (tinysa-cli %s)\n", name, ver.String())

	if flag.NArg() > 0 {
		runOne(client, strings.Join(flag.Args(), " "), *timeout)
		return
	}
	runConsole(client, *timeout)
}

func printPorts() error {
	devs, err := serial.ListPorts()
	if err != nil {
		return err
	}
	for _, d := range devs {
		mark := ""
		if d.IsTinySA {
			mark = "  <- tinySA"
		}
		fmt.Printf("%-20s %s:%s %-14s %s%s\n", d.Name, d.VID, d.PID, d.Serial, d.Product, mark)
	}
	return nil
}

func runOne(c *serial.Client, cmd string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := c.Query(ctx, cmd)
	if err != nil {
		fatal(err)
	}
	fmt.Println(out)
}

func runConsole(c *serial.Client, timeout time.Duration) {
	fmt.Println("interactive console: type a command, 'quit' or Ctrl-D to exit")
	sc := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !sc.Scan() {
			fmt.Println()
			return
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if line == "quit" || line == "exit" {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		out, err := c.Query(ctx, line)
		cancel()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			continue
		}
		fmt.Println(out)
	}
}

func devNames(devs []serial.Device) string {
	names := make([]string, 0, len(devs))
	for _, d := range devs {
		names = append(names, d.Name)
	}
	return strings.Join(names, ", ")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "tinysa-cli:", err)
	os.Exit(1)
}
