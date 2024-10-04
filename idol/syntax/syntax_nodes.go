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

package syntax

import (
	"bytes"
	"iter"
	"math"
	"slices"
	"strconv"
	"strings"
)

type Span struct {
	start, len uint32
}

func NewSpan(start, len uint32) Span {
	return Span{start, len}
}

func (s *Span) Start() uint32 {
	return s.start
}

func (s *Span) End() uint32 {
	return s.start + s.len
}

func (s *Span) Len() uint32 {
	return s.len
}

type Node interface {
	Span() Span

	ChildNodes() iter.Seq[Node]

	privChildren() []Node

	UnparseTo(buf *bytes.Buffer)
}

func Unparse(node Node) string {
	var buf bytes.Buffer
	node.UnparseTo(&buf)
	return buf.String()
}

func Walk(node Node, walkFn func(Node) bool) {
	if node == nil || !walkFn(node) {
		return
	}
	for _, child := range node.privChildren() {
		Walk(child, walkFn)
	}
	walkFn(nil)
}

func iterChildren(childNodes []Node) iter.Seq[Node] {
	return func(yield func(Node) bool) {
		for _, child := range childNodes {
			if !yield(child) {
				return
			}
		}
	}
}

type leafNode struct{}

func (*leafNode) ChildNodes() iter.Seq[Node] {
	return func(_yield func(Node) bool) {}
}

func (*leafNode) privChildren() []Node {
	return nil
}

type ParseError struct {
	leafNode
	span Span
	err  error
}

var _ Node = (*ParseError)(nil)

func (e *ParseError) Span() Span {
	return e.span
}

func (e *ParseError) UnparseTo(buf *bytes.Buffer) {
	panic("Error.UnparseTo: unimplemented")
}

func (e *ParseError) Get() error {
	return e.err
}

type Space struct {
	leafNode
	raw   string
	start uint32
}

var _ Node = (*Space)(nil)

func (n *Space) Span() Span {
	return Span{
		start: n.start,
		len:   uint32(len(n.raw)),
	}
}

func (n *Space) UnparseTo(buf *bytes.Buffer) {
	buf.WriteString(n.raw)
}

type Newline struct {
	leafNode
	start uint32
	crlf  bool
}

var _ Node = (*Newline)(nil)

func (n *Newline) Span() Span {
	var len uint32
	if n.crlf {
		len = 2
	} else {
		len = 1
	}
	return Span{
		start: n.start,
		len:   len,
	}
}

func (n *Newline) UnparseTo(buf *bytes.Buffer) {
	if n.crlf {
		buf.WriteString("\r\n")
	} else {
		buf.WriteByte('\n')
	}
}

type Comment struct {
	leafNode
	raw   string
	start uint32
}

var _ Node = (*Comment)(nil)

func (n *Comment) Span() Span {
	return Span{
		start: n.start,
		len:   uint32(len(n.raw)),
	}
}

func (n *Comment) UnparseTo(buf *bytes.Buffer) {
	buf.WriteString(n.raw)
}

func (n *Comment) Text() string {
	return n.raw
}

func (n *Comment) IsDocComment() bool {
	return strings.HasPrefix(n.raw, "##")
}

type IntLit struct {
	leafNode
	raw   string
	value uint64
	start uint32
}

var _ Node = (*IntLit)(nil)

func (n *IntLit) Span() Span {
	return Span{
		start: n.start,
		len:   uint32(len(n.raw)),
	}
}

func (n *IntLit) UnparseTo(buf *bytes.Buffer) {
	buf.WriteString(n.raw)
}

