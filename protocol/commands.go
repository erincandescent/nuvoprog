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
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
)

func unmarshal(buf []byte, dst interface{}) error {
	return binary.Read(bytes.NewReader(buf), binary.LittleEndian, dst)
}

func marshalCommand(cmd uint32, body interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, cmd); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, body); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func checkResp(cmd uint32, buf []byte) error {
	var respc uint32
	if err := unmarshal(buf, &respc); err != nil {
		return err
	}

	if respc != cmd {
		return fmt.Errorf("Invalid response command %08x, expected %08x", respc, cmd)
	}

	return nil
}

type FirmwareVersion uint32

const (
	FirmwareVersion6909 FirmwareVersion = 6069

	FirmwareVersionRequired = FirmwareVersion6909
)

func (v FirmwareVersion) String() string {
	return fmt.Sprintf("%d", v)
}

type ProductID uint32

const (
	ProductIDNuLinkME ProductID = 0x00550501
)

func (p ProductID) String() string {
	switch p {
	case ProductIDNuLinkME:
		return "Nu-Link-Me"

	default:
		return fmt.Sprintf("0x%08x", uint32(p))
	}
}

const FlagIsNulinkPro = 0x00000001

type VersionInfo struct {
	FirmwareVersion FirmwareVersion
	ProductID       ProductID
	Flags           uint32

	// [NuLink pro only] Target voltage
	TargetVoltage uint16
	// [NuLink pro only] USB voltage
	USBVoltage uint16
}

func (vi VersionInfo) String() string {
	s := fmt.Sprintf("%16s - Firmware Version %s", vi.ProductID, vi.FirmwareVersion)

	if vi.Flags&FlagIsNulinkPro != 0 {
		s = fmt.Sprintf("%s (Target voltage: %f; USB voltage %f)", s,
			float64(vi.TargetVoltage)/1000,
			float64(vi.USBVoltage)/1000)
	}

	return s
}

func (d *Device) GetVersion() (VersionInfo, error) {
	req := make([]byte, d.MaxPayloadSize())
	for i := range req {
		req[i] = 0xFF
	}

	resp, err := d.Request(req)
	if err != nil {
		return VersionInfo{}, err
	}

	info := VersionInfo{}
	if err := unmarshal(resp, &info); err != nil {
		return VersionInfo{}, err
	}
	return info, nil
}

type ChipFamily uint32

const (
	// M2351 family (taken from Nuvoton's OpenOCD patch)
	ChipFamilyM2351 = 0x321

	// N76E003 family
	// This is probably "1T 8051" family, but I only have one
	// test device. If multiple devices use it, then we'll rename
	// this
	ChipFamilyN76E003 = 0x800
)

func (f ChipFamily) String() string {
	switch f {
	case ChipFamilyM2351:
		return "M2351"
	case ChipFamilyN76E003:
		return "N76E003"
	default:
		return fmt.Sprintf("0x%08x", uint32(f))
	}
}

type Config struct {
	Clock       uint32
	ChipFamily  ChipFamily
	Voltage     uint32
	PowerTarget uint32
	USBFuncE    uint32
}

func (d *Device) SetConfig(c Config) error {
	log.Print("Setting config ", c)
	cmdBuf, err := marshalCommand(0xA2, c)
	if err != nil {
		log.Print("Marshalling error ", err)
		return err
	}

	resp, err := d.Request(cmdBuf)
	if err != nil {
		log.Print("Communications error ", err)
		return err
	}

	if err := checkResp(0xA2, resp); err != nil {
		log.Print("Response error ", err)
		return err
	}
	log.Println("OK")
	return nil
}

// Reset Type. Constants taken from OpenOCD patch
type ResetType uint32

const (
	ResetAuto             ResetType = 0
	ResetHW               ResetType = 1
	ResetSysResetReq      ResetType = 2
	ResetVecReset         ResetType = 3
	ResetFastRescue       ResetType = 4
	ResetNoneNuLink       ResetType = 5
	ResetNone2_8051T1Only ResetType = 6
)

func (t ResetType) String() string {
	switch t {
	case ResetAuto:
		return "Auto"
	case ResetHW:
		return "HW"
	case ResetSysResetReq:
		return "Sys Reset Req"
	case ResetVecReset:
		return "Vec Reset"
	case ResetFastRescue:
		return "Fast Rescue"
	case ResetNoneNuLink:
		return "None (NuLink)"
	case ResetNone2_8051T1Only:
		return "None2 (8051T1Only)"
	default:
		return fmt.Sprintf("0x%08x", uint32(t))
	}
}

// Type of connection after reset. Constants taken from OpenOCD patch
type ResetConnType uint32

const (
	ConnectNormal     ResetConnType = 0
	ConnectPreReset   ResetConnType = 1
	ConnectUnderReset ResetConnType = 2
	ConnectNone       ResetConnType = 3
	ConnectDisconnect ResetConnType = 4
	ConnectICPMode    ResetConnType = 5
)

func (ct ResetConnType) String() string {
	switch ct {
	case ConnectNormal:
		return "Normal"
	case ConnectPreReset:
		return "Pre Reset"
	case ConnectUnderReset:
		return "Under Reset"
	case ConnectNone:
		return "None"
	case ConnectDisconnect:
		return "Disconnect"
	case ConnectICPMode:
		return "ICP Mode"
	default:
		return fmt.Sprintf("0x%08x", uint32(ct))
	}
}

