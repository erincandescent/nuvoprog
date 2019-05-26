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
package target

import (
	"encoding"
	"fmt"
	"strings"

	"github.com/erincandescent/nuvoprog/protocol"
)

type Config interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler

	// Returns the LDROM size specified by this config,
	// (0 if not present)
	GetLDROMSize() uint
}

// Configuration space configuration for target
type ConfigSpace struct {
	// In Intel Hex files, configuration data will be stored
	// at this offset
	IHexOffset uint32

	// Minimum size of configuration data to be valid
	MinSize uint
	// Size to use when issuing reads
	ReadSize uint8
	// Size to use when issuing writes (data will be padded with FFs)
	WriteSize uint8

	// Create a new Config object
	NewConfig func() Config
}

// Decode config bytes
func (cs *ConfigSpace) Decode(buf []byte) (Config, error) {
	cfgo := cs.NewConfig()
	return cfgo, cfgo.UnmarshalBinary(buf)
}

// Definition of a target
type Definition struct {
	// Name of target device
	Name string

	// Device family
	Family protocol.ChipFamily

	// Device ID
	DeviceID protocol.DeviceID

	// Program memory size
	ProgMemSize uint

	// LDROM offset
	// If LDROM is enabled, then it starts at this address in
	// program space from the perspective of the programmer
	LDROMOffset uint

	// Config space configuration
	Config ConfigSpace
}

var (
	targetByName = map[string]*Definition{}
	targetByID   = map[uint64]*Definition{}
)

func Register(td *Definition) {
	name := strings.ToLower(td.Name)
	id := uint64(td.Family)<<32 | uint64(td.DeviceID)

	if _, ok := targetByName[name]; ok {
		panic("Target already registered with name " + name)
	}

	if _, ok := targetByID[id]; ok {
		panic(fmt.Sprintf("Target already registered with ID %08x:%08x", td.Family, td.DeviceID))
	}

	targetByName[name] = td
	targetByID[id] = td
}

func ByName(name string) *Definition {
	return targetByName[strings.ToLower(name)]
}

func ByID(f protocol.ChipFamily, d protocol.DeviceID) *Definition {
	return targetByID[uint64(f)<<32|uint64(d)]
}
