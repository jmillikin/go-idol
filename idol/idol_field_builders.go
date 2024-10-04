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
	"strings"
)

// BoolFieldBuilder {{{

type BoolFieldBuilder struct {
	value bool
}

func (b *BoolFieldBuilder) IsPresent() bool {
	return b.value
}

func (b *BoolFieldBuilder) PutThunk(thunk []uint8) {
	if b.IsPresent() {
		binary.LittleEndian.PutUint16(thunk[2:4], 0x8000)
		binary.LittleEndian.PutUint32(thunk[4:8], 1)
	}
}

func (b *BoolFieldBuilder) Get() bool {
	return b.value
}

func (b *BoolFieldBuilder) Set(value bool) {
	b.value = value
}

// }}}

// EnumFieldBuilder {{{

type EnumFieldBuilder[T interface {
	~uint8 | ~uint16 | ~uint32 | ~int8 | ~int16 | ~int32
}] struct {
	value T
}

func (b *EnumFieldBuilder[T]) IsPresent() bool {
	return b.value != 0
}

func (b *EnumFieldBuilder[T]) PutThunk(thunk []uint8) {
	if b.IsPresent() {
		binary.LittleEndian.PutUint16(thunk[2:4], 0x8000)
		binary.LittleEndian.PutUint32(thunk[4:8], uint32(b.value))
	}
}

func (b *EnumFieldBuilder[T]) Get() T {
	return b.value
}

func (b *EnumFieldBuilder[T]) Set(value T) {
	b.value = value
}

// }}}

// EnumArrayFieldBuilder {{{

type EnumArrayFieldBuilder[T any] struct{}

// }}}

// Uint8ArrayFieldBuilder {{{

type Uint8ArrayFieldBuilder struct {
	value     interface{}
	valueSize uint32
}

func (b *Uint8ArrayFieldBuilder) IsPresent() bool {
	return b.valueSize > 0
}

func (b *Uint8ArrayFieldBuilder) DataSize() uint32 {
	if b.valueSize%8 == 0 {
		return b.valueSize
	}
	return (b.valueSize + 0b111) & 0xFFFFFFF8
}

func (b *Uint8ArrayFieldBuilder) PutThunk(thunk []uint8) {
	if b.IsPresent() {
		binary.LittleEndian.PutUint16(thunk[2:4], 0xC000)
		binary.LittleEndian.PutUint32(thunk[4:8], b.valueSize)
	}
}

func (b *Uint8ArrayFieldBuilder) EncodeData(w io.Writer) error {
	if !b.IsPresent() {
		return nil
	}

	if value, ok := b.value.(string); ok {
		if _, err := io.WriteString(w, value); err != nil {
			return err
		}
	} else {
		if _, err := w.Write(b.value.([]uint8)); err != nil {
			return err
		}
	}

	// FIXME
	var err error
	paddedSize := (b.valueSize + 0b111) & 0xFFFFFFF8
	if paddedSize > b.valueSize {
		padLen := paddedSize - b.valueSize
		for ii := uint32(0); ii < padLen; ii++ {
			_, err = w.Write([]byte{0x00})
		}
	}
	return err
}

func (b *Uint8ArrayFieldBuilder) Set(value Uint8Array) {
	b.value = value.buf
	b.valueSize = uint32(len(value.buf))
}

func (b *Uint8ArrayFieldBuilder) SetBytes(value []uint8) {
	b.value = value
	b.valueSize = uint32(len(value))
}

func (b *Uint8ArrayFieldBuilder) SetString(value string) {
	b.value = value
	b.valueSize = uint32(len(value))
}

func (b *Uint8ArrayFieldBuilder) Extend(values Uint8Array) {
	if b.value == nil {
		b.Set(values)
		return
	}
	if prev, ok := b.value.(string); ok {
		value := prev + values.buf
		b.value = value
		b.valueSize = uint32(len(value))
	} else {
		var buf strings.Builder
		buf.Write(b.value.([]uint8))
		buf.WriteString(values.buf)
		value := buf.String()
		b.value = value
		b.valueSize = uint32(len(value))
	}
}

// }}}

// Uint16FieldBuilder {{{

type Uint16FieldBuilder struct {
	value uint16
}

func (b *Uint16FieldBuilder) IsPresent() bool {
	return b.value != 0
}

func (b *Uint16FieldBuilder) PutThunk(thunk []uint8) {
	if b.IsPresent() {
		binary.LittleEndian.PutUint16(thunk[2:4], 0x8000)
		binary.LittleEndian.PutUint32(thunk[4:8], uint32(b.value))
	}
}

func (b *Uint16FieldBuilder) Get() uint16 {
	return b.value
}

func (b *Uint16FieldBuilder) Set(value uint16) {
	b.value = value
}

