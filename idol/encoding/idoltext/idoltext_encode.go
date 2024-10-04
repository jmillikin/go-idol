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

package idoltext

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"go.idol-lang.org/idol"
)

func Encode[T any](message idol.AsMessage[T]) string {
	var buf strings.Builder
	EncodeTo(message, &buf)
	return buf.String()
}

func EncodeTo[T any](message idol.AsMessage[T], w io.Writer) error {
	e := encoder{w: w}
	e.visitMessage(message.Idol__Message().Fields())
	return e.err
}

type encoder struct {
	w      io.Writer
	indent int
	err    error
}

func (e *encoder) line(s string) {
	if indent := strings.Repeat("\t", e.indent); indent != "" {
		if _, err := io.WriteString(e.w, indent); err != nil {
			e.err = err
			return
		}
	}
	if _, err := io.WriteString(e.w, s); err != nil {
		e.err = err
		return
	}
	if _, err := io.WriteString(e.w, "\n"); err != nil {
		e.err = err
		return
	}
}

func (e *encoder) linef(format string, a ...any) {
	e.line(fmt.Sprintf(format, a...))
}

func (e *encoder) visitMessage(fields idol.MessageFields) {
	for tag, value := range fields.Values() {
		if e.err != nil {
			return
		}
		name := fields.Name(tag)
		e.visitField(name, value)
	}
}

func (e *encoder) visitField(name string, value any) {
	if scalar := fmtScalar(value); scalar != "" {
		e.linef("%s = %s", name, scalar)
		return
	}

	if value, ok := value.(idol.Uint8Array); ok {
		var buf strings.Builder
		for ii, b := range value.Iter() {
			if ii != 0 {
				buf.WriteString(", ")
			}
			fmt.Fprintf(&buf, "0x%02X", b)
		}
		e.linef("%s = [%s]", name, buf.String())
		return
	}

	if value, ok := value.(idol.TextArray); ok {
		e.linef("%s = [", name)
		e.indent += 1
		for _, item := range value.Iter() {
			e.line(quote(item))
		}
		e.indent -= 1
		e.line("]")
		return
	}

	// Idol__Message() Message[T]
	// Fields() MessageFields
	if asMsg := reflect.ValueOf(value).MethodByName("Idol__Message"); asMsg.IsValid() {
		fields := asMsg.Call(nil)[0].Interface().(interface {
			Fields() idol.MessageFields
		}).Fields()

		e.linef("%s = {", name)
		e.indent += 1
		e.visitMessage(fields)
		e.indent -= 1
		e.line("}")
		return
	}

	messages := reflect.ValueOf(value).MethodByName("IterMessages")
	if messages.IsValid() {
		for _, item := range messages.Call(nil)[0].Seq2() {
			fieldsFn := item.MethodByName("Fields")
			if !item.IsValid() {
				panic("fieldsFn not valid")
			}
			itemFieldsValue := fieldsFn.Call(nil)[0]
			itemFields := itemFieldsValue.Interface().(idol.MessageFields)
			e.linef("%s {", name)
			e.indent += 1
			e.visitMessage(itemFields)
			e.indent -= 1
			e.line("}")
		}
		return
	}

	panic(fmt.Sprintf("fmtField: unhandled value %s (%T)", value, value))
}

func fmtScalar(value any) string {
	switch value := value.(type) {
	case bool:
		if value {
			return ".true"
		}
		return ".false"
	case uint8:
		return strconv.FormatUint(uint64(value), 10)
	case uint16:
		return strconv.FormatUint(uint64(value), 10)
	case uint32:
		return strconv.FormatUint(uint64(value), 10)
	case uint64:
		return strconv.FormatUint(value, 10)
	case int8:
		return strconv.FormatInt(int64(value), 10)
	case int16:
		return strconv.FormatInt(int64(value), 10)
	case int32:
		return strconv.FormatInt(int64(value), 10)
	case int64:
		return strconv.FormatInt(value, 10)
	case string:
		return quote(value)
	}

	switch reflect.TypeOf(value).Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fallthrough
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf(".%s", value)
	}

	return ""
}

func quote(text string) string {
	var buf strings.Builder
	buf.WriteByte('"')
	for _, c := range text {
		if c == '\\' || c == '"' {
			buf.WriteByte('\\')
			buf.WriteRune(c)
			continue
		}
		if c == '\t' {
			buf.WriteString("\\t")
			continue
		}
		if c == '\n' {
			buf.WriteString("\\n")
			continue
		}
		if c < 0x20 || c == 0x7F {
			fmt.Fprintf(&buf, "\\x%02X", c)
			continue
		}
		buf.WriteRune(c)
	}
	buf.WriteByte('"')
	return buf.String()
}
