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
package n76

import (
	"errors"

	"github.com/erincandescent/nuvoprog/protocol"
	"github.com/erincandescent/nuvoprog/target"
)

//go:generate enumer -type=BootSelect -trimprefix=BootFrom -transform=snake -json -text

type N76E003LDROMSize byte

const (
	N76E003LDROM0KB N76E003LDROMSize = iota
	N76E003LDROM1KB
	N76E003LDROM2KB
	N76E003LDROM3KB
	N76E003LDROM4KB
)

type N76E003Config struct {
	// CONFIG0.CBS[7]
	BootSelect BootSelect `json:"boot_select"`
	// CONFIG0.OCDPWM[5]
	PWMEnabledDuringOCD bool `json:"pwm_enabled_during_ocd"`
	// CONFIG0.OCDEN[4]
	OCDEnabled bool `json:"ocd_enabled"`
	// CONFIG0.RPD[2]
	ResetPinDisabled bool `json:"reset_pin_disabled"`

	// CONFIG0.LOCK[1]
	Locked bool `json:"locked"`

	// CONFIG1.LDSIZE[2:0]
	LDROMSize N76E003LDROMSize `json:"ldrom_size"`

	// CONFIG2.CBODEN[7]
	BODDisabled bool `json:"bod_disabled"`

	// CONFIG2.COV[5:4]
	BODVoltage BODVoltage `json:"bod_voltage"`

	// CONFIG2.BOIAP[3]
	IAPEnabledInBrownout bool `json:"iap_enabled_in_brownout"`

	// CONFIG2.CBORST[2]
	BODResetDisabled bool `json:"bod_reset_disabled"`

	// CONFIG3.WDTEN[7:4]
	WDT WDTMode `json:"wdt"`
}

func (cfg *N76E003Config) UnmarshalBinary(buf []byte) error {
	if len(buf) < 4 {
		return errors.New("Too short for config bytes")
	}

	cfg.BootSelect = BootFromAPROM
	if buf[0]&0x80 == 0 {
		cfg.BootSelect = BootFromLDROM
	}

	cfg.PWMEnabledDuringOCD = buf[0]&0x20 == 0
	cfg.OCDEnabled = buf[0]&0x10 == 0
	cfg.ResetPinDisabled = buf[0]&0x04 == 0
	cfg.Locked = buf[0]&0x02 == 0

	switch buf[1] & 0x7 {
	case 7:
		cfg.LDROMSize = N76E003LDROM0KB
	case 6:
		cfg.LDROMSize = N76E003LDROM1KB
	case 5:
		cfg.LDROMSize = N76E003LDROM2KB
	case 4:
		cfg.LDROMSize = N76E003LDROM3KB
	default:
		cfg.LDROMSize = N76E003LDROM4KB
	}

	cfg.BODDisabled = buf[2]&0x80 == 0
	switch (buf[2] >> 4) & 0x3 {
	case 0:
		cfg.BODVoltage = BODVoltage4v4
	case 1:
		cfg.BODVoltage = BODVoltage3v7
	case 2:
		cfg.BODVoltage = BODVoltage2v7
	default:
		cfg.BODVoltage = BODVoltage2v2
	}

	cfg.IAPEnabledInBrownout = buf[2]&0x08 == 0
	cfg.BODResetDisabled = buf[2]&0x04 == 0
	switch buf[3] >> 4 {
	case 0xF:
		cfg.WDT = WDTDisabled
	case 0x5:
		cfg.WDT = WDTEnabled
	default:
		cfg.WDT = WDTEnabledAlways
	}

	return nil
}

func (cfg *N76E003Config) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 8)
	for i := range buf {
		buf[i] = 0xFF
	}

	if cfg.BootSelect == BootFromLDROM {
		buf[0] &= 0x7F
	}

	if cfg.PWMEnabledDuringOCD {
		buf[0] &= 0xDF
	}

	if cfg.OCDEnabled {
		buf[0] &= 0xEF
	}

	if cfg.ResetPinDisabled {
		buf[0] &= 0xFB
	}

	if cfg.Locked {
		buf[0] &= 0xFD
	}

	switch cfg.LDROMSize {
	case N76E003LDROM0KB:
		buf[1] = 0xFF
	case N76E003LDROM1KB:
		buf[1] = 0xFE
	case N76E003LDROM2KB:
		buf[1] = 0xFD
	case N76E003LDROM3KB:
		buf[1] = 0xFC
	case N76E003LDROM4KB:
		buf[1] = 0xFB
	}

	if cfg.BODDisabled {
		buf[2] &= 0x7F
	}

	switch cfg.BODVoltage {
	case BODVoltage4v4:
		buf[2] &= 0xCF
	case BODVoltage3v7:
		buf[2] &= 0xDF
	case BODVoltage2v7:
		buf[2] &= 0xEF
	case BODVoltage2v2:
		buf[2] &= 0xFF
	}

	if cfg.IAPEnabledInBrownout {
		buf[2] &= 0xF7
	}

	if cfg.BODResetDisabled {
		buf[2] &= 0xFB
	}

	switch cfg.WDT {
	case WDTDisabled:
		buf[3] = 0xFF
	case WDTEnabled:
		buf[3] = 0x5F
	case WDTEnabledAlways:
		buf[3] = 0x0F
	}

	// Sense checking: We should unmarshal to the same values
	var newCfg N76E003Config
	if err := newCfg.UnmarshalBinary(buf); err != nil {
		return nil, err
	}

	if newCfg != *cfg {
		panic("Roundtrip error")
	}

	return buf, nil
}

func (c *N76E003Config) GetLDROMSize() uint {
	switch c.LDROMSize {
	case N76E003LDROM0KB:
		return 0
	case N76E003LDROM1KB:
		return 1024
	case N76E003LDROM2KB:
		return 2048
	case N76E003LDROM3KB:
		return 3072
	case N76E003LDROM4KB:
		return 4096
	default:
		panic("Invalid size")
	}
}

var N76E003 = &target.Definition{
	Name:        "N76E003",
	Family:      protocol.ChipFamilyN76E003,
	DeviceID:    protocol.DeviceN76E003,
	ProgMemSize: 12 * 1024,
	LDROMOffset: 0x3800,
	Config: target.ConfigSpace{
		IHexOffset: 0x30000,
		MinSize:    4,
		ReadSize:   8,
		WriteSize:  32,
		NewConfig:  func() target.Config { return new(N76E003Config) },
	},
}

func init() {
	target.Register(N76E003)
}