// }}}

// Uint32FieldBuilder {{{

type Uint32FieldBuilder struct {
	value uint32
}

func (b *Uint32FieldBuilder) IsPresent() bool {
	return b.value != 0
}

func (b *Uint32FieldBuilder) PutThunk(thunk []uint8) {
	if b.IsPresent() {
		binary.LittleEndian.PutUint16(thunk[2:4], 0x8000)
		binary.LittleEndian.PutUint32(thunk[4:8], b.value)
	}
}

func (b *Uint32FieldBuilder) Get() uint32 {
	return b.value
}

func (b *Uint32FieldBuilder) Set(value uint32) {
	b.value = value
}

// }}}

// Uint64FieldBuilder {{{

type Uint64FieldBuilder struct {
	value uint64
}

func (b *Uint64FieldBuilder) IsPresent() bool {
	return b.value != 0
}

func (b *Uint64FieldBuilder) DataSize() uint32 {
	if b.value == 0 {
		return 0
	}
	return 8
}

func (b *Uint64FieldBuilder) PutThunk(thunk []uint8) {
	if b.IsPresent() {
		binary.LittleEndian.PutUint16(thunk[2:4], 0xC000)
		binary.LittleEndian.PutUint32(thunk[4:8], 8)
	}
}

func (b *Uint64FieldBuilder) EncodeData(w io.Writer) error {
	if !b.IsPresent() {
		return nil
	}

	tmp := make([]uint8, 8)
	binary.LittleEndian.PutUint64(tmp, b.value)
	_, err := w.Write(tmp)
	return err
}

func (b *Uint64FieldBuilder) Get() uint64 {
	return b.value
}

func (b *Uint64FieldBuilder) Set(value uint64) {
	b.value = value
}

// }}}

// TextFieldBuilder {{{

type TextFieldBuilder struct {
	value Text
}

func (b *TextFieldBuilder) IsPresent() bool {
	return len(b.value) > 0
}

func (b *TextFieldBuilder) DataSize() uint32 {
	if !b.IsPresent() {
		return 0
	}

	size := uint32(len(b.value)) + 1
	if size%8 != 0 {
		size = (size + 0b111) & 0xFFFFFFF8
	}
	return size
}

func (b *TextFieldBuilder) PutThunk(thunk []uint8) {
	if b.IsPresent() {
		binary.LittleEndian.PutUint16(thunk[2:4], 0xC000)
		binary.LittleEndian.PutUint32(thunk[4:8], uint32(len(b.value)+1))
	}
}

func (b *TextFieldBuilder) EncodeData(w io.Writer) error {
	if !b.IsPresent() {
		return nil
	}

	if _, err := io.WriteString(w, b.value); err != nil {
		return err
	}

	// FIXME
	size := uint32(len(b.value))
	paddedSize := size + 1
	if paddedSize%8 != 0 {
		paddedSize = ((size + 1) + 0b111) & 0xFFFFFFF8
	}
	padLen := paddedSize - size
	var err error
	for ii := uint32(0); ii < padLen; ii++ {
		_, err = w.Write([]byte{0x00})
	}
	return err
}

func (b *TextFieldBuilder) Get() Text {
	return b.value
}

func (b *TextFieldBuilder) Set(value Text) {
	b.value = value
}

// }}}

// TextArrayFieldBuilder {{{

type TextArrayFieldBuilder struct {
	values     []string
	valuesSize uint32
}

func (b *TextArrayFieldBuilder) IsPresent() bool {
	return len(b.values) > 0
}

func (b *TextArrayFieldBuilder) DataSize() uint32 {
	if len(b.values) == 0 {
		return 0
	}
	size := 4 + 4*uint32(len(b.values)) + b.valuesSize
	if size%8 != 0 {
		size = (size + 0b111) & 0xFFFFFFF8
	}
	return size
}

func (b *TextArrayFieldBuilder) Add(value Text) {
	b.values = append(b.values, value)
	b.valuesSize += uint32(len(value)) + 1
}

func (b *TextArrayFieldBuilder) Set(values []Text) {
	b.values = append([]Text{}, values...)
	b.valuesSize = 0
	for _, value := range values {
		b.valuesSize += uint32(len(value)) + 1
	}
}

func (b *TextArrayFieldBuilder) Extend(values TextArray) {
	for _, value := range values.Iter() {
		b.values = append(b.values, value)
		b.valuesSize += uint32(len(value)) + 1
	}
}

func (b *TextArrayFieldBuilder) PutThunk(thunk []uint8) {
	if b.IsPresent() {
		binary.LittleEndian.PutUint16(thunk[2:4], 0xC000)
		binary.LittleEndian.PutUint32(thunk[4:8], b.DataSize())
	}
}

