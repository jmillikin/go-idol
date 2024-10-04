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

package idol

import (
	"bytes"
	"encoding/binary"
	"io"
	"iter"
	"math"
)

type AsMessage[T any] interface {
	Idol__Message() Message[T]
}

type Message[T any] interface {
	Self() T
	Type() MessageType[T]
	Size() uint32
	Fields() MessageFields
	Clone() MessageBuilder[T]

	isMessage(T)
}

type MessageFields interface {
	Name(tag uint16) string
	Has(tag uint16) bool
	Values() iter.Seq2[uint16, any]
}

func Clone[T any](message AsMessage[T]) MessageBuilder[T] {
	return message.Idol__Message().Clone()
}

type IsGeneratedMessage[T any] struct{}

func (IsGeneratedMessage[T]) isMessage(T) {}

type AsMessageType[T any] interface {
	Idol__MessageType() MessageType[T]
}

type MessageType[T any] interface {
	Decode(ctx *DecodeCtx, buf []uint8) error
	DecodeAs(ctx *DecodeCtx, buf []uint8) (T, error)

	isMessageType(T)
}

type IsGeneratedMessageType[T any] struct{}

func (IsGeneratedMessageType[T]) isMessageType(T) {}

type AsMessageBuilder[T any] interface {
	Idol__MessageBuilder() MessageBuilder[T]
}

type MessageBuilder[T any] interface {
	Self() AsMessageBuilder[T]
	Size() uint32
	EncodeTo(ctx *EncodeCtx, w io.Writer) error

	isMessageBuilder(T)
}

type IsGeneratedMessageBuilder[T any] struct{}

func (IsGeneratedMessageBuilder[T]) isMessageBuilder(T) {}

type DecodeCtx struct{}

func Decode[T AsMessageType[T]](ctx *DecodeCtx, buf []uint8) error {
	var zero T
	return zero.Idol__MessageType().Decode(ctx, buf)
}

func DecodeAs[T AsMessageType[T]](ctx *DecodeCtx, buf []uint8) (T, error) {
	var zero T
	return zero.Idol__MessageType().DecodeAs(ctx, buf)
}

type EncodeCtx struct{}