func newIntLit(token string, kind TokenKind, start uint32) (*IntLit, error) {
	base := 10
	valueStr := token
	if valueStr[0] == '-' {
		valueStr = valueStr[1:]
	}
	switch kind {
	case T_BIN_INT_LIT:
		base = 2
		valueStr = valueStr[2:]
	case T_OCT_INT_LIT:
		base = 8
		valueStr = valueStr[2:]
	case T_DEC_INT_LIT:
		base = 10
		valueStr = valueStr[2:]
	case T_HEX_INT_LIT:
		base = 16
		valueStr = valueStr[2:]
	}

	value, err := strconv.ParseUint(valueStr, base, 64)
	if err != nil {
		return nil, errIntLitTooPositive(token, start)
	}
	if token[0] == '-' {
		if value > (uint64(math.MaxInt64) + 1) {
			return nil, errIntLitTooNegative(token, start)
		}
		value = uint64(-int64(value))
	}

	return &IntLit{
		raw:   token,
		value: value,
		start: start,
	}, nil
}

func newUnsignedIntLit(token string, kind TokenKind, start uint32) (*IntLit, error) {
	base := 10
	valueStr := token
	switch kind {
	case T_BIN_INT_LIT:
		base = 2
		valueStr = valueStr[2:]
	case T_OCT_INT_LIT:
		base = 8
		valueStr = valueStr[2:]
	case T_DEC_INT_LIT:
		base = 10
		valueStr = valueStr[2:]
	case T_HEX_INT_LIT:
		base = 16
		valueStr = valueStr[2:]
	}

	value, err := strconv.ParseUint(valueStr, base, 64)
	if err != nil {
		return nil, errIntLitTooPositive(token, start)
	}
	return &IntLit{
		raw:   token,
		value: value,
		start: start,
	}, nil
}

func (n *IntLit) GetUint8() (uint8, bool) {
	if n.raw[0] != '-' && n.value <= math.MaxUint8 {
		return uint8(n.value), true
	}
	return 0, false
}

func (n *IntLit) GetUint16() (uint16, bool) {
	if n.raw[0] != '-' && n.value <= math.MaxUint16 {
		return uint16(n.value), true
	}
	return 0, false
}

func (n *IntLit) GetUint32() (uint32, bool) {
	if n.raw[0] != '-' && n.value <= math.MaxUint32 {
		return uint32(n.value), true
	}
	return 0, false
}

func (n *IntLit) GetUint64() (uint64, bool) {
	if n.raw[0] != '-' {
		return n.value, true
	}
	return 0, false
}

func (n *IntLit) GetInt8() (int8, bool) {
	if n.raw[0] == '-' {
		v := int64(n.value)
		if v >= math.MinInt8 && v <= math.MaxInt8 {
			return int8(v), true
		}
	}
	if n.value <= math.MaxInt8 {
		return int8(n.value), true
	}
	return 0, false
}

func (n *IntLit) GetInt16() (int16, bool) {
	if n.raw[0] == '-' {
		v := int64(n.value)
		if v >= math.MinInt16 && v <= math.MaxInt16 {
			return int16(v), true
		}
	}
	if n.value <= math.MaxInt16 {
		return int16(n.value), true
	}
	return 0, false
}

func (n *IntLit) GetInt32() (int32, bool) {
	if n.raw[0] == '-' {
		v := int64(n.value)
		if v >= math.MinInt32 && v <= math.MaxInt32 {
			return int32(v), true
		}
	}
	if n.value <= math.MaxInt32 {
		return int32(n.value), true
	}
	return 0, false
}

func (n *IntLit) GetInt64() (int64, bool) {
	if n.raw[0] == '-' || n.value <= math.MaxInt64 {
		return int64(n.value), true
	}
	return 0, false
}

type TextLit struct {
	leafNode
	raw        string
	value      string
	start      uint32
	validAsciz bool
	validText  bool
}

var _ Node = (*TextLit)(nil)

func (n *TextLit) Span() Span {
	return Span{
		start: n.start,
		len:   uint32(len(n.raw)),
	}
}

func (n *TextLit) UnparseTo(buf *bytes.Buffer) {
	buf.WriteString(n.raw)
}

