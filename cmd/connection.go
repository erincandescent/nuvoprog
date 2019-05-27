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
package cmd

import (
	"errors"
	"fmt"

	"github.com/erincandescent/nuvoprog/protocol"
	"github.com/erincandescent/nuvoprog/target"
)

func connectToTarget() (*protocol.Device, *target.Definition, error) {
	devs, err := protocol.Connect()
	if err != nil {
		return nil, nil, err
	}

	switch {
	case len(devs) == 0:
		return nil, nil, errors.New("No programmer found")
	case len(devs) > 1:
		for _, dev := range devs {
			dev.Close()
		}
		return nil, nil, errors.New("Multiple programmers found - you must specify one")
	}

	dev := devs[0]
	// Defer like this to avoid capturing the value of dev now
	defer func() { dev.Close() }()

	ver, err := dev.GetVersion()
	if err != nil {
		return nil, nil, err
	}

	if ver.FirmwareVersion < protocol.FirmwareVersionRequired {
		return nil, nil, errors.New("Your programmer's firmware is out of date")
	}

	if targetName == "" {
		return nil, nil, errors.New("Target device not specified")
	}

	targetDev := target.ByName(targetName)
	if targetDev == nil {
		return nil, nil, fmt.Errorf("Target device '%s' not found", targetName)
	}

	// Most of this structure is TODO
	cfg := protocol.Config{
		Clock:       1000,
		ChipFamily:  targetDev.Family,
		Voltage:     3300,
		PowerTarget: 0,
		USBFuncE:    0,
	}

	if err := dev.SetConfig(cfg); err != nil {
		return nil, nil, err
	}

	if err := dev.Reset(protocol.Reset{
		Type:       protocol.ResetAuto,
		Connection: protocol.ConnectICPMode,
		Mode:       protocol.ResetExtMode,
	}); err != nil {
		return nil, nil, err
	}

	if err := dev.Reset(protocol.Reset{
		Type:       protocol.ResetNoneNuLink,
		Connection: protocol.ConnectICPMode,
		Mode:       protocol.ResetExtMode,
	}); err != nil {
		return nil, nil, err
	}

	devID, err := dev.CheckID()
	if err != nil {
		return nil, nil, err
	}

	if devID != targetDev.DeviceID {
		return nil, nil, errors.New("Unsupported device")
	}

	// Swivel to prevent defer closing our device
	d2 := dev
	dev = nil
	return d2, targetDev, nil
}

func resetAndCloseDevice(dev *protocol.Device) {
	// Experimentally observed sequence of commands to get the device to run again
	dev.Reset(protocol.Reset{
		Type:       protocol.ResetAuto,
		Connection: protocol.ConnectICPMode,
		Mode:       protocol.ResetExtMode,
	})

	dev.Reset(protocol.Reset{
		Type:       protocol.ResetAuto,
		Connection: protocol.ConnectDisconnect,
		Mode:       protocol.ResetMode1,
	})

	dev.Reset(protocol.Reset{
		Type:       protocol.ResetNoneNuLink,
		Connection: protocol.ConnectDisconnect,
		Mode:       protocol.ResetExtMode,
	})
	dev.Close()
}