func Encode[T any](
	ctx *EncodeCtx,
	builder AsMessageBuilder[T],
) ([]uint8, error) {
	b := builder.Idol__MessageBuilder()
	var buf bytes.Buffer
	buf.Grow(int(b.Size()))
	if err := b.EncodeTo(ctx, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func EncodeTo[T any](
	ctx *EncodeCtx,
	builder AsMessageBuilder[T],
	w io.Writer,
) error {
	b := builder.Idol__MessageBuilder()
	if g, ok := w.(interface{ Grow(int) }); ok {
		g.Grow(int(b.Size()))
	}
	return b.EncodeTo(ctx, w)
}

// DecodedMessage {{{

type DecodedMessage struct {
	buf string
}

func (msg DecodedMessage) Size() uint32 {
	if len(msg.buf) == 0 {
		return 8
	}
	return leUint32([]byte(msg.buf[0:4]))
}

func (msg DecodedMessage) Has(tag uint16) bool {
	if len(msg.buf) == 0 {
		return false
	}
	thunkCount := leUint16([]byte(msg.buf[6:8]))
	if tag > thunkCount {
		return false
	}
	thunkOff := uint32(tag) * 8
	return msg.buf[thunkOff+3] != 0x00
}

func (msg DecodedMessage) GetIndirect(tag uint16) string {
	if !msg.Has(tag) {
		return ""
	}
	thunkOff := uint32(tag) * 8
	valueSize := leUint32([]byte(msg.buf[thunkOff+4 : thunkOff+8]))
	if valueSize == 0 {
		return ""
	}
	valueOff := leUint32([]byte(msg.buf[thunkOff : thunkOff+4]))
	valueOff = (valueOff & 0x0FFFFFFF) << 3
	return msg.buf[valueOff : valueOff+valueSize]
}

func (msg DecodedMessage) GetBool(tag uint16) bool {
	return msg.GetUint32(tag) == 1
}

func (msg DecodedMessage) GetUint8Array(tag uint16) Uint8Array {
	return Uint8Array{msg.GetIndirect(tag)}
}

func (msg DecodedMessage) GetUint32(tag uint16) uint32 {
	if !msg.Has(tag) {
		return 0
	}
	thunkOff := uint32(tag) * 8
	return leUint32([]byte(msg.buf[thunkOff+4 : thunkOff+8]))
}

func (msg DecodedMessage) GetUint64(tag uint16) uint64 {
	if buf := msg.GetIndirect(tag); len(buf) > 0 {
		return leUint64([]uint8(buf))
	}
	return 0
}

func (msg DecodedMessage) GetTextArray(tag uint16) TextArray {
	return TextArray{msg.GetIndirect(tag)}
}

func (msg DecodedMessage) GetAsciz(tag uint16) Asciz {
	if buf := msg.GetIndirect(tag); len(buf) > 0 {
		return buf
	}
	return "\x00"
}

func (msg DecodedMessage) GetText(tag uint16) Text {
	if buf := msg.GetIndirect(tag); len(buf) > 0 {
		return buf[:len(buf)-1]
	}
	return ""
}

// }}}

// MessageDecoder {{{

type MessageDecoder struct {
	ctx *DecodeCtx
	buf []uint8
	err error
}

func NewMessageDecoder(ctx *DecodeCtx, buf []uint8) *MessageDecoder {
	if uint64(len(buf)) > math.MaxUint32 {
		return &MessageDecoder{
			err: errTODO(),
		}
	}
	bufLen := uint32(len(buf))

	if bufLen < 8 {
		return &MessageDecoder{
			err: errTODO(),
		}
	}
	if bufLen%8 != 0 {
		return &MessageDecoder{
			err: errTODO(),
		}
	}
	if bufLen > MaxMessageSize {
		return &MessageDecoder{
			err: errTODO(),
		}
	}

	messageSize := leUint32(buf[0:4])
	if messageSize != bufLen {
		return &MessageDecoder{
			err: errTODO(),
		}
	}
	messageFlags := leUint16(buf[4:6])
	if messageFlags != 0x0000 {
		return &MessageDecoder{
			err: errTODO(),
		}
	}
	if err := decodeThunks(buf, uint64(messageSize)); err != nil {
		return &MessageDecoder{
			err: err,
		}
	}
	return &MessageDecoder{ctx, buf, nil}
}

func (d *MessageDecoder) Finish() error {
	if d.err != nil {
		return d.err
	}
	return nil
}

func (d *MessageDecoder) has(tag uint16) bool {
	if len(d.buf) == 0 {
		return false
	}
	thunkCount := leUint16(d.buf[6:8])
	if tag > thunkCount {
		return false
	}
	thunkOff := uint32(tag) * 8
	return d.buf[thunkOff+3] != 0x00
}

func (d *MessageDecoder) getIndirect(tag uint16) []uint8 {
	if !d.has(tag) {
		return nil
	}
	thunkOff := uint32(tag) * 8
	valueSize := leUint32(d.buf[thunkOff+4 : thunkOff+8])
	if valueSize == 0 {
		return nil
	}
	valueOff := leUint32(d.buf[thunkOff : thunkOff+4])
	valueOff = (valueOff & 0x0FFFFFFF) << 3
	return d.buf[valueOff : valueOff+valueSize]
}

func (d *MessageDecoder) Bool(tag uint16) {
	if d.err != nil {
		return
	}
	// TODO
}

func (d *MessageDecoder) BoolArray(tag uint16) {
	if d.err != nil {
		return
	}
	// TODO
}

func (d *MessageDecoder) Uint8(tag uint16) {
	if d.err != nil {
		return
	}
	// TODO
}

func (d *MessageDecoder) Uint8Array(tag uint16) {
	if d.err != nil {
		return
	}
	// TODO
}

func (d *MessageDecoder) Uint16(tag uint16) {
	if d.err != nil {
		return
	}
	// TODO
}

func (d *MessageDecoder) Uint32(tag uint16) {
	if d.err != nil {
		return
	}
	// TODO
}

func (d *MessageDecoder) Uint64(tag uint16) {
	if d.err != nil {
		return
	}
	// TODO
}

func (d *MessageDecoder) Text(tag uint16) {
	if d.err != nil {
		return
	}
	// TODO
}

func (d *MessageDecoder) TextArray(tag uint16) {
	if d.err != nil {
		return
	}
	// TODO
}

func (d *MessageDecoder) Message(
	tag uint16,
	decode func(ctx *DecodeCtx, buf []uint8) error,
) {
	if d.err != nil {
		return
	}
	buf := d.getIndirect(tag)
	if len(buf) == 0 {
		return
	}
	if err := decode(d.ctx, buf); err != nil {
		d.err = err
		return
	}
}

func (d *MessageDecoder) MessageArray(
	tag uint16,
	decode func(ctx *DecodeCtx, buf []uint8) error,
) {
	if d.err != nil {
		return
	}
	buf := d.getIndirect(tag)
	if len(buf) == 0 {
		return
	}

	bufLen := uint64(len(buf))
	if bufLen < 4 {
		d.err = errTODO()
		return
	}

	arrayLen := leUint32(buf[0:4])
	sizeOff := 4
	valueOff := 4 + uint64(arrayLen)*4
	if arrayLen&0x01 == 0x00 {
		valueOff += 4
	}
	if valueOff > bufLen {
		d.err = errTODO()
		return
	}

	for ii := uint32(0); ii < arrayLen; ii++ {
		valueSize := leUint32(buf[sizeOff : sizeOff+4])
		if valueSize == 0 {
			sizeOff += 4
			continue
		}
		if valueSize%8 != 0 {
			d.err = errTODO()
			return
		}
		valueEnd := valueOff + uint64(valueSize)
		if valueEnd > bufLen {
			d.err = errTODO()
			return
		}
		if err := decode(d.ctx, buf[valueOff:valueEnd]); err != nil {
			d.err = err
			return
		}
		sizeOff += 4
		valueOff = valueEnd
	}

	if valueOff != bufLen {
		d.err = errTODO()
		return
	}
}

func decodeThunks(buf []uint8, messageSize uint64) error {
	thunkCount := uint32(leUint16(buf[6:8]))
	dataOff := uint64(8 + thunkCount*8)
	if dataOff > messageSize {
		return errTODO()
	}

	valueOff := dataOff
	for tag := uint32(1); tag <= thunkCount; tag++ {
		thunk := buf[tag*8 : tag*8+8]
		flags := leUint16(thunk[2:4])
		if flags&0x8000 == 0x0000 {
			if leUint64(thunk) == 0 {
				continue
			}
			return errTODO()
		}
		if flags&0x3FFF != 0x0000 {
			return errTODO()
		}
		if flags&0x4000 == 0x4000 {
			valueSize := uint64(leUint32(thunk[4:8]))
			paddedSize := (valueSize + 0b111) & 0xFFFFFFF8
			if paddedSize > valueSize {
				padding := buf[valueOff+valueSize : valueOff+paddedSize]
				for _, pad := range padding {
					if pad != 0x00 {
						return errTODO()
					}
				}
			}
			thunkOffset := (uint32(valueOff) >> 3) | (uint32(flags) << 16)
			binary.LittleEndian.PutUint32(thunk[0:4], thunkOffset)
			valueOff += paddedSize
			if valueOff > messageSize {
				return errTODO()
			}
			continue
		}
		if handles := leUint16(thunk[0:2]); handles > 0 {
			if handles != 1 {
				return errTODO()
			}
			value := uint64(leUint32(thunk[4:8]))
			if value != 0xFFFFFFFF {
				return errTODO()
			}

			panic("decoding handles: TODO")
		}
	}

	if valueOff != messageSize {
		return errTODO()
	}

	return nil
}

// }}}

// MessageSizeBuilder {{{

type MessageSizeBuilder struct {
	dataSize  uint32
	thunksLen uint16
}

func (b *MessageSizeBuilder) Scalar(tag uint16) {
	b.thunksLen = tag
}

func (b *MessageSizeBuilder) Indirect(tag uint16, dataSize uint32) {
	b.thunksLen = tag
	if dataSize > 0 {
		s := uint64(b.dataSize) + uint64(dataSize)
		if s > math.MaxUint32 {
			s = math.MaxUint32
		}
		b.dataSize = uint32(s)
	}
}

func (b MessageSizeBuilder) Finish() (uint32, uint16) {
	if b.thunksLen == 0 {
		return 0, 0
	}
	s := 8 + uint64(b.thunksLen)*8 + uint64(b.dataSize)
	if s > math.MaxUint32 {
		s = math.MaxUint32
	}
	return uint32(s), b.thunksLen
}

// }}}
