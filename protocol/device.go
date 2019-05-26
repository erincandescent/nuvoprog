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
	"errors"
	"fmt"

	"github.com/google/gousb"
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
	0x0416511c: &deviceConfig{
		NewFramer: NewV1Framer,
		EPOut:     0x04,
		EPIn:      0x83,
	},
}

type Device struct {
	config *deviceConfig
	framer Framer
	seqNo  uint8
	dev    *gousb.Device
	cfg    *gousb.Config
	ifc    *gousb.Interface
	in     *gousb.InEndpoint
	out    *gousb.OutEndpoint
}

func (d *Device) Path() string {
	return fmt.Sprintf("%d.%d", d.dev.Desc.Bus, d.dev.Desc.Address)
}

func (d *Device) MaxPayloadSize() int {
	return d.framer.MaxBodyLength()
}

func (d *Device) nextSequenceNumber() uint8 {
	d.seqNo++
	if d.seqNo >= 0x80 {
		d.seqNo = 0
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
	l, err := d.out.Write([]byte(msgBytes))
	if err != nil {
		return err
	} else if l != len(msgBytes) {
		return ErrWriteSizeIncorrect
	}

	return nil
}

func (d *Device) Receive() ([]byte, error) {
	inBuf := make([]byte, d.framer.FrameLength())
	l, err := d.in.Read(inBuf)
	if err != nil {
		return nil, err
	} else if l != d.framer.FrameLength() {
		return nil, ErrReadSizeIncorrect
	}

	respf, err := d.framer.Unframe(inBuf)
	if err != nil {
		return nil, err
	} else if respf.SequenceNumber() != d.seqNo {
		return nil, ErrSequenceNumberIncorrect
	}
	return respf.Body(), nil
}

func (d *Device) Request(body []byte) ([]byte, error) {
	if err := d.Send(body); err != nil {
		return nil, err
	}

	return d.Receive()
}

func (d *Device) Close() {
	if d != nil && d.ifc != nil {
		d.ifc.Close()
		d.dev = nil
		d.cfg = nil
		d.ifc = nil
		d.in = nil
		d.out = nil
	}
}

func Connect(ctx *gousb.Context) ([]*Device, error) {
	baseDevs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		vidpid := (uint32(desc.Vendor) << 16) | uint32(desc.Product)
		_, ok := devices[vidpid]
		return ok
	})

	if err != nil {
		return nil, err
	}

	nldevs := make([]*Device, 0, len(baseDevs))
	defer func() {
		for _, d := range nldevs {
			d.Close()
		}
	}()

	for _, usbdev := range baseDevs {
		vidpid := (uint32(usbdev.Desc.Vendor) << 16) | uint32(usbdev.Desc.Product)
		devcfg := devices[vidpid]

		if devcfg == nil {
			continue
		}

		if len(usbdev.Desc.Configs) != 1 {
			return nil, fmt.Errorf("Too many configs (%d)", len(usbdev.Desc.Configs))
		}

		var cfgdesc gousb.ConfigDesc
		for _, cfg := range usbdev.Desc.Configs {
			cfgdesc = cfg
		}

		haveInterface := false
		var ifcdesc gousb.InterfaceSetting
	ifc:
		for _, ifc := range cfgdesc.Interfaces {
			for _, setting := range ifc.AltSettings {
				if setting.Class == gousb.ClassData {
					haveInterface = true
					ifcdesc = setting
					break ifc
				}
			}
		}

		if !haveInterface {
			return nil, errors.New("Unable to find interface setting")
		}

		cfg, err := usbdev.Config(cfgdesc.Number)
		if err != nil {
			return nil, err
		}

		ifc, err := cfg.Interface(ifcdesc.Number, ifcdesc.Alternate)
		if err != nil {
			return nil, err
		}

		inep, err := ifc.InEndpoint(devcfg.EPIn)
		if err != nil {
			return nil, err
		}

		outep, err := ifc.OutEndpoint(devcfg.EPOut)
		if err != nil {
			return nil, err
		}

		nldevs = append(nldevs, &Device{
			config: devcfg,
			framer: devcfg.NewFramer(),
			seqNo:  0,
			dev:    usbdev,
			cfg:    cfg,
			ifc:    ifc,
			in:     inep,
			out:    outep,
		})
	}

	// Clear nldevs before returning so we don't
	// immediately close them all
	rdevs := nldevs
	nldevs = nil
	return rdevs, nil
}
