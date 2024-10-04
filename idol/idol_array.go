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
	"fmt"
	"iter"
	"strconv"
	"strings"
	"unsafe"
)

func intsString[T int8 | int16 | int32 | int64](xs iter.Seq2[uint32, T]) string {
	var buf strings.Builder
	buf.WriteByte('[')
	for ii, x := range xs {
		if ii > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(strconv.FormatInt(int64(x), 10))
	}
	buf.WriteByte(']')
	return buf.String()
}

func uintsString[T uint8 | uint16 | uint32 | uint64](xs iter.Seq2[uint32, T]) string {
	var buf strings.Builder
	buf.WriteByte('[')
	for ii, x := range xs {
		if ii > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(strconv.FormatUint(uint64(x), 10))
	}
	buf.WriteByte(']')
	return buf.String()
}

// BoolArray {{{

type BoolArray struct {
	buf string
}

func (a BoolArray) Len() uint32 {
	return uint32(len(a.buf))
}

func (a BoolArray) Collect() []bool {
	out := make([]bool, len(a.buf))
	for ii, v := range []byte(a.buf) {
		out[ii] = v != 0
	}
	return out
}

func (a BoolArray) Get(idx uint32) (bool, bool) {
	if idx >= a.Len() {
		return false, false
	}
	return a.buf[idx] != 0, true
}

func (a BoolArray) Iter() iter.Seq2[uint32, bool] {
	return func(yield func(uint32, bool) bool) {
		for ii, v := range []byte(a.buf) {
			if !yield(uint32(ii), v != 0) {
				return
			}
		}
	}
}

func (a BoolArray) String() string {
	var out strings.Builder
	out.WriteByte('[')
	for ii, x := range a.Iter() {
		if ii > 0 {
			out.WriteString(", ")
		}
		if x {
			out.WriteString(".true")
		} else {
			out.WriteString(".false")
		}
	}
	out.WriteByte(']')
	return out.String()
}

// }}}

// Uint8Array {{{

type Uint8Array struct {
	buf string
}

func (a Uint8Array) Len() uint32 {
	return uint32(len(a.buf))
}

func (a Uint8Array) Collect() []uint8 {
	return []uint8(a.buf)
}

func (a Uint8Array) Get(idx uint32) (uint8, bool) {
	if idx >= a.Len() {
		return 0, false
	}
	return a.buf[idx], true
}

func (a Uint8Array) Iter() iter.Seq2[uint32, uint8] {
	return func(yield func(uint32, uint8) bool) {
		len := a.Len()
		for ii := uint32(0); ii < len; ii++ {
			if !yield(ii, a.buf[ii]) {
				return
			}
		}
	}
}

func (a Uint8Array) String() string {
	return uintsString(a.Iter())
}

// }}}

// Int8Array {{{

type Int8Array struct {
	buf string
}

func (a Int8Array) Len() uint32 {
	return uint32(len(a.buf))
}

func (a Int8Array) Collect() []int8 {
	out := make([]int8, len(a.buf))
	for ii, v := range []byte(a.buf) {
		out[ii] = int8(v)
	}
	return out
}

func (a Int8Array) Get(idx uint32) (int8, bool) {
	if idx >= a.Len() {
		return 0, false
	}
	return int8(a.buf[idx]), true
}

func (a Int8Array) Iter() iter.Seq2[uint32, int8] {
	return func(yield func(uint32, int8) bool) {
		len := a.Len()
		for ii := uint32(0); ii < len; ii++ {
			if !yield(ii, int8(a.buf[ii])) {
				return
			}
		}
	}
}

func (a Int8Array) String() string {
	return intsString(a.Iter())
}

// }}}

// Uint16Array {{{

type Uint16Array struct {
	buf string
}

func (a Uint16Array) Len() uint32 {
	return uint32(len(a.buf) / 2)
}

func (a Uint16Array) Collect() []uint16 {
	len := len(a.buf) / 2
	out := make([]uint16, len)
	for ii := 0; ii < len; ii++ {
		off := ii * 2
		out[ii] = leUint16([]byte(a.buf[off : off+2]))
	}
	return out
}

func (a Uint16Array) Get(idx uint32) (uint16, bool) {
	if idx > a.Len() {
		return 0, false
	}
	off := idx * 2
	return leUint16([]byte(a.buf[off : off+2])), true
}