// Reset mode
type ResetMode uint32

const (
	// Described as "ext mode" by OpenOCD patch?
	ResetExtMode ResetMode = 0

	// Mode 1, used when disconnecting device
	ResetMode1 ResetMode = 1
)

func (rm ResetMode) String() string {
	switch rm {
	case ResetExtMode:
		return "Ext Mode"
	default:
		return fmt.Sprintf("0x%08x", uint32(rm))
	}
}

type Reset struct {
	Type       ResetType
	Connection ResetConnType
	Mode       ResetMode
}

func (d *Device) Reset(r Reset) error {
	log.Print("Performing reset ", r)
	cmdBuf, err := marshalCommand(0xE2, r)
	if err != nil {
		log.Println("Marshalling error ", err)
		return err
	}

	resp, err := d.Request(cmdBuf)
	if err != nil {
		log.Println("Communications error ", err)
		return err
	}

	if err := checkResp(0xE2, resp); err != nil {
		log.Print("Response error ", err)
		return err
	}
	log.Println("OK")
	return nil
}

type DeviceID uint32

const (
	// N76E003, Observed in trace
	//
	// Matches IAP registers:
	//   0x00CCDDDD where
	//		CC   = Company ID
	// 		DDDD = Device ID
	DeviceN76E003 = 0xDA3650
)

func (id DeviceID) String() string {
	switch id {
	case DeviceN76E003:
		return "N76E003"
	default:
		return fmt.Sprintf("0x%08x", uint32(id))
	}
}

func (d *Device) CheckID() (DeviceID, error) {
	log.Print("Checking device ID")

	var fill uint32
	cmdBuf, err := marshalCommand(0xA3, fill)
	if err != nil {
		log.Println("Marshalling error ", err)
		return 0, err
	}

	resp, err := d.Request(cmdBuf)
	if err != nil {
		log.Println("Communications error ", err)
		return 0, err
	}

	if err := checkResp(0xA3, resp); err != nil {
		log.Print("Response error ", err)
		return 0, err
	}

	var did DeviceID
	if err := unmarshal(resp[4:], &did); err != nil {
		log.Print("Unmarshalling error ", err)
		return 0, err
	}

	log.Println("OK, Device ID ", did)
	return did, nil
}

type MemorySpace uint16

const (
	ProgramSpace MemorySpace = 0x0000
	ConfigSpace  MemorySpace = 0x0003
)

func (s MemorySpace) String() string {
	switch s {
	case ProgramSpace:
		return "program"

	case ConfigSpace:
		return "config"

	default:
		return fmt.Sprintf("0x%04x", uint16(s))
	}
}

type memCmd struct {
	Addr   uint16
	Space  MemorySpace
	Length uint32
}

func (d *Device) ReadMemory(space MemorySpace, address uint16, length uint8) ([]byte, error) {
	log.Printf("Reading %d bytes from %s 0x%04x", length, space, address)
	cmdBuf, err := marshalCommand(0xA1, memCmd{
		Addr:   address,
		Space:  space,
		Length: uint32(length),
	})
	if err != nil {
		log.Println("Marshalling error ", err)
		return nil, err
	}

	resp, err := d.Request(cmdBuf)
	if err != nil {
		log.Println("Communications error ", err)
		return nil, err
	}

	log.Printf("OK %x", resp)

	return resp, nil
}

func (d *Device) EraseFlashChip() error {
	log.Print("Erasing flash")
	cmdBuf, err := marshalCommand(0xA4, struct{}{})
	if err != nil {
		log.Println("Marshalling error ", err)
		return err
	}

	resp, err := d.Request(cmdBuf)
	if err != nil {
		log.Println("Communications error ", err)
		return err
	}

	if err := checkResp(0xA4, resp); err != nil {
		log.Print("Response error ", err)
		return err
	}
	log.Print("OK")
	return nil
}

func (d *Device) WriteMemory(space MemorySpace, address uint16, data []byte) error {
	log.Printf("Writing %d bytes to %s 0x%04x %s", len(data), space, address, hex.EncodeToString(data))
	cmdBuf, err := marshalCommand(0xA0, memCmd{
		Addr:   address,
		Space:  space,
		Length: uint32(len(data)),
	})

	if err != nil {
		log.Println("Marshalling error ", err)
		return err
	}

	cmdBuf = append(cmdBuf, data...)

	resp, err := d.Request(cmdBuf)
	if err != nil {
		log.Println("Communications error ", err)
		return err
	}

	if err := checkResp(0xA0, resp); err != nil {
		log.Print("Response error ", err)
		return err
	}
	log.Print("OK")
	return nil
}

// Not sure what this command does, but Nuvoton's software issues it
func (d *Device) UnknownA5() error {
	log.Print("A5")
	cmdBuf, err := marshalCommand(0xA5, struct{}{})
	if err != nil {
		log.Println("Marshalling error ", err)
		return err
	}

	for len(cmdBuf) < 24 {
		cmdBuf = append(cmdBuf, 0x00)
	}

	resp, err := d.Request(cmdBuf)
	if err != nil {
		log.Println("Communications error ", err)
		return err
	}

	if err := checkResp(0xA5, resp); err != nil {
		log.Print("Response error ", err)
		return err
	}
	log.Print("OK")
	return nil
}
