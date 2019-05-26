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

package ihex

import (
	"bufio"
	"errors"
	"io"
)

var (
	ErrInvalidPrefix       = errors.New("Colon prefix missing")
	ErrInvalidHex          = errors.New("Invalid hex digit")
	ErrInvalidEOL          = errors.New("Invalid line ending")
	ErrInvalidChecksum     = errors.New("Invalid checksum")
	ErrInvalidRecordLength = errors.New("Length invalid for record")
)

type PacketType byte

const (
	Data PacketType = iota
	EOF
	ExtendedSegmentAddress
	StartSegmentAddress
	ExtendedLinearAddress
	StartLinearAddress
)

type Packet struct {
	Type    PacketType
	Address uint16
	Data    []byte
}

func DataPacket(addr uint16, data []byte) Packet {
	return Packet{
		Type:    Data,
		Address: addr,
		Data:    data,
	}
}

func EOFPacket() Packet {
	return Packet{
		Type: EOF,
	}
}

func ExtendedSegmentAddressPacket(seg uint16) Packet {
	return Packet{
		Type: ExtendedSegmentAddress,
		Data: []byte{byte(seg >> 8), byte(seg)},
	}
}

func ExtendedLinearAddressPacket(seg uint16) Packet {
	return Packet{
		Type: ExtendedLinearAddress,
		Data: []byte{byte(seg >> 8), byte(seg)},
	}
}

func readHexByte(rdr *bufio.Reader) (byte, error) {
	n0, err := rdr.ReadByte()
	if err != nil {
		return 0, err
	}
	n1, err := rdr.ReadByte()
	if err != nil {
		return 0, err
	}

	var b byte
	switch {
	case n0 >= '0' && n0 <= '9':
		b = (n0 - '0') << 4
	case n0 >= 'A' && n0 <= 'F':
		b = (n0 - 'A' + 10) << 4
	case n0 >= 'a' && n0 <= 'f':
		b = (n0 - 'a' + 10) << 4
	default:
		return 0, ErrInvalidHex
	}

	switch {
	case n1 >= '0' && n1 <= '9':
		b |= (n1 - '0')
	case n1 >= 'A' && n1 <= 'F':
		b |= (n1 - 'A' + 10)
	case n1 >= 'a' && n1 <= 'f':
		b |= (n1 - 'a' + 10)
	default:
		return 0, ErrInvalidHex
	}

	return b, nil
}

func readHexWord(rdr *bufio.Reader) (uint16, error) {
	b0, err := readHexByte(rdr)
	if err != nil {
		return 0, err
	}
	b1, err := readHexByte(rdr)
	if err != nil {
		return 0, err
	}
	return uint16(b0)<<8 | uint16(b1), nil
}

func ReadPacket(rdr *bufio.Reader) (Packet, error) {
pfx:
	for {
		b, err := rdr.ReadByte()
		if err != nil {
			return Packet{}, err
		}

		switch b {
		case '\n', '\r':
			continue

		case ':':
			break pfx

		default:
			return Packet{}, ErrInvalidPrefix
		}
	}

	length, err := readHexByte(rdr)
	if err != nil {
		return Packet{}, err
	}

	addr, err := readHexWord(rdr)
	if err != nil {
		return Packet{}, err
	}

	ptype, err := readHexByte(rdr)
	if err != nil {
		return Packet{}, err
	}

	var expsum byte
	expsum = length + byte(addr>>8) + byte(addr) + ptype

	buf := make([]byte, 0, length)
	for i := 0; i < int(length); i++ {
		b, err := readHexByte(rdr)
		if err != nil {
			return Packet{}, err
		}
		expsum += b
		buf = append(buf, b)
	}

	recsum, err := readHexByte(rdr)
	if err != nil {
		return Packet{}, err
	}

	if -expsum != recsum {
		return Packet{}, ErrInvalidChecksum
	}

	b, err := rdr.ReadByte()
	switch {
	case err == io.EOF:
		// OK
	case err != nil:
		return Packet{}, err
	case b == '\r', b == '\n':
		// OK
	default:
		return Packet{}, ErrInvalidEOL
	}

	return Packet{
		Type:    PacketType(ptype),
		Address: addr,
		Data:    buf,
	}, nil
}