func (a Uint16Array) Iter() iter.Seq2[uint32, uint16] {
	return func(yield func(uint32, uint16) bool) {
		len := len(a.buf) / 2
		for ii := 0; ii < len; ii++ {
			off := ii * 2
			value := leUint16([]byte(a.buf[off : off+2]))
			if !yield(uint32(ii), value) {
				return
			}
		}
	}
}

func (a Uint16Array) String() string {
	return uintsString(a.Iter())
}

// }}}

// Int16Array {{{

type Int16Array struct {
	buf string
}

func (a Int16Array) Len() uint32 {
	return uint32(len(a.buf) / 2)
}

func (a Int16Array) Collect() []int16 {
	len := len(a.buf) / 2
	out := make([]int16, len)
	for ii := 0; ii < len; ii++ {
		off := ii * 2
		out[ii] = int16(leUint16([]byte(a.buf[off : off+2])))
	}
	return out
}

func (a Int16Array) Get(idx uint32) (int16, bool) {
	if idx > a.Len() {
		return 0, false
	}
	off := idx * 2
	return int16(leUint16([]byte(a.buf[off : off+2]))), true
}

func (a Int16Array) Iter() iter.Seq2[uint32, int16] {
	return func(yield func(uint32, int16) bool) {
		len := len(a.buf) / 2
		for ii := 0; ii < len; ii++ {
			off := ii * 2
			value := leUint16([]byte(a.buf[off : off+2]))
			if !yield(uint32(ii), int16(value)) {
				return
			}
		}
	}
}

func (a Int16Array) String() string {
	return intsString(a.Iter())
}

// }}}

// Uint32Array {{{

type Uint32Array struct {
	buf string
}

func (a Uint32Array) Len() uint32 {
	return uint32(len(a.buf) / 4)
}

func (a Uint32Array) Collect() []uint32 {
	len := len(a.buf) / 4
	out := make([]uint32, len)
	for ii := 0; ii < len; ii++ {
		off := ii * 4
		out[ii] = leUint32([]byte(a.buf[off : off+4]))
	}
	return out
}

func (a Uint32Array) Get(idx uint32) (uint32, bool) {
	if idx > a.Len() {
		return 0, false
	}
	off := idx * 4
	return leUint32([]byte(a.buf[off : off+4])), true
}

func (a Uint32Array) Iter() iter.Seq2[uint32, uint32] {
	return func(yield func(uint32, uint32) bool) {
		len := len(a.buf) / 4
		for ii := 0; ii < len; ii++ {
			off := ii * 4
			value := leUint32([]byte(a.buf[off : off+4]))
			if !yield(uint32(ii), value) {
				return
			}
		}
	}
}

func (a Uint32Array) String() string {
	return uintsString(a.Iter())
}

// }}}

// Int32Array {{{

type Int32Array struct {
	buf string
}

func (a Int32Array) Len() uint32 {
	return uint32(len(a.buf) / 4)
}

func (a Int32Array) Collect() []int32 {
	len := len(a.buf) / 4
	out := make([]int32, len)
	for ii := 0; ii < len; ii++ {
		off := ii * 4
		out[ii] = int32(leUint32([]byte(a.buf[off : off+4])))
	}
	return out
}

func (a Int32Array) Get(idx uint32) (int32, bool) {
	if idx > a.Len() {
		return 0, false
	}
	off := idx * 4
	return int32(leUint32([]byte(a.buf[off : off+4]))), true
}

func (a Int32Array) Iter() iter.Seq2[uint32, int32] {
	return func(yield func(uint32, int32) bool) {
		len := len(a.buf) / 4
		for ii := 0; ii < len; ii++ {
			off := ii * 4
			value := leUint32([]byte(a.buf[off : off+4]))
			if !yield(uint32(ii), int32(value)) {
				return
			}
		}
	}
}

func (a Int32Array) String() string {
	return intsString(a.Iter())
}

// }}}

// Uint64Array {{{

type Uint64Array struct {
	buf string
}

func (a Uint64Array) Len() uint32 {
	return uint32(len(a.buf) / 8)
}

func (a Uint64Array) Collect() []uint64 {
	len := len(a.buf) / 8
	out := make([]uint64, len)
	for ii := 0; ii < len; ii++ {
		off := ii * 8
		out[ii] = leUint64([]byte(a.buf[off : off+8]))
	}
	return out
}

func (a Uint64Array) Get(idx uint32) (uint64, bool) {
	if idx > a.Len() {
		return 0, false
	}
	off := idx * 8
	return leUint64([]byte(a.buf[off : off+8])), true
}