func newTextLit(token string, start uint32, flags uint8) (*TextLit, error) {
	value := token[1 : len(token)-1]
	if flags&tokenFlagTextHasNoEscapes != 0 {
		return &TextLit{
			raw:        token,
			value:      value,
			start:      start,
			validAsciz: true,
			validText:  true,
		}, nil
	}

	invalid := func() (*TextLit, error) {
		return nil, errTextLitInvalid(start, token)
	}

	var buf bytes.Buffer
	escaped := false
	validAsciz := true
	validText := true
	for len(value) > 0 {
		c := value[0]
		if !escaped {
			if c == 0x5C {
				escaped = true
			} else {
				buf.WriteByte(c)
			}
			value = value[1:]
			continue
		}
		escaped = false

		switch c {
		case 0x22, 0x5C:
			buf.WriteByte(c)
			value = value[1:]
		case 0x6E:
			buf.WriteByte(0x0A)
			value = value[1:]
		case 0x74:
			buf.WriteByte(0x09)
			value = value[1:]
		case 0x78:
			if len(value) < 3 {
				return invalid()
			}
			b, err := strconv.ParseUint(value[1:3], 16, 8)
			if err != nil {
				return invalid()
			}
			if b == 0 {
				validAsciz = false
				validText = false
			}
			if b > 0x7F {
				validText = false
			}
			buf.WriteByte(uint8(b))
			value = value[3:]
		case 0x75:
			value = value[1:]
			if len(value) == 0 || value[0] != 0x7B {
				return invalid()
			}
			value = value[1:]

			var hex string
			hexEnd := false
			for ii, hc := range value {
				if hc == 0x7D {
					hex = value[:ii]
					value = value[ii:]
					hexEnd = true
					break
				}
			}
			if !hexEnd || len(hex) == 0 || len(hex) > 6 {
				return invalid()
			}

			if len(value) == 0 || value[0] != 0x7D {
				return invalid()
			}
			value = value[1:]

			scalar, err := strconv.ParseUint(hex, 16, 32)
			if err != nil {
				return invalid()
			}
			if scalar == 0 {
				validAsciz = false
				validText = false
			}
			if scalar > 0x10FFFF {
				return invalid()
			}

			buf.WriteRune(rune(scalar))
		default:
			return invalid()
		}
	}
	if escaped {
		return invalid()
	}
	return &TextLit{
		raw:        token,
		value:      buf.String(),
		start:      start,
		validAsciz: validAsciz,
		validText:  validText,
	}, nil
}

func (n *TextLit) IsAsciz() bool {
	return true
}

func (n *TextLit) IsText() bool {
	return n.validText
}

func (n *TextLit) GetAsciz() (string, bool) {
	if !n.validAsciz {
		return "", false
	}
	return n.value, true
}

func (n *TextLit) GetText() (string, bool) {
	if !n.validText {
		return "", false
	}
	return n.value, true
}

type Sigil struct {
	leafNode
	raw   byte
	start uint32
}

var _ Node = (*Sigil)(nil)

func (n *Sigil) Span() Span {
	return Span{
		start: n.start,
		len:   1,
	}
}

func (n *Sigil) UnparseTo(buf *bytes.Buffer) {
	buf.WriteByte(n.raw)
}

type Ident struct {
	leafNode
	raw   string
	start uint32
}

var _ Node = (*Ident)(nil)

func (n *Ident) Span() Span {
	return Span{
		start: n.start,
		len:   uint32(len(n.raw)),
	}
}

func (n *Ident) UnparseTo(buf *bytes.Buffer) {
	buf.WriteString(n.raw)
}

func (n *Ident) Get() string {
	return n.raw
}

type Keyword struct {
	leafNode
	raw   string
	start uint32
}

var _ Node = (*Keyword)(nil)

func (n *Keyword) Span() Span {
	return Span{
		start: n.start,
		len:   uint32(len(n.raw)),
	}
}

func (n *Keyword) UnparseTo(buf *bytes.Buffer) {
	buf.WriteString(n.raw)
}

type TypeName struct {
	span       Span
	childNodes []Node

	scope *Ident
	name  *Ident
}

var _ Node = (*TypeName)(nil)

func (n *TypeName) Span() Span {
	return n.span
}

func (n *TypeName) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *TypeName) privChildren() []Node {
	return n.childNodes
}

