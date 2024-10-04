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

package compiler

import (
	"bytes"
	"fmt"
	"math"
	"strings"

	"go.idol-lang.org/idol/schema_idl"
	"go.idol-lang.org/idol/syntax"
)

type Error struct {
	code    uint32
	message string
	span    syntax.Span
}

var _ error = (*Error)(nil)

func (err *Error) Error() string {
	return fmt.Sprintf("E%d: %s", err.code, err.message)
}

func (err *Error) Code() uint32 {
	return err.code
}

func (err *Error) Message() string {
	return err.message
}

func (err *Error) Span() syntax.Span {
	return err.span
}

func errInvalidNamespace(namespace string, span syntax.Span) error {
	return &Error{
		code:    3000,
		message: fmt.Sprintf("Invalid namespace name %q", namespace),
		span:    span,
	}
}

func errImportNamespaceNotFound(ns string, span syntax.Span) error {
	return &Error{
		code:    3001,
		message: fmt.Sprintf("Namespace %q not found in dependencies", ns),
		span:    span,
	}
}

func errImportAsConflict(prevNs, ns, alias string, span syntax.Span) error {
	return &Error{
		code: 3002,
		message: fmt.Sprintf(
			"Import of namespace %q as '%s' conflicts with earlier"+
				" import of namespace %q as '%s'",
			ns, alias, prevNs, alias,
		),
		span: span,
	}
}

func errImportAsNotFound(alias string, span syntax.Span) error {
	return &Error{
		code:    3003,
		message: fmt.Sprintf("No namespace imported as '%s'", alias),
		span:    span,
	}
}

func errImportNameConflict(prevNs, ns, name string, span syntax.Span) error {
	return &Error{
		code: 3004,
		message: fmt.Sprintf(
			"Import of '%s' from namespace %q conflicts with earlier import"+
				" of '%s' from namespace %q",
			name, ns, name, prevNs,
		),
		span: span,
	}
}

func errImportNameNotFound(ns, name string, span syntax.Span) error {
	return &Error{
		code: 3005,
		message: fmt.Sprintf(
			"Name '%s' not found in imported namespace %q",
			name, ns,
		),
		span: span,
	}
}

func errImportNameDefinitionConflict(ns, name string, span syntax.Span) error {
	return &Error{
		code: 3006,
		message: fmt.Sprintf(
			"Name '%s' imported from namespace %q has conflicting definitions",
			name, ns,
		),
		span: span,
	}
}

func errImportedNameNotType(
	got string,
	ns string,
	name string,
	span syntax.Span,
) error {
	return &Error{
		code: 3007,
		message: fmt.Sprintf(
			"Name '%s' imported from namespace %q is a %s, not a type",
			name, ns, got,
		),
		span: span,
	}
}

func errImportedNameNotConst(
	got string,
	ns string,
	name string,
	span syntax.Span,
) error {
	return &Error{
		code: 0x30000, // TODO
		message: fmt.Sprintf(
			"Name '%s' imported from namespace %q is a %s, not a constant",
			name, ns, got,
		),
		span: span,
	}
}

func errOptionNameConflict(name string, node syntax.Node) error {
	return &Error{
		code:    3010,
		message: fmt.Sprintf("Option '%s' already assigned", name),
		span:    node.Span(),
	}
}

// TODO: exportable_name_not_found

// TODO: type_name_not_found

// TODO: option_name_conflict

// TODO: option_value_type_invalid

func declTypeString(decl declNode) string {
	switch decl.(type) {
	case *syntax.Const:
		return "const"
	case *syntax.Enum:
		return "enum"
	case *syntax.Struct:
		return "struct"
	case *syntax.Message:
		return "message"
	case *syntax.Union:
		return "union"
	case *syntax.Protocol:
		return "protocol"
	default:
		panic("unreachable")
	}
}

func errDeclNameConflict(decl, prevDecl declNode) error {
	return &Error{
		code: 3012,
		message: fmt.Sprintf(
			"Declaration of %s '%s' conflicts with earlier declaration of %s '%s'",
			declTypeString(decl),
			decl.Name().Get(),
			declTypeString(prevDecl),
			prevDecl.Name().Get(),
		),
		span: decl.Name().Span(),
	}
}