func (a Uint64Array) Iter() iter.Seq2[uint32, uint64] {
	return func(yield func(uint32, uint64) bool) {
		len := len(a.buf) / 8
		for ii := 0; ii < len; ii++ {
			off := ii * 8
			value := leUint64([]byte(a.buf[off : off+8]))
			if !yield(uint32(ii), value) {
				return
			}
		}
	}
}

func (a Uint64Array) String() string {
	return uintsString(a.Iter())
}

// }}}

// Int64Array {{{

type Int64Array struct {
	buf string
}

func (a Int64Array) Len() uint32 {
	return uint32(len(a.buf) / 8)
}

func (a Int64Array) Collect() []int64 {
	len := len(a.buf) / 8
	out := make([]int64, len)
	for ii := 0; ii < len; ii++ {
		off := ii * 8
		out[ii] = int64(leUint64([]byte(a.buf[off : off+8])))
	}
	return out
}

func (a Int64Array) Get(idx uint32) (int64, bool) {
	if idx > a.Len() {
		return 0, false
	}
	off := idx * 8
	return int64(leUint64([]byte(a.buf[off : off+8]))), true
}

func (a Int64Array) Iter() iter.Seq2[uint32, int64] {
	return func(yield func(uint32, int64) bool) {
		len := len(a.buf) / 8
		for ii := 0; ii < len; ii++ {
			off := ii * 8
			value := leUint64([]byte(a.buf[off : off+8]))
			if !yield(uint32(ii), int64(value)) {
				return
			}
		}
	}
}

func (a Int64Array) String() string {
	return intsString(a.Iter())
}

// }}}

type dynArray string

func (a dynArray) len() uint32 {
	if len(a) == 0 {
		return 0
	}
	return leUint32([]byte(a[0:4]))
}

func (a dynArray) get(align bool, idx uint32) (string, bool) {
	len := a.len()
	if idx >= len {
		return "", false
	}
	sizeOff := 4
	valueOff := 4 + len*4
	if align && len&0x01 == 0x00 {
		valueOff += 4
	}
	for ii := uint32(0); ii < idx; ii++ {
		size := leUint32([]byte(a[sizeOff : sizeOff+4]))
		sizeOff += 4
		valueOff += size
	}
	size := leUint32([]byte(a[sizeOff : sizeOff+4]))
	if size == 0 {
		return "", true
	}
	return string(a)[valueOff : valueOff+size], true
}

func (a dynArray) iter(align bool, yield func(uint32, string) bool) {
	arrayLen := a.len()
	if arrayLen == 0 {
		return
	}
	sizeOff := 4
	valueOff := 4 + arrayLen*4
	if align && arrayLen&0x01 == 0x00 {
		valueOff += 4
	}
	for ii := uint32(0); ii < arrayLen; ii++ {
		size := leUint32([]byte(a[sizeOff : sizeOff+4]))
		value := string(a)[valueOff : valueOff+size]
		sizeOff += 4
		valueOff += size
		if !yield(ii, value) {
			return
		}
	}
}

// AscizArray {{{

type AscizArray struct {
	buf string
}

func (a AscizArray) Len() uint32 {
	return dynArray(a.buf).len()
}

func (a AscizArray) Collect() []Asciz {
	aLen := a.Len()
	if aLen == 0 {
		return []Asciz{}
	}
	out := make([]Asciz, 0, aLen)
	dynArray(a.buf).iter(false, func(_ uint32, value string) bool {
		if len(value) == 0 {
			value = "\x00"
		}
		out = append(out, value)
		return true
	})
	return out
}

func (a AscizArray) Get(idx uint32) (Asciz, bool) {
	value, ok := dynArray(a.buf).get(false, idx)
	if !ok || len(value) == 0 {
		return "\x00", ok
	}
	return value, true
}

func (a AscizArray) Iter() iter.Seq2[uint32, Asciz] {
	return func(yield func(uint32, Asciz) bool) {
		dynArray(a.buf).iter(false, func(idx uint32, value string) bool {
			if len(value) == 0 {
				value = "\x00"
			}
			return yield(idx, value)
		})
	}
}

func (a AscizArray) String() string {
	var buf strings.Builder
	buf.WriteByte('[')
	for ii, x := range a.Iter() {
		if ii > 0 {
			buf.WriteString(", ")
		}
		quoteAsciz(x, &buf)
	}
	buf.WriteByte(']')
	return buf.String()
}

// }}}

// TextArray {{{

type TextArray struct {
	buf string
}

func (a TextArray) Len() uint32 {
	return dynArray(a.buf).len()
}