func (n *TypeName) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *TypeName) Scope() *Ident {
	return n.scope
}

func (n *TypeName) Name() *Ident {
	return n.name
}

type ValueName struct {
	span       Span
	childNodes []Node

	scope *Ident
	name  *Ident
}

var _ Node = (*ValueName)(nil)

func (n *ValueName) Span() Span {
	return n.span
}

func (n *ValueName) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *ValueName) privChildren() []Node {
	return n.childNodes
}

func (n *ValueName) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *ValueName) Scope() *Ident {
	return n.scope
}

func (n *ValueName) Name() *Ident {
	return n.name
}

type ExportName struct {
	span       Span
	childNodes []Node

	scope *Ident
	name  *Ident
}

var _ Node = (*ExportName)(nil)

func (n *ExportName) Span() Span {
	return n.span
}

func (n *ExportName) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *ExportName) privChildren() []Node {
	return n.childNodes
}

func (n *ExportName) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *ExportName) Scope() *Ident {
	return n.scope
}

func (n *ExportName) Name() *Ident {
	return n.name
}

type Tag struct {
	span       Span
	childNodes []Node
	value      *IntLit
}

var _ Node = (*Tag)(nil)

func (n *Tag) Span() Span {
	return n.span
}

func (n *Tag) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *Tag) privChildren() []Node {
	return n.childNodes
}

func (n *Tag) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *Tag) Value() *IntLit {
	return n.value
}

type Schema struct {
	span       Span
	childNodes []Node
}

var _ Node = (*Schema)(nil)

func (n *Schema) Span() Span {
	return n.span
}

func (n *Schema) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *Schema) privChildren() []Node {
	return n.childNodes
}

func (n *Schema) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

type Namespace struct {
	span       Span
	childNodes []Node
	namespace  *TextLit
}

var _ Node = (*Namespace)(nil)

func (n *Namespace) Span() Span {
	return n.span
}

func (n *Namespace) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *Namespace) privChildren() []Node {
	return n.childNodes
}

func (n *Namespace) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *Namespace) Namespace() *TextLit {
	return n.namespace
}

type Import struct {
	span       Span
	childNodes []Node

	namespace   *TextLit
	importAs    *Ident
	importNames []*Ident
}

var _ Node = (*Import)(nil)

func (n *Import) Span() Span {
	return n.span
}

func (n *Import) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *Import) privChildren() []Node {
	return n.childNodes
}

func (n *Import) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *Import) Namespace() *TextLit {
	return n.namespace
}

func (n *Import) ImportAs() *Ident {
	return n.importAs
}

func (n *Import) ImportNames() iter.Seq[*Ident] {
	return slices.Values(n.importNames)
}

type Export struct {
	span       Span
	childNodes []Node

	exportAs    exportAs
	exportNames []*ExportName
}

type exportAs struct {
	exportName *ExportName
	name       *Ident
}

var _ Node = (*Export)(nil)

func (n *Export) Span() Span {
	return n.span
}

func (n *Export) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *Export) privChildren() []Node {
	return n.childNodes
}

func (n *Export) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *Export) ExportAs() (*ExportName, *Ident) {
	return n.exportAs.exportName, n.exportAs.name
}

func (n *Export) ExportNames() iter.Seq[*ExportName] {
	return slices.Values(n.exportNames)
}

type Decorator struct {
	span       Span
	childNodes []Node
	value      Node
}

var _ Node = (*Decorator)(nil)

func (n *Decorator) Span() Span {
	return n.span
}

func (n *Decorator) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *Decorator) privChildren() []Node {
	return n.childNodes
}

func (n *Decorator) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *Decorator) GetOptions() *Options {
	if value, ok := n.value.(*Options); ok {
		return value
	}
	return nil
}

func (n *Decorator) GetOption() *Option {
	if value, ok := n.value.(*Option); ok {
		return value
	}
	return nil
}

type Options struct {
	span       Span
	childNodes []Node
	schema     *TypeName
	options    []*OptionsOption
}

var _ Node = (*Options)(nil)

