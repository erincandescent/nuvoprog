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
	"bufio"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/erincandescent/nuvoprog/ihex"
	"github.com/erincandescent/nuvoprog/target"
)

type TargetData struct {
	TargetDefinition *target.Definition
	Config           []byte
	Data             []byte
}

func (d *TargetData) read(rd io.ReadCloser, offset, length uint32, config bool, kind string) (err error) {
	defer rd.Close()
	hrd := ihex.NewReader(rd)

	var b ihex.Block
	for b, err = hrd.Next(); err == nil; b, err = hrd.Next() {
		switch {
		case b.Address+uint32(len(b.Data)) <= length:
			copy(d.Data[offset+b.Address:], b.Data)

		case b.Address == d.TargetDefinition.Config.IHexOffset && config:
			d.Config = b.Data

		default:
			return fmt.Errorf("Block 0x%08x+%02d out of range for %s", b.Address, len(b.Data), kind)
		}
	}

	if err == io.EOF {
		err = nil
	}

	return
}

func (d *TargetData) APROM() ([]byte, error) {
	cfg, err := d.TargetDefinition.Config.Decode(d.Config)
	if err != nil {
		return nil, err
	}

	size := d.TargetDefinition.ProgMemSize - cfg.GetLDROMSize()
	return d.Data[:size], nil
}

func (d *TargetData) LDROM() ([]byte, error) {
	cfg, err := d.TargetDefinition.Config.Decode(d.Config)
	if err != nil {
		return nil, err
	}

	apsize := d.TargetDefinition.ProgMemSize - cfg.GetLDROMSize()

	if int(apsize) != len(d.Data) {
		return d.Data[apsize:], nil
	} else {
		return nil, nil
	}
}

func (d *TargetData) Write(ws io.WriteCloser) (err error) {
	w := ihex.NewWriter(ws)
	defer func() {
		if err == nil {
			err = w.Close()
		}
	}()

	if len(d.Config) > 0 {
		err = w.Write(d.TargetDefinition.Config.IHexOffset, d.Config)
		if err != nil {
			return
		}
	}

	err = w.Write(0, d.Data)
	return
}

func WriteHexBlock(ws io.WriteCloser, buf []byte) (err error) {
	w := ihex.NewWriter(ws)
	defer func() {
		if err == nil {
			err = w.Close()
		}
	}()
	err = w.Write(0, buf)
	return
}

func (d *TargetData) WriteAPROM(ws io.WriteCloser) error {
	aprom, err := d.APROM()
	if err != nil {
		return err
	}
	return WriteHexBlock(ws, aprom)
}

func (d *TargetData) WriteLDROM(ws io.WriteCloser) error {
	ldrom, err := d.LDROM()
	if err != nil {
		return err
	}
	return WriteHexBlock(ws, ldrom)
}

func openRead(arg string) (io.ReadCloser, error) {
	if arg == "-" {
		return ioutil.NopCloser(os.Stdin), nil
	} else {
		return os.Open(arg)
	}
}

type stdoutW struct {
	*bufio.Writer
}

func (w *stdoutW) Close() error {
	return w.Flush()
}

type fileW struct {
	*bufio.Writer
	f *os.File
}

func (w *fileW) Close() error {
	nm := w.f.Name()
	nms := strings.TrimSuffix(nm, "~")

	if err := w.Flush(); err != nil {
		return err
	}

	if err := w.f.Close(); err != nil {
		return err
	}

	return os.Rename(nm, nms)
}

func openWrite(arg string) (io.WriteCloser, error) {
	if arg == "-" {
		return &stdoutW{bufio.NewWriter(os.Stdout)}, nil
	} else {
		f, err := os.Create(arg + "~")
		if err != nil {
			return nil, err
		}

		return &fileW{
			bufio.NewWriter(f),
			f,
		}, nil
	}
}

func readConfig(td *target.Definition, arg string) ([]byte, error) {
	arg = strings.TrimSpace(arg)

	switch {
	case arg == "":
		return nil, errors.New("No configuration specified")
	case arg[0] == '{':
		cfgo := td.Config.NewConfig()
		if err := json.Unmarshal([]byte(arg), cfgo); err != nil {
			return nil, fmt.Errorf("Parsing configuration: %s", err)
		}

		return cfgo.MarshalBinary()

	case arg[0] >= '0' && arg[0] <= '9',
		arg[0] >= 'A' && arg[0] <= 'F',
		arg[0] >= 'a' && arg[0] <= 'f':
		cfg, err := hex.DecodeString(arg)
		if err != nil {
			return nil, err
		} else if len(cfg) < int(td.Config.MinSize) {
			return nil, errors.New("Specified configuration too short")
		} else if len(cfg) > int(td.Config.WriteSize) {
			return nil, errors.New("Specified configuration too long")
		} else {
			return cfg, nil
		}

	case arg[0] == '@':
		f, err := openRead(arg[1:])
		if err != nil {
			return nil, err
		}
		defer f.Close()

		buf, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, err
		}

		cfgo := td.Config.NewConfig()
		if err := json.Unmarshal(buf, cfgo); err != nil {
			return nil, fmt.Errorf("Parsing configuration: %s", err)
		}

		return cfgo.MarshalBinary()

	default:
		return nil, fmt.Errorf("'%s' not understood for config parameter", arg)
	}
}

func NewTargetData(td *target.Definition) *TargetData {
	d := &TargetData{}
	d.TargetDefinition = td
	d.Data = make([]byte, td.ProgMemSize)

	for i := range d.Data {
		d.Data[i] = 0xFF
	}

	return d
}

func ReadTargetData(
	config, image, aprom, ldrom string,
	td *target.Definition,
	needImage bool,
) (*TargetData, error) {
	var err error
	d := NewTargetData(td)

	if image == "" && aprom == "" && ldrom == "" && needImage {
		return nil, errors.New("No input files specified")
	} else if image == "" && aprom == "" && ldrom == "" {
		return nil, errors.New("Can only specify maximum of two of Image, APROM and LDROM")
	}

	if image != "" {
		rd, err := openRead(image)
		if err != nil {
			return nil, err
		}

		if err := d.read(rd, 0, uint32(td.ProgMemSize), true, "image"); err != nil {
			return nil, err
		}
	}

	if config != "" {
		d.Config, err = readConfig(td, config)
		if err != nil {
			return nil, err
		}
	}

	if len(d.Config) == 0 {
		return nil, errors.New("No configuration bytes specified in image or config parameter")
	}

	cfgo := td.Config.NewConfig()
	if err := cfgo.UnmarshalBinary(d.Config); err != nil {
		return nil, err
	}

	ldromSz := cfgo.GetLDROMSize()
	apromSz := td.ProgMemSize - ldromSz

	if ldromSz == 0 && ldrom != "" {
		return nil, errors.New("ldrom parameter specified but configuration does not support LDROM")
	}

	if aprom != "" {
		rd, err := openRead(aprom)
		if err != nil {
			return nil, err
		}

		for i := 0; i < int(apromSz); i++ {
			d.Data[i] = 0xFF
		}

		if err := d.read(rd, 0, uint32(apromSz), true, "aprom"); err != nil {
			return nil, err
		}
	}

	if ldrom != "" {
		rd, err := openRead(ldrom)
		if err != nil {
			return nil, err
		}

		for i := apromSz; i < td.ProgMemSize; i++ {
			d.Data[i] = 0xFF
		}

		if err := d.read(rd, uint32(apromSz), uint32(ldromSz), true, "ldrom"); err != nil {
			return nil, err
		}
	}

	return d, nil
}