var hlut = "0123456789ABCDEF"

func appendHexByte(buf []byte, sum *byte, b byte) []byte {
	*sum += b
	return append(buf,
		hlut[b>>4],
		hlut[b&0xF])
}

func WritePacket(w io.Writer, p Packet) error {
	// Fixed overhead:
	// 1 Colon
	// 2 Packet
	// 4 Address
	// 2 Length
	// 2 Checksum
	// 1 Newline
	// = 12
	// + 2n data bytes

	var sum byte
	buf := make([]byte, 0, 12+2*len(p.Data))
	buf = append(buf, ':')
	buf = appendHexByte(buf, &sum, byte(len(p.Data)))
	buf = appendHexByte(buf, &sum, byte(p.Address>>8))
	buf = appendHexByte(buf, &sum, byte(p.Address))
	buf = appendHexByte(buf, &sum, byte(p.Type))
	for _, b := range p.Data {
		buf = appendHexByte(buf, &sum, b)
	}
	buf = appendHexByte(buf, &sum, 0-sum)
	buf = append(buf, '\n')

	_, err := w.Write(buf)
	return err
}

type Reader struct {
	r   *bufio.Reader
	seg uint32
	eof bool
}

type Block struct {
	Address uint32
	Data    []byte
}

func NewReader(r io.Reader) *Reader {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}

	return &Reader{br, 0, false}
}

func (r *Reader) Next() (Block, error) {
	if r.eof {
		return Block{}, io.EOF
	}

	for {
		p, err := ReadPacket(r.r)
		if err != nil {
			return Block{}, err
		}

		switch p.Type {
		case Data:
			return Block{
				Address: r.seg + uint32(p.Address),
				Data:    p.Data,
			}, nil
		case EOF:
			r.eof = true
			return Block{}, io.EOF
		case ExtendedSegmentAddress:
			if len(p.Data) != 2 {
				return Block{}, ErrInvalidRecordLength
			}
			r.seg = uint32(p.Data[0])<<12 | uint32(p.Data[1])<<4
		case StartSegmentAddress:
			continue
		case ExtendedLinearAddress:
			if len(p.Data) != 2 {
				return Block{}, ErrInvalidRecordLength
			}
			r.seg = uint32(p.Data[0])<<24 | uint32(p.Data[1])<<16
		case StartLinearAddress:
			continue
		}
	}
}

type Writer struct {
	w   io.WriteCloser
	seg uint32
}

func NewWriter(w io.WriteCloser) *Writer {
	return &Writer{w, 0}
}

func (w *Writer) write(addr uint32, buf []byte) error {
	if len(buf) == 0 {
		return nil
	}

	off := addr - w.seg
	if off > 0xFFFF {
		w.seg = addr & 0xFFFF0000
		off = addr - w.seg

		if err := WritePacket(w.w, ExtendedLinearAddressPacket(uint16(w.seg>>16))); err != nil {
			return err
		}
	}

	return WritePacket(w.w, DataPacket(uint16(off), buf))
}

func (w *Writer) Write(addr uint32, buf []byte) error {
	lead := int(32 - (addr & 31))
	if addr&31 != 0 && len(buf) > lead {
		if err := w.write(addr, buf[:lead]); err != nil {
			return err
		}
		addr += uint32(lead)
		buf = buf[lead:]
	}

	for len(buf) > 32 {
		if err := w.write(addr, buf[:32]); err != nil {
			return err
		}
		addr += 32
		buf = buf[32:]
	}

	return w.write(addr, buf)
}

func (w *Writer) WriteBlock(b Block) {
	w.Write(b.Address, b.Data)
}

func (w *Writer) Close() error {
	if err := WritePacket(w.w, EOFPacket()); err != nil {
		w.w.Close()
		w.w = nil
		return err
	}

	err := w.w.Close()
	w.w = nil
	return err
}