func (n *Options) Span() Span {
	return n.span
}

func (n *Options) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *Options) privChildren() []Node {
	return n.childNodes
}

func (n *Options) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *Options) Schema() *TypeName {
	return n.schema
}

func (n *Options) Options() iter.Seq[*OptionsOption] {
	return slices.Values(n.options)
}

type OptionsOption struct {
	span       Span
	childNodes []Node
	name       *OptionName
	value      Node
}

var _ Node = (*OptionsOption)(nil)

func (n *OptionsOption) Span() Span {
	return n.span
}

func (n *OptionsOption) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *OptionsOption) privChildren() []Node {
	return n.childNodes
}

func (n *OptionsOption) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *OptionsOption) Name() *OptionName {
	return n.name
}

func (n *OptionsOption) Value() Node {
	return n.value
}

type Option struct {
	span       Span
	childNodes []Node
	name       *OptionName
	value      Node
}

var _ Node = (*Option)(nil)

func (n *Option) Span() Span {
	return n.span
}

func (n *Option) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *Option) privChildren() []Node {
	return n.childNodes
}

func (n *Option) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *Option) Name() *OptionName {
	return n.name
}

func (n *Option) Value() Node {
	return n.value
}

type OptionName struct {
	span       Span
	childNodes []Node
}

var _ Node = (*OptionName)(nil)

func (n *OptionName) Span() Span {
	return n.span
}

func (n *OptionName) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *OptionName) privChildren() []Node {
	return n.childNodes
}

func (n *OptionName) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

type Const struct {
	span       Span
	childNodes []Node
	name       *Ident
	typeName   *TypeName
	value      Node
	decorators []*Decorator
}

var _ Node = (*Const)(nil)

func (n *Const) Span() Span {
	return n.span
}

func (n *Const) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *Const) privChildren() []Node {
	return n.childNodes
}

func (n *Const) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *Const) Name() *Ident {
	return n.name
}

func (n *Const) TypeName() *TypeName {
	return n.typeName
}

func (n *Const) Value() Node {
	return n.value
}

func (n *Const) Decorators() []*Decorator {
	return n.decorators
}

func (n *Const) setDecorators(decorators []*Decorator) {
	n.decorators = decorators
}

type EnumRef struct {
	span       Span
	childNodes []Node

	name *Ident
}

var _ Node = (*EnumRef)(nil)

func (n *EnumRef) Span() Span {
	return n.span
}

func (n *EnumRef) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *EnumRef) privChildren() []Node {
	return n.childNodes
}

func (n *EnumRef) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *EnumRef) Name() *Ident {
	return n.name
}

type Enum struct {
	span       Span
	childNodes []Node
	name       *Ident
	type_      *Ident
	items      []*EnumItem
	decorators []*Decorator
}

var _ Node = (*Enum)(nil)

func (n *Enum) Span() Span {
	return n.span
}

func (n *Enum) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *Enum) privChildren() []Node {
	return n.childNodes
}

func (n *Enum) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *Enum) Name() *Ident {
	return n.name
}

func (n *Enum) Type() *Ident {
	return n.type_
}

func (n *Enum) Items() []*EnumItem {
	return n.items
}

func (n *Enum) Decorators() []*Decorator {
	return n.decorators
}

func (n *Enum) setDecorators(decorators []*Decorator) {
	n.decorators = decorators
}

type EnumItem struct {
	span       Span
	childNodes []Node
	name       *Ident
	value      Node
	decorators []*Decorator
}

var _ Node = (*EnumItem)(nil)

func (n *EnumItem) Span() Span {
	return n.span
}

func (n *EnumItem) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *EnumItem) privChildren() []Node {
	return n.childNodes
}

func (n *EnumItem) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *EnumItem) Name() *Ident {
	return n.name
}

func (n *EnumItem) Value() Node {
	return n.value
}

func (n *EnumItem) Decorators() []*Decorator {
	return n.decorators
}

func (n *EnumItem) setDecorators(decorators []*Decorator) {
	n.decorators = decorators
}

