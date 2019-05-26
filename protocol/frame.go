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
	"encoding/binary"
	"errors"
)

var ErrFrameLengthIncorrect = errors.New("Frame length incorrect")
var ErrBodyLengthTooLong = errors.New("Body length too long")
var ErrTooShortForCommand = errors.New("Frame too short to contain command")

type Frame interface {
	SequenceNumber() byte
	BodyLength() int
	Body() []byte
	Command() (uint32, error)
	Bytes() []byte
}

type V1Frame []byte

func (f V1Frame) SequenceNumber() byte {
	return f[0]
}

func (f V1Frame) BodyLength() int {
	return int(f[1])
}

func (f V1Frame) Body() []byte {
	reqLen := 2 + f.BodyLength()
	return f[2:reqLen]
}

func (f V1Frame) Command() (uint32, error) {
	body := f.Body()
	if len(body) < 4 {
		return 0, ErrTooShortForCommand
	}

	return binary.LittleEndian.Uint32(body), nil
}

func (f V1Frame) Bytes() []byte {
	return []byte(f)
}

type Framer interface {
	FrameLength() int
	MaxBodyLength() int
	Frame(seqno byte, body []byte) (Frame, error)
	Unframe(pkg []byte) (Frame, error)
}

type V1Framer struct{}

func (f V1Framer) FrameLength() int {
	return 64
}

func (f V1Framer) MaxBodyLength() int {
	return 62
}

func (f V1Framer) Frame(seqno byte, body []byte) (Frame, error) {
	if len(body) > 62 {
		return nil, ErrBodyLengthTooLong
	}

	buf := make([]byte, 64)
	buf[0] = seqno
	buf[1] = byte(len(body))
	copy(buf[2:], body)

	return V1Frame(buf), nil
}

func (f V1Framer) Unframe(pkg []byte) (Frame, error) {
	if len(pkg) != 64 {
		return nil, ErrFrameLengthIncorrect
	}

	if pkg[1] > 62 {
		return nil, ErrBodyLengthTooLong
	}

	return V1Frame(pkg), nil
}

func NewV1Framer() Framer {
	return new(V1Framer)
}
