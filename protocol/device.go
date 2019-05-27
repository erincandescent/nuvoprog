// Copyright © 2019 Erin Shepherd
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
package protocol

import (
	"encoding/hex"
	"errors"
	"log"

	"github.com/karalabe/hid"
)

var (
	ErrWriteSizeIncorrect      = errors.New("Write of incorrect size")
	ErrReadSizeIncorrect       = errors.New("Read of incorrect size")
	ErrSequenceNumberIncorrect = errors.New("Incorrect sequence number")
)

type deviceConfig struct {
	NewFramer func() Framer
	EPOut     int
	EPIn      int
}

var devices = map[uint32]*deviceConfig{
	// Nu-Link-ME, UART off
	0x0416511c: &deviceConfig{
		NewFramer: NewV1Framer,
		EPOut:     0x04,
		EPIn:      0x83,
	},
	// Nu-Link-ME, UART on
	0x0416511d: &deviceConfig{
		NewFramer: NewV1Framer,
		EPOut:     0x04,
		EPIn:      0x83,
	},
}

type Device struct {
	config *deviceConfig
	framer Framer
	seqNo  uint8
	dev    *hid.Device
}

func (d *Device) Path() string {
	return d.dev.Path
}

func (d *Device) MaxPayloadSize() int {
	return d.framer.MaxBodyLength()
}

func (d *Device) nextSequenceNumber() uint8 {
	d.seqNo++
	if d.seqNo >= 0x80 {
		d.seqNo = 1
	}
	return d.seqNo
}

func (d *Device) Send(body []byte) error {
	seqNum := d.nextSequenceNumber()

	msg, err := d.framer.Frame(seqNum, body)
	if err != nil {
		return err
	}

	msgBytes := msg.Bytes()
	log.Println("> ", hex.EncodeToString([]byte(msgBytes)))
	l, err := d.dev.Write([]byte(msgBytes))
	if err != nil {
		return err
	} else if l != len(msgBytes) {
		return ErrWriteSizeIncorrect
	}

	return nil
}

func (d *Device) Receive() ([]byte, error) {
	inBuf := make([]byte, d.framer.FrameLength())

	attempt := 0
	for {
		l, err := d.dev.Read(inBuf)
		if err != nil {
			return nil, err
		} else if l != d.framer.FrameLength() {
			return nil, ErrReadSizeIncorrect
		}

		log.Println("< ", hex.EncodeToString([]byte(inBuf)))
		respf, err := d.framer.Unframe(inBuf)
		if err != nil {
			return nil, err
		} else if respf.SequenceNumber() != d.seqNo {
			log.Println("Expecting sequence number ", d.seqNo, ", got ", respf.SequenceNumber())
			attempt++
			if attempt == 5 {
				return nil, ErrSequenceNumberIncorrect
			} else {
				continue
			}
		}

		return respf.Body(), nil
	}
}

func (d *Device) Request(body []byte) ([]byte, error) {
	if err := d.Send(body); err != nil {
		return nil, err
	}

	return d.Receive()
}

func (d *Device) Close() {
	if d != nil && d.dev != nil {
		d.dev.Close()
		d.dev = nil
	}
}

func Connect() ([]*Device, error) {
	var nldevs []*Device
	defer func() {
		for _, d := range nldevs {
			d.Close()
		}
	}()

	for _, deviceInfo := range hid.Enumerate(0, 0) {
		vidpid := (uint32(deviceInfo.VendorID) << 16) | uint32(deviceInfo.ProductID)
		devcfg := devices[vidpid]

		if devcfg == nil {
			continue
		}

		dev, err := deviceInfo.Open()
		if err != nil {
			return nil, err
		}

		nldevs = append(nldevs, &Device{
			config: devcfg,
			framer: devcfg.NewFramer(),
			seqNo:  0,
			dev:    dev,
		})
	}

	// Clear nldevs before returning so we don't
	// immediately close them all
	rdevs := nldevs
	nldevs = nil
	return rdevs, nil
}