func errDeclNameConflictsWithImport(decl declNode, namespace string) error {
	return &Error{
		code: 3013,
		message: fmt.Sprintf(
			"Declaration of %s '%s' conflicts with name imported from namespace %q",
			declTypeString(decl), decl.Name().Get(), namespace,
		),
		span: decl.Name().Span(),
	}
}

func errDeclNameConflictsWithImportAs(decl declNode, namespace string) error {
	name := decl.Name()
	return &Error{
		code: 3014,
		message: fmt.Sprintf(
			"Declaration of %s '%s' conflicts with import of namespace %q as '%s'",
			declTypeString(decl), name.Get(), namespace, name.Get(),
		),
		span: name.Span(),
	}
}

func errValueTypeMismatch(
	destination syntax.Node,
	valueType *typeInfo,
	valueNode syntax.Node,
) error {
	var dstNodeType, dstName string
	switch dst := destination.(type) {
	case *syntax.Const:
		dstNodeType = "constant"
		dstName = dst.Name().Get()
	case *syntax.Option:
		dstNodeType = "option"
		var nameBuf bytes.Buffer
		dst.Name().UnparseTo(&nameBuf)
		dstName = nameBuf.String()
	case *syntax.OptionsOption:
		dstNodeType = "option"
		var nameBuf bytes.Buffer
		dst.Name().UnparseTo(&nameBuf)
		dstName = nameBuf.String()
	default:
		panic("unreachable")
	}

	var valueBuf bytes.Buffer
	valueNode.UnparseTo(&valueBuf)
	value := valueBuf.String()

	typeName := valueType.typeName
	if typeName != "" {
		if namespace, local, ok := strings.Cut(typeName, "\x1F"); ok {
			typeName = fmt.Sprintf("%q.%s", namespace, local)
		}
	} else {
		switch valueType.type_ {
		case schema_idl.Type_BOOL:
			typeName = "bool"
		case schema_idl.Type_U8:
			typeName = "u8"
		case schema_idl.Type_I8:
			typeName = "i8"
		case schema_idl.Type_U16:
			typeName = "u16"
		case schema_idl.Type_I16:
			typeName = "i16"
		case schema_idl.Type_U32:
			typeName = "u32"
		case schema_idl.Type_I32:
			typeName = "i32"
		case schema_idl.Type_U64:
			typeName = "u64"
		case schema_idl.Type_I64:
			typeName = "i64"
		case schema_idl.Type_F32:
			typeName = "f32"
		case schema_idl.Type_F64:
			typeName = "f64"
		case schema_idl.Type_ASCIZ:
			typeName = "asciz"
		case schema_idl.Type_TEXT:
			typeName = "text"
		default:
			panic("unreachable")
		}
	}

	return &Error{
		code: 3015,
		message: fmt.Sprintf(
			"Cannot assign value %s to %s %s (type '%s')",
			value, dstNodeType, dstName, typeName,
		),
		span: valueNode.Span(),
	}
}

func errValueOutOfRange(type_ schema_idl.Type, node *syntax.IntLit) error {
	if value, ok := node.GetInt64(); ok {
		return errValueOutOfRange2(type_, value, node.Span())
	}
	value, _ := node.GetUint64()
	return errValueOutOfRange2(type_, value, node.Span())
}

