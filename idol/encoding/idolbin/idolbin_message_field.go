// Copyright (c) 2024 John Millikin <john@john-millikin.com>
//
// Permission to use, copy, modify, and/or distribute this software for any
// purpose with or without fee is hereby granted.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH
// REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF MERCHANTABILITY
// AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT,
// INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM
// LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE OR
// OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR
// PERFORMANCE OF THIS SOFTWARE.
//
// SPDX-License-Identifier: 0BSD

package idolbin

import (
	"math"
	"strings"
	"unsafe"

	"go.idol-lang.org/idol"
)

type MessageField struct {
	buf   string
	thunk [8]uint8
	tag   uint16
}

func (f *MessageField) HandleCount() uint16 {
	return leUint16(f.thunk[0:2])
}

func (f *MessageField) Tag() uint16 {
	return f.tag
}

func (f *MessageField) IsPresent() bool {
	return f.thunk[3]&0x80 == 0x80
}

func (f *MessageField) IsScalar() bool {
	return f.thunk[3] == 0x80
}

func (f *MessageField) IsIndirect() bool {
	return f.thunk[3] == 0xC0
}

func (f *MessageField) GetBool() (bool, error) {
	scalar, err := f.GetUint32()
	if err != nil {
		return false, err
	}
	if scalar == 0 {
		return false, nil
	}
	if scalar == 1 {
		return true, nil
	}
	return false, errTODO() // bad bool
}

func (f *MessageField) GetUint8() (uint8, error) {
	scalar, err := f.GetUint32()
	if err != nil {
		return 0, err
	}
	if scalar > math.MaxUint8 {
		return 0, errTODO()
	}
	return uint8(scalar), nil
}

func (f *MessageField) GetUint16() (uint16, error) {
	scalar, err := f.GetUint32()
	if err != nil {
		return 0, err
	}
	if scalar > math.MaxUint16 {
		return 0, errTODO()
	}
	return uint16(scalar), nil
}

func (f *MessageField) GetUint32() (uint32, error) {
	if f.thunk[3] == 0x00 {
		return 0, nil
	}
	if f.thunk[3] == 0x80 {
		return leUint32(f.thunk[4:8]), nil
	}
	return 0, errTODO() // not a scalar
}

func (f *MessageField) GetUint8Array() (idol.Uint8Array, error) {
	value, err := f.getIndirect()
	if err != nil {
		return idol.Uint8Array{}, err
	}
	return *(*idol.Uint8Array)(unsafe.Pointer(&value)), nil
}

func (f *MessageField) GetAsciz() (idol.Asciz, error) {
	value, err := f.getIndirect()
	if err != nil {
		return "", err
	}
	if len(value) == 0 {
		return "\x00", nil
	}
	nul := strings.IndexByte(value, 0x00)
	if nul == -1 {
		return "", errTODO()
	}
	if nul != len(value)-1 {
		return "", errTODO()
	}
	return value, nil
}

func (f *MessageField) GetText() (idol.Text, error) {
	value, err := f.getIndirect()
	if err != nil || len(value) == 0 {
		return "", err
	}
	nul := strings.IndexByte(value, 0x00)
	if nul == -1 {
		return "", errTODO()
	}
	if nul != len(value)-1 {
		return "", errTODO()
	}
	return value[:nul], nil
}

func (f *MessageField) GetTextArray() (idol.TextArray, error) {
	value, err := f.getIndirect()
	if err != nil || len(value) == 0 {
		return idol.TextArray{}, err
	}
	if err := validateTextArray(value); err != nil {
		return idol.TextArray{}, err
	}
	return *(*idol.TextArray)(unsafe.Pointer(&value)), nil
}

func (f *MessageField) GetMessageArray() (idol.MessageArray[Message], error) {
	value, err := f.getIndirect()
	if err != nil {
		return idol.MessageArray[Message]{}, err
	}
	if err := validateMessageArray(value); err != nil {
		return idol.MessageArray[Message]{}, err
	}
	return *(*idol.MessageArray[Message])(unsafe.Pointer(&value)), nil
}

func (f *MessageField) getIndirect() (string, error) {
	if f.thunk[3] == 0x00 {
		return "", nil
	}
	if f.thunk[3] == 0x80 {
		return "", errTODO() // not an indirect
	}

	thunkCount := uint32(leUint16([]byte(f.buf[6:8])))
	valueOff := 8 + thunkCount*8
	for ii := uint32(1); ii < uint32(f.tag); ii++ {
		thunk := f.buf[ii*8 : ii*8+8]
		if thunk[3]&0x40 == 0x00 {
			continue
		}
		size := leUint32([]byte(thunk[4:8]))
		paddedSize := (size + 0b111) & 0xFFFFFFF8
		valueOff += paddedSize
	}

	valueSize := leUint32(f.thunk[4:8])
	return f.buf[valueOff : valueOff+valueSize], nil
}

func validateMessageArray(buf string) error {
	if buf == "" {
		return nil
	}
	bufLen := uint64(len(buf))
	if bufLen < 4 {
		return errTODO()
	}

	arrayLen := leUint32([]byte(buf[0:4]))
	sizeOff := 4
	valueOff := 4 + uint64(arrayLen)*4
	if arrayLen&0x01 == 0x00 {
		valueOff += 4
	}
	if valueOff > bufLen {
		return errTODO()
	}

	for ii := uint32(0); ii < arrayLen; ii++ {
		valueSize := leUint32([]byte(buf[sizeOff : sizeOff+4]))
		if valueSize == 0 {
			return errTODO()
		}
		if valueSize%8 != 0 {
			return errTODO()
		}
		valueEnd := valueOff + uint64(valueSize)
		if valueEnd > bufLen {
			return errTODO()
		}

		if _, err := NewMessage(buf[valueOff:valueEnd]); err != nil {
			return err
		}
		sizeOff += 4
		valueOff = valueEnd
	}

	if valueOff != bufLen {
		return errTODO()
	}

	return nil
}

func validateTextArray(buf string) error {
	if buf == "" {
		return nil
	}
	bufLen := uint64(len(buf))
	if bufLen < 4 {
		return errTODO()
	}

	arrayLen := leUint32([]byte(buf[0:4]))
	sizeOff := 4
	valueOff := 4 + uint64(arrayLen)*4
	if arrayLen&0x01 == 0x00 {
		valueOff += 4
	}
	if valueOff > bufLen {
		return errTODO()
	}

	for ii := uint32(0); ii < arrayLen; ii++ {
		valueSize := leUint32([]byte(buf[sizeOff : sizeOff+4]))
		if valueSize == 0 {
			return errTODO()
		}
		valueEnd := valueOff + uint64(valueSize)
		if valueEnd > bufLen {
			return errTODO()
		}

		value := buf[valueOff:valueEnd]
		nul := strings.IndexByte(value, 0x00)
		if nul == -1 {
			return errTODO()
		}
		if nul != len(value)-1 {
			return errTODO()
		}

		sizeOff += 4
		valueOff = valueEnd
	}

	if valueOff != bufLen {
		return errTODO()
	}

	return nil
}