type Struct struct {
	span       Span
	childNodes []Node
	name       *Ident
	decorators []*Decorator
	fields     []*StructField
}

var _ Node = (*Struct)(nil)

func (n *Struct) Span() Span {
	return n.span
}

func (n *Struct) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *Struct) privChildren() []Node {
	return n.childNodes
}

func (n *Struct) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *Struct) Name() *Ident {
	return n.name
}

func (n *Struct) Decorators() []*Decorator {
	return n.decorators
}

func (n *Struct) setDecorators(decorators []*Decorator) {
	n.decorators = decorators
}

func (n *Struct) Fields() []*StructField {
	return n.fields
}

type StructField struct {
	span       Span
	childNodes []Node
	name       *Ident
	fieldType  *FieldType
	decorators []*Decorator
}

var _ Node = (*StructField)(nil)

func (n *StructField) Span() Span {
	return n.span
}

func (n *StructField) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *StructField) privChildren() []Node {
	return n.childNodes
}

func (n *StructField) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *StructField) Name() *Ident {
	return n.name
}

func (n *StructField) FieldType() *FieldType {
	return n.fieldType
}

func (n *StructField) Decorators() []*Decorator {
	return n.decorators
}

func (n *StructField) setDecorators(decorators []*Decorator) {
	n.decorators = decorators
}

type Message struct {
	span       Span
	childNodes []Node
	name       *Ident
	decorators []*Decorator
	fields     []*MessageField
}

var _ Node = (*Message)(nil)

func (n *Message) Span() Span {
	return n.span
}

func (n *Message) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *Message) privChildren() []Node {
	return n.childNodes
}

func (n *Message) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *Message) Name() *Ident {
	return n.name
}

func (n *Message) Decorators() []*Decorator {
	return n.decorators
}

func (n *Message) setDecorators(decorators []*Decorator) {
	n.decorators = decorators
}

func (n *Message) Fields() []*MessageField {
	return n.fields
}

type MessageField struct {
	span       Span
	childNodes []Node
	name       *Ident
	tag        *Tag
	fieldType  *FieldType
	decorators []*Decorator
}

var _ Node = (*MessageField)(nil)

func (n *MessageField) Span() Span {
	return n.span
}

func (n *MessageField) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *MessageField) privChildren() []Node {
	return n.childNodes
}

func (n *MessageField) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *MessageField) Name() *Ident {
	return n.name
}

func (n *MessageField) Tag() *Tag {
	return n.tag
}

func (n *MessageField) FieldType() *FieldType {
	return n.fieldType
}

func (n *MessageField) Decorators() []*Decorator {
	return n.decorators
}

func (n *MessageField) setDecorators(decorators []*Decorator) {
	n.decorators = decorators
}

type Union struct {
	span       Span
	childNodes []Node
	name       *Ident
	decorators []*Decorator
	fields     []*UnionField
}

var _ Node = (*Union)(nil)

func (n *Union) Span() Span {
	return n.span
}

func (n *Union) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *Union) privChildren() []Node {
	return n.childNodes
}

func (n *Union) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *Union) Name() *Ident {
	return n.name
}

func (n *Union) Decorators() []*Decorator {
	return n.decorators
}

func (n *Union) setDecorators(decorators []*Decorator) {
	n.decorators = decorators
}

func (n *Union) Fields() []*UnionField {
	return n.fields
}

type UnionField struct {
	span       Span
	childNodes []Node
	name       *Ident
	tag        *Tag
	fieldType  *FieldType
	decorators []*Decorator
}

var _ Node = (*UnionField)(nil)

func (n *UnionField) Span() Span {
	return n.span
}

func (n *UnionField) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *UnionField) privChildren() []Node {
	return n.childNodes
}

func (n *UnionField) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *UnionField) Name() *Ident {
	return n.name
}

func (n *UnionField) Tag() *Tag {
	return n.tag
}

func (n *UnionField) FieldType() *FieldType {
	return n.fieldType
}

func (n *UnionField) Decorators() []*Decorator {
	return n.decorators
}