func (b *TextArrayFieldBuilder) EncodeData(w io.Writer) error {
	if !b.IsPresent() {
		return nil
	}

	var wrote int
	var sizes bytes.Buffer
	tmp := make([]uint8, 4)
	binary.LittleEndian.PutUint32(tmp, uint32(len(b.values)))
	sizes.Write(tmp)
	for _, value := range b.values {
		binary.LittleEndian.PutUint32(tmp, uint32(len(value))+1)
		sizes.Write(tmp)
	}
	if n, err := w.Write(sizes.Bytes()); err != nil {
		return err
	} else {
		wrote += n
	}
	for _, value := range b.values {
		if n, err := io.WriteString(w, value); err != nil {
			return err
		} else {
			wrote += n
		}
		if n, err := w.Write([]byte{0x00}); err != nil {
			return err
		} else {
			wrote += n
		}
	}
	// FIXME
	if wrote%8 != 0 {
		for ii := 0; ii < 8-(wrote%8); ii++ {
			w.Write([]byte{0x00})
		}
	}
	return nil
}

// }}}

// MessageFieldBuilder {{{

type MessageFieldBuilder[T interface {
	Idol__Message() Message[T]
}] struct {
	value MessageBuilder[T]
}

func (b *MessageFieldBuilder[T]) IsPresent() bool {
	return b.value != nil
}

func (b *MessageFieldBuilder[T]) DataSize() uint32 {
	if b.value == nil {
		return 0
	}
	return b.value.Size()
}

func (b *MessageFieldBuilder[T]) PutThunk(thunk []uint8) {
	if b.value == nil {
		return
	}
	binary.LittleEndian.PutUint16(thunk[2:4], 0xC000)
	binary.LittleEndian.PutUint32(thunk[4:8], b.value.Size())
}

func (b *MessageFieldBuilder[T]) EncodeData(ctx *EncodeCtx, w io.Writer) error {
	if b.value == nil {
		return nil
	}
	return b.value.EncodeTo(ctx, w)
}

func (b *MessageFieldBuilder[T]) Get() AsMessageBuilder[T] {
	return b.value.Self()
}

func (b *MessageFieldBuilder[T]) Set(value AsMessageBuilder[T]) {
	b.value = value.Idol__MessageBuilder()
}

func (b *MessageFieldBuilder[T]) Clear() {
	b.value = nil
}

// }}}

// MessageArrayFieldBuilder {{{

type MessageArrayFieldBuilder[T interface {
	Idol__Message() Message[T]
}] struct {
	values []MessageBuilder[T]
}

func (b *MessageArrayFieldBuilder[T]) IsPresent() bool {
	return len(b.values) > 0
}

func (b *MessageArrayFieldBuilder[T]) DataSize() uint32 {
	if len(b.values) == 0 {
		return 0
	}
	dataSize := 4 + 4*uint32(len(b.values))
	if len(b.values)&0x01 == 0x00 {
		dataSize += 4
	}
	for _, value := range b.values {
		dataSize += value.Size()
	}
	return dataSize
}

func (b *MessageArrayFieldBuilder[T]) PutThunk(thunk []uint8) {
	if b.IsPresent() {
		binary.LittleEndian.PutUint16(thunk[2:4], 0xC000)
		binary.LittleEndian.PutUint32(thunk[4:8], b.DataSize())
	}
}

func (b *MessageArrayFieldBuilder[T]) EncodeData(ctx *EncodeCtx, w io.Writer) error {
	if !b.IsPresent() {
		return nil
	}

	var sizes bytes.Buffer
	tmp := make([]uint8, 4)
	binary.LittleEndian.PutUint32(tmp, uint32(len(b.values)))
	sizes.Write(tmp)
	for _, value := range b.values {
		binary.LittleEndian.PutUint32(tmp, value.Size())
		sizes.Write(tmp)
	}
	if len(b.values)&0x01 == 0x00 {
		sizes.Write([]uint8{0, 0, 0, 0})
	}
	if _, err := w.Write(sizes.Bytes()); err != nil {
		return err
	}
	for _, value := range b.values {
		if err := value.EncodeTo(ctx, w); err != nil {
			return err
		}
	}
	return nil
}

func (b *MessageArrayFieldBuilder[T]) Add(value interface {
	Idol__MessageBuilder() MessageBuilder[T]
}) {
	b.values = append(b.values, value.Idol__MessageBuilder())
}

func (b *MessageArrayFieldBuilder[T]) Extend(values MessageArray[T]) {
	for _, value := range values.Iter() {
		b.values = append(b.values, value.Idol__Message().Clone())
	}
}

func (b *MessageArrayFieldBuilder[T]) Clear() {
	b.values = nil
}

// }}}
