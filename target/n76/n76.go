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

// package n76 contians N76 family device definitions
package n76

type BootSelect int

const (
	BootFromLDROM BootSelect = iota
	BootFromAPROM
)

//go:generate enumer -type=N76E003LDROMSize -trimprefix=N76E003LDROM -transform=snake -json -text

type BODVoltage byte

const (
	BODVoltage4v4 BODVoltage = iota
	BODVoltage3v7
	BODVoltage2v7
	BODVoltage2v2
)

//go:generate enumer -type=BODVoltage -trimprefix=BODVoltage -transform=snake -json -text

type WDTMode byte

const (
	WDTDisabled WDTMode = iota
	WDTEnabled
	WDTEnabledAlways
)

//go:generate enumer -type=WDTMode -trimprefix=WDT -transform=snake -json -text