func (n *UnionField) setDecorators(decorators []*Decorator) {
	n.decorators = decorators
}

type FieldType struct {
	span       Span
	childNodes []Node

	typeName *TypeName
	isArray  bool
	arrayLen *IntLit
}

var _ Node = (*FieldType)(nil)

func (n *FieldType) Span() Span {
	return n.span
}

func (n *FieldType) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *FieldType) privChildren() []Node {
	return n.childNodes
}

func (n *FieldType) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *FieldType) TypeName() *TypeName {
	return n.typeName
}

func (n *FieldType) IsArray() bool {
	return n.isArray
}

func (n *FieldType) ArrayLen() *IntLit {
	return n.arrayLen
}

type Protocol struct {
	span       Span
	childNodes []Node
	name       *Ident
	decorators []*Decorator
	rpcs       []*ProtocolRpc
	events     []*ProtocolEvent
}

var _ Node = (*Protocol)(nil)

func (n *Protocol) Span() Span {
	return n.span
}

func (n *Protocol) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *Protocol) privChildren() []Node {
	return n.childNodes
}

func (n *Protocol) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *Protocol) Name() *Ident {
	return n.name
}

func (n *Protocol) Decorators() []*Decorator {
	return n.decorators
}

func (n *Protocol) setDecorators(decorators []*Decorator) {
	n.decorators = decorators
}

func (n *Protocol) Rpcs() iter.Seq[*ProtocolRpc] {
	return slices.Values(n.rpcs)
}

func (n *Protocol) Events() iter.Seq[*ProtocolEvent] {
	return slices.Values(n.events)
}

type ProtocolRpc struct {
	span             Span
	childNodes       []Node
	name             *Ident
	tag              *Tag
	requestType      *TypeName
	requestIsStream  bool
	responseType     *TypeName
	responseIsStream bool
	decorators       []*Decorator
}

var _ Node = (*ProtocolRpc)(nil)

func (n *ProtocolRpc) Span() Span {
	return n.span
}

func (n *ProtocolRpc) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *ProtocolRpc) privChildren() []Node {
	return n.childNodes
}

func (n *ProtocolRpc) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *ProtocolRpc) Name() *Ident {
	return n.name
}

func (n *ProtocolRpc) Tag() *Tag {
	return n.tag
}

func (n *ProtocolRpc) RequestType() *TypeName {
	return n.requestType
}

func (n *ProtocolRpc) RequestIsStream() bool {
	return n.requestIsStream
}

func (n *ProtocolRpc) ResponseType() *TypeName {
	return n.responseType
}

func (n *ProtocolRpc) ResponseIsStream() bool {
	return n.responseIsStream
}

func (n *ProtocolRpc) Decorators() []*Decorator {
	return n.decorators
}

func (n *ProtocolRpc) setDecorators(decorators []*Decorator) {
	n.decorators = decorators
}

type ProtocolEvent struct {
	span        Span
	childNodes  []Node
	name        *Ident
	tag         *Tag
	payloadType *TypeName
	decorators  []*Decorator
}

var _ Node = (*ProtocolEvent)(nil)

func (n *ProtocolEvent) Span() Span {
	return n.span
}

func (n *ProtocolEvent) ChildNodes() iter.Seq[Node] {
	return iterChildren(n.childNodes)
}

func (n *ProtocolEvent) privChildren() []Node {
	return n.childNodes
}

func (n *ProtocolEvent) UnparseTo(buf *bytes.Buffer) {
	for _, childNode := range n.childNodes {
		childNode.UnparseTo(buf)
	}
}

func (n *ProtocolEvent) Name() *Ident {
	return n.name
}

func (n *ProtocolEvent) Tag() *Tag {
	return n.tag
}

func (n *ProtocolEvent) PayloadType() *TypeName {
	return n.payloadType
}

func (n *ProtocolEvent) Decorators() []*Decorator {
	return n.decorators
}

func (n *ProtocolEvent) setDecorators(decorators []*Decorator) {
	n.decorators = decorators
}