func errValueOutOfRange2[V int64 | uint64](
	type_ schema_idl.Type,
	value V,
	valueSpan syntax.Span,
) error {
	var typeName string
	var valueMin any = 0
	var valueMax any = 12345
	var rangeDesc string
	switch type_ {
	case schema_idl.Type_U8:
		typeName = "u8"
		valueMax = math.MaxUint8
	case schema_idl.Type_I8:
		typeName = "i8"
		valueMin = math.MinInt8
		valueMax = math.MaxInt8
	case schema_idl.Type_U16:
		typeName = "u16"
		valueMax = math.MaxUint16
	case schema_idl.Type_I16:
		typeName = "i16"
		valueMin = int16(math.MinInt16)
		valueMax = int16(math.MaxInt16)
	case schema_idl.Type_U32:
		typeName = "u32"
		valueMax = uint32(math.MaxUint32)
	case schema_idl.Type_I32:
		typeName = "i32"
		valueMin = int32(math.MinInt32)
		valueMax = int32(math.MaxInt32)
	case schema_idl.Type_U64:
		typeName = "u64"
		valueMax = uint64(math.MaxUint64)
	case schema_idl.Type_I64:
		typeName = "i64"
		valueMin = int64(math.MinInt64)
		valueMax = int64(math.MaxInt64)
	case schema_idl.Type_F32:
		typeName = "f32"
		valueMin = -int64(maxFloat32)
		valueMax = int64(maxFloat32)
		rangeDesc = "floating-point unrounded integer "
	case schema_idl.Type_F64:
		typeName = "f64"
		valueMin = -int64(maxFloat64)
		valueMax = int64(maxFloat64)
		rangeDesc = "floating-point unrounded integer "
	default:
		panic("unreachable")
	}

	return &Error{
		code: 3016,
		message: fmt.Sprintf(
			"Value %d out of %srange [%d, %d] for type '%s'",
			value, rangeDesc, valueMin, valueMax, typeName,
		),
		span: valueSpan,
	}
}

func errInvalidBoolValue(node *syntax.EnumRef) error {
	return &Error{
		code:    3017,
		message: "Invalid value for type 'bool' (expected '.true' or '.false')",
		span:    node.Span(),
	}
}

func errInvalidAscizValue(node *syntax.TextLit) error {
	return &Error{
		code:    3018,
		message: "Invalid value for type 'asciz' (contains NUL)",
		span:    node.Span(),
	}
}

func errInvalidTextValue(node *syntax.TextLit) error {
	return &Error{
		code:    3019,
		message: "Invalid value for type 'text' (contains NUL and/or non-ASCII byte escape)",
		span:    node.Span(),
	}
}

func errConstTypeInvalid(name *syntax.TypeName) error {
	return &Error{
		code: 3020,
		message: fmt.Sprintf(
			"Invalid type '%s' for constant", fmtTypeName(name),
		),
		span: name.Span(),
	}
}

func errEnumTypeInvalid(name *syntax.Ident) error {
	return &Error{
		code:    3021,
		message: fmt.Sprintf("Invalid type '%s' for enum", name.Get()),
		span:    name.Span(),
	}
}

func errEnumItemNameConflict(
	enumType schema_idl.Type,
	prevValue *uint64,
	prevAlias string,
	name *syntax.Ident,
) error {
	var prev string
	if prevAlias == "" {
		switch enumType {
		case schema_idl.Type_U8:
			fallthrough
		case schema_idl.Type_U16:
			fallthrough
		case schema_idl.Type_U32:
			fallthrough
		case schema_idl.Type_U64:
			prev = fmt.Sprintf("%d", *prevValue)
		case schema_idl.Type_I8:
			fallthrough
		case schema_idl.Type_I16:
			fallthrough
		case schema_idl.Type_I32:
			fallthrough
		case schema_idl.Type_I64:
			prev = fmt.Sprintf("%d", int64(*prevValue))
		}
	} else {
		prev = "." + prevAlias
	}

	return &Error{
		code: 3022,
		message: fmt.Sprintf(
			"Enum item '%s' conflicts with name of earlier item '%s' (= %s)",
			name.Get(),
			name.Get(),
			prev,
		),
		span: name.Span(),
	}
}

func errEnumItemValueConflict(
	enumType schema_idl.Type,
	value uint64,
	name, prevName string,
	valueNode syntax.Node,
) error {
	var valueStr string
	switch enumType {
	case schema_idl.Type_U8:
		fallthrough
	case schema_idl.Type_U16:
		fallthrough
	case schema_idl.Type_U32:
		fallthrough
	case schema_idl.Type_U64:
		valueStr = fmt.Sprintf("%d", value)
	case schema_idl.Type_I8:
		fallthrough
	case schema_idl.Type_I16:
		fallthrough
	case schema_idl.Type_I32:
		fallthrough
	case schema_idl.Type_I64:
		valueStr = fmt.Sprintf("%d", int64(value))
	}

	return &Error{
		code: 3023,
		message: fmt.Sprintf(
			"Enum item '%s' value %s conflicts with value of earlier item '%s'",
			name, valueStr, prevName,
		),
		span: valueNode.Span(),
	}
}

