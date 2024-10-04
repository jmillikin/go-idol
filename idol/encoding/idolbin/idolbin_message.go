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
	"iter"
	"math"
	"unsafe"

	"go.idol-lang.org/idol"
)

type Message struct {
	buf string
}

func NewMessage(buf string) (Message, error) {
	if len(buf) > math.MaxUint32 {
		return Message{}, errTODO()
	}
	bufLen := uint32(len(buf))

	if bufLen < 8 {
		return Message{}, errTODO()
	}
	if bufLen%8 != 0 {
		return Message{}, errTODO()
	}
	if bufLen > idol.MaxMessageSize {
		return Message{}, errTODO()
	}

	messageSize := leUint32([]byte(buf[0:4]))
	if messageSize != bufLen {
		return Message{}, errTODO()
	}
	messageFlags := leUint16([]byte(buf[4:6]))
	if messageFlags != 0x0000 {
		return Message{}, errTODO()
	}
	if err := validateEncodedThunks(buf, uint64(messageSize)); err != nil {
		return Message{}, err
	}
	return Message{buf}, nil
}

func (msg Message) Size() uint32 {
	if len(msg.buf) == 0 {
		return 0
	}
	return leUint32([]byte(msg.buf[0:4]))
}

func (msg Message) Field(tag uint16) *MessageField {
	if len(msg.buf) == 0 || tag == 0 {
		return nil
	}
	thunkCount := leUint16([]byte(msg.buf[6:8]))
	if tag > thunkCount {
		return nil
	}

	msgPtr := unsafe.Pointer(unsafe.StringData(msg.buf))
	thunk := (*[8]uint8)(unsafe.Add(msgPtr, uint32(tag)*8))
	return &MessageField{
		buf:   msg.buf,
		thunk: *thunk,
		tag:   tag,
	}
}

func (msg Message) Fields() iter.Seq2[uint16, *MessageField] {
	return func(yield func(uint16, *MessageField) bool) {
		if len(msg.buf) == 0 {
			return
		}
		thunkCount := uint32(leUint16([]byte(msg.buf[6:8])))
		msgPtr := unsafe.Pointer(unsafe.StringData(msg.buf))
		for ii := uint32(1); ii <= thunkCount; ii++ {
			if msg.buf[ii*8+3] != 0x00 {
				thunk := (*[8]uint8)(unsafe.Add(msgPtr, ii*8))
				field := &MessageField{
					buf:   msg.buf,
					thunk: *thunk,
					tag:   uint16(ii),
				}
				if !yield(field.tag, field) {
					return
				}
			}
		}
	}
}

func validateEncodedThunks(buf string, messageSize uint64) error {
	thunkCount := uint32(leUint16([]byte(buf[6:8])))
	dataOff := uint64(8 + thunkCount*8)
	if dataOff > messageSize {
		return errTODO()
	}

	valueOff := dataOff
	for tag := uint32(1); tag <= thunkCount; tag++ {
		thunk := buf[tag*8 : tag*8+8]
		flags := leUint16([]byte(thunk[2:4]))
		if flags&0x8000 == 0x0000 {
			if leUint64([]byte(thunk)) == 0 {
				continue
			}
			return errTODO()
		}
		if flags&0x3FFF != 0x0000 {
			return errTODO()
		}
		if flags&0x4000 == 0x4000 {
			valueSize := uint64(leUint32([]byte(thunk[4:8])))
			paddedSize := (valueSize + 0b111) & 0xFFFFFFF8
			if paddedSize > valueSize {
				padding := buf[valueOff+valueSize : valueOff+paddedSize]
				for _, pad := range padding {
					if pad != 0x00 {
						return errTODO()
					}
				}
			}
			valueOff += paddedSize
			if valueOff > messageSize {
				return errTODO()
			}
			continue
		}
		if handles := leUint16([]byte(thunk[0:2])); handles > 0 {
			if handles != 1 {
				return errTODO()
			}
			value := uint64(leUint32([]byte(thunk[4:8])))
			if value != 0xFFFFFFFF {
				return errTODO()
			}
		}
	}

	if valueOff != messageSize {
		return errTODO()
	}

	return nil
}
