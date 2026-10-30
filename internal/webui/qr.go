// SPDX-License-Identifier: GPL-3.0-or-later

package webui

import (
	"errors"

	"rsc.io/qr"
)

// QRCodePNG renders text (normally the http://<lan-ip>:<port> URL) as a PNG
// QR code. scale is the number of image pixels per QR module.
func QRCodePNG(text string, scale int) ([]byte, error) {
	if text == "" {
		return nil, errors.New("qr: empty text")
	}
	if len(text) > 1024 {
		return nil, errors.New("qr: text is too long")
	}
	if scale < 1 || scale > 16 {
		scale = 8
	}

	code, err := qr.Encode(text, qr.M)
	if err != nil {
		return nil, err
	}
	code.Scale = scale
	return code.PNG(), nil
}