func errStructEmpty(name string, span syntax.Span) error {
	return &Error{
		code:    3024,
		message: fmt.Sprintf("Struct '%s' contains no fields", name),
		span:    span,
	}
}

func errFieldNameConflict(
	recordType string,
	prev, field interface {
		syntax.Node
		Name() *syntax.Ident
	},
) error {
	type tagged interface {
		Tag() *syntax.Tag
	}

	var prevTag, fieldTag string
	if prev, ok := prev.(tagged); ok {
		if t, ok := prev.Tag().Value().GetUint64(); ok {
			prevTag = fmt.Sprintf(" (tag @%d)", t)
		}
	}
	if field, ok := field.(tagged); ok {
		if t, ok := field.Tag().Value().GetUint64(); ok {
			fieldTag = fmt.Sprintf(" (tag @%d)", t)
		}
	}
	return &Error{
		code: 3025,
		message: fmt.Sprintf(
			"%s field name '%s'%s conflicts with name of earlier field '%s'%s",
			recordType,
			field.Name().Get(),
			fieldTag,
			prev.Name().Get(),
			prevTag,
		),
		span: field.Name().Span(),
	}
}

func errFieldTagConflict(
	recordType string,
	tag uint16,
	prev, field interface {
		syntax.Node
		Name() *syntax.Ident
		Tag() *syntax.Tag
	},
) error {
	return &Error{
		code: 3026,
		message: fmt.Sprintf(
			"%s field '%s' (tag @%d) conflicts with tag of earlier field '%s'",
			recordType,
			field.Name().Get(),
			tag,
			prev.Name().Get(),
		),
		span: field.Tag().Span(),
	}
}

func errFieldTagOutOfRange(field syntax.Node, tagNode *syntax.Tag) error {
	var typeName string
	switch field.(type) {
	case *syntax.MessageField:
		typeName = "message"
	case *syntax.UnionField:
		typeName = "union"
	default:
		panic("unreachable")
	}

	var value any
	if v, ok := tagNode.Value().GetInt64(); ok {
		value = v
	} else {
		value, _ = tagNode.Value().GetUint64()
	}
	return &Error{
		code: 3027,
		message: fmt.Sprintf(
			"Value %d out of range [1, 65535] for %s field tag",
			value, typeName,
		),
		span: tagNode.Span(),
	}
}

func errProtocolItemNameConflict(item, prevItem syntax.Node) error {
	var name *syntax.Ident
	var itemType, prevItemType string
	switch item := item.(type) {
	case *syntax.ProtocolRpc:
		name = item.Name()
		itemType = "rpc"
	case *syntax.ProtocolEvent:
		name = item.Name()
		itemType = "event"
	default:
		panic("unreachable")
	}
	switch prevItem.(type) {
	case *syntax.ProtocolRpc:
		prevItemType = "rpc"
	case *syntax.ProtocolEvent:
		prevItemType = "event"
	default:
		panic("unreachable")
	}

	return &Error{
		code: 3028,
		message: fmt.Sprintf(
			"Protocol %s '%s' conflicts with name of earlier %s '%s' ",
			itemType,
			name.Get(),
			prevItemType,
			name.Get(),
		),
		span: name.Span(),
	}
}
func errProtocolItemTagConflict(item, prevItem syntax.Node) error {
	var name, prevName *syntax.Ident
	var tagNode *syntax.Tag
	var tag uint64
	var itemType, prevItemType string
	switch item := item.(type) {
	case *syntax.ProtocolRpc:
		name = item.Name()
		itemType = "rpc"
		tagNode = item.Tag()
		tag, _ = tagNode.Value().GetUint64()
	case *syntax.ProtocolEvent:
		name = item.Name()
		itemType = "event"
		tagNode = item.Tag()
		tag, _ = tagNode.Value().GetUint64()
	default:
		panic("unreachable")
	}
	switch prev := prevItem.(type) {
	case *syntax.ProtocolRpc:
		prevName = prev.Name()
		prevItemType = "rpc"
	case *syntax.ProtocolEvent:
		prevName = prev.Name()
		prevItemType = "event"
	default:
		panic("unreachable")
	}

	return &Error{
		code: 3029,
		message: fmt.Sprintf(
			"Protocol %s '%s' (tag @%d) conflicts with tag of earlier %s '%s' ",
			itemType,
			name.Get(),
			tag,
			prevItemType,
			prevName.Get(),
		),
		span: tagNode.Span(),
	}
}