func (a TextArray) Collect() []Text {
	aLen := a.Len()
	if aLen == 0 {
		return []Text{}
	}
	out := make([]Text, 0, aLen)
	dynArray(a.buf).iter(false, func(_ uint32, value string) bool {
		if len(value) > 0 {
			value = value[:len(value)-1]
		}
		out = append(out, value)
		return true
	})
	return out
}

func (a TextArray) Get(idx uint32) (Text, bool) {
	value, ok := dynArray(a.buf).get(false, idx)
	if !ok || len(value) == 0 {
		return "", ok
	}
	return value[:len(value)-1], true
}

func (a TextArray) Iter() iter.Seq2[uint32, Text] {
	return func(yield func(uint32, Text) bool) {
		dynArray(a.buf).iter(false, func(idx uint32, value string) bool {
			if len(value) > 0 {
				value = value[:len(value)-1]
			}
			return yield(idx, value)
		})
	}
}

func (a TextArray) String() string {
	var buf strings.Builder
	buf.WriteByte('[')
	for ii, x := range a.Iter() {
		if ii > 0 {
			buf.WriteString(", ")
		}
		quoteText(x, &buf)
	}
	buf.WriteByte(']')
	return buf.String()
}

// }}}

// MessageArray {{{

type MessageArray[T any] struct {
	buf string
}

func (a MessageArray[T]) Len() uint32 {
	return dynArray(a.buf).len()
}

func (a MessageArray[T]) Collect() []T {
	aLen := a.Len()
	if aLen == 0 {
		return []T{}
	}
	out := make([]T, 0, aLen)
	dynArray(a.buf).iter(true, func(_ uint32, value string) bool {
		out = append(out, *(*T)(unsafe.Pointer(&value)))
		return true
	})
	return out
}

func (a MessageArray[T]) Get(idx uint32) (T, bool) {
	value, ok := dynArray(a.buf).get(true, idx)
	if !ok {
		var empty T
		return empty, false
	}
	return *(*T)(unsafe.Pointer(&value)), true
}

func (a MessageArray[T]) Iter() iter.Seq2[uint32, T] {
	return func(yield func(uint32, T) bool) {
		dynArray(a.buf).iter(true, func(idx uint32, value string) bool {
			return yield(idx, *(*T)(unsafe.Pointer(&value)))
		})
	}
}

func (a MessageArray[T]) IterMessages() iter.Seq2[uint32, Message[T]] {
	var zero T
	_ = any(zero).(AsMessage[T])
	return func(yield func(uint32, Message[T]) bool) {
		dynArray(a.buf).iter(true, func(idx uint32, value string) bool {
			item := *(*T)(unsafe.Pointer(&value))
			msg := any(item).(AsMessage[T]).Idol__Message()
			return yield(idx, msg)
		})
	}
}

func (a MessageArray[T]) String() string {
	var buf strings.Builder
	buf.WriteByte('[')
	for ii, x := range a.Iter() {
		if ii > 0 {
			buf.WriteString(", ")
		}
		buf.WriteByte('{')
		fmt.Fprintf(&buf, "%s", x)
		buf.WriteByte('}')
	}
	buf.WriteByte(']')
	return buf.String()
}

// }}}

// TODO: unify with `encoding/idoltext`

func quoteAsciz(text string, buf *strings.Builder) string {
	buf.WriteByte('"')
	for _, c := range []byte(text) {
		if c == 0x00 {
			continue
		}
		if c == 0x22 || c == 0x5C {
			buf.WriteByte('\\')
			buf.WriteByte(c)
			continue
		}
		if c == 0x09 {
			buf.WriteString("\\t")
			continue
		}
		if c == 0x0A {
			buf.WriteString("\\n")
			continue
		}
		if c < 0x20 || c >= 0x7F {
			fmt.Fprintf(buf, "\\x%02X", c)
			continue
		}
		buf.WriteByte(c)
	}
	buf.WriteByte('"')
	return buf.String()
}

func quoteText(text string, buf *strings.Builder) string {
	buf.WriteByte('"')
	for _, c := range text {
		if c == 0x22 || c == 0x5C {
			buf.WriteByte('\\')
			buf.WriteRune(c)
			continue
		}
		if c == 0x09 {
			buf.WriteString("\\t")
			continue
		}
		if c == 0x0A {
			buf.WriteString("\\n")
			continue
		}
		if c < 0x20 || c == 0x7F {
			fmt.Fprintf(buf, "\\x%02X", c)
			continue
		}
		buf.WriteRune(c)
	}
	buf.WriteByte('"')
	return buf.String()
}