func errProtocolTagOutOfRange(tagNode *syntax.Tag) error {
	var value any
	if v, ok := tagNode.Value().GetInt64(); ok {
		value = v
	} else {
		value, _ = tagNode.Value().GetUint64()
	}
	return &Error{
		code: 3030,
		message: fmt.Sprintf(
			"Value %d out of range [0, %d] for protocol item tag",
			value, uint64(math.MaxUint64),
		),
		span: tagNode.Value().Span(),
	}
}

func errOptionsSchemaMustBeMessage(
	name *syntax.TypeName,
	gotType schema_idl.Type,
) error {
	got := strings.ToLower(gotType.String())
	return &Error{
		code: 3031,
		message: fmt.Sprintf(
			"Options schema must be an imported message (got '%s')", got,
		),
		span: name.Span(),
	}
}

func errOptionsSchemaMustBeImported(name *syntax.TypeName) error {
	return &Error{
		code:    3032,
		message: "Options schema must be an imported message",
		span:    name.Span(),
	}
}

func errOptionsNameThroughNonMessage(
	name *syntax.OptionName,
	type_ schema_idl.Type,
	typeName string,
) error {
	return &Error{
		code:    0x30000,
		message: "errOptionsNameThroughNonMessage",
		span:    name.Span(),
	}
}
func errOptionTypeInvalid(type_ schema_idl.Type, typeName string) error {
	return &Error{
		code:    0x30000,
		message: "errOptionTypeInvalid",
	}
}

func errOptionNotFound() error {
	return &Error{
		code:    0x30000,
		message: "errOptionNotFound",
	}
}

func errEnumValueNotOk() error {
	return &Error{
		code:    0x30000,
		message: "errEnumValueNotOk",
	}
}

func errEnumValueNotIntOrAlias() error {
	return &Error{
		code:    0x30000,
		message: "errEnumValueNotIntOrAlias",
	}
}

func errEnumAliasTargetNotFound() error {
	return &Error{
		code:    0x30000,
		message: "errEnumAliasTargetNotFound",
	}
}

func errStructFieldUnsizedArray() error {
	return &Error{
		code:    0x30000,
		message: "errStructFieldUnsizedArray",
	}
}

func errArrayLenNotU32() error {
	return &Error{
		code:    0x30000,
		message: "errArrayLenNotU32",
	}
}

func errArrayLenZero() error {
	return &Error{
		code:    0x30000,
		message: "errArrayLenZero",
	}
}

func errArrayLenMaxU32() error {
	return &Error{
		code:    0x30000,
		message: "errArrayLenMaxU32",
	}
}

func errResolvedDeclNotType() error {
	return &Error{
		code:    0x30000,
		message: "errResolvedDeclNotType",
	}
}

func errResolvedDeclNotConst() error {
	return &Error{
		code:    0x30000,
		message: "errResolvedDeclNotConst",
	}
}

func errTypeNameNotFound() error {
	return &Error{
		code:    0x30000,
		message: "errTypeNameNotFound",
	}
}

func errExportNameNotFound() error {
	return &Error{
		code:    0x30000,
		message: "errExportNameNotFound",
	}
}

func errValueNameNotFound() error {
	return &Error{
		code:    0x30000,
		message: "errValueNameNotFound",
	}
}

func errImportedConstantCorrupt() error {
	return &Error{
		code:    0x30000,
		message: "errImportedConstantCorrupt",
	}
}

func errEnumRefNotFound() error {
	return &Error{
		code:    0x30000,
		message: "errEnumRefNotFound",
	}
}
