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
	"maps"
	"slices"

	"go.idol-lang.org/idol/schema_idl"
	"go.idol-lang.org/idol/syntax"
)

type SchemaSet struct {
	decls map[string] /*namespace*/ map[string] /* decl name */ *mergedDecl
}

func (s *SchemaSet) resolveExport(
	namespace string,
	name string,
	nameSpan syntax.Span,
) (*exportInfo, error) {
	decls := s.decls[namespace]
	decl, ok := decls[name]
	if !ok {
		return nil, errImportNameNotFound(namespace, name, nameSpan)
	}
	if decl.conflict {
		return nil, errImportNameDefinitionConflict(namespace, name, nameSpan)
	}
	switch decl.type_ {
	case declType_UNKNOWN:
		return &exportInfo{}, nil
	case declType_CONST:
		return &exportInfo{
			type_:    schema_idl.ExportType_CONST,
			imported: decl.value,
		}, nil
	case declType_ENUM:
		return &exportInfo{
			type_:    schema_idl.ExportType_ENUM,
			imported: decl.value,
		}, nil
	case declType_STRUCT:
		return &exportInfo{
			type_:    schema_idl.ExportType_STRUCT,
			imported: decl.value,
		}, nil
	case declType_MESSAGE:
		return &exportInfo{
			type_:    schema_idl.ExportType_MESSAGE,
			imported: decl.value,
		}, nil
	case declType_UNION:
		return &exportInfo{
			type_:    schema_idl.ExportType_UNION,
			imported: decl.value,
		}, nil
	case declType_PROTOCOL:
		return &exportInfo{
			type_:    schema_idl.ExportType_PROTOCOL,
			imported: decl.value,
		}, nil
	}
	panic("unreachable")
}

func (s *SchemaSet) resolveType(
	namespace string,
	name string,
	importSpan syntax.Span,
	useSpan syntax.Span,
) (*typeInfo, error) {
	decls := s.decls[namespace]
	decl, ok := decls[name]
	if !ok {
		return nil, errImportNameNotFound(namespace, name, importSpan)
	}
	if decl.conflict {
		return nil, errImportNameDefinitionConflict(namespace, name, importSpan)
	}
	switch decl.type_ {
	case declType_UNKNOWN:
		return &typeInfo{
			typeName: name,
		}, nil
	case declType_ENUM:
		return &typeInfo{
			type_:    decl.enumType,
			typeName: name,
			imported: decl.value,
		}, nil
	case declType_STRUCT:
		return &typeInfo{
			type_:    schema_idl.Type_STRUCT,
			typeName: name,
			imported: decl.value,
		}, nil
	case declType_MESSAGE:
		return &typeInfo{
			type_:    schema_idl.Type_MESSAGE,
			typeName: name,
			imported: decl.value,
		}, nil
	case declType_UNION:
		return &typeInfo{
			type_:    schema_idl.Type_UNION,
			typeName: name,
			imported: decl.value,
		}, nil
	case declType_CONST:
		return nil, errImportedNameNotType("const", namespace, name, useSpan)
	case declType_PROTOCOL:
		return nil, errImportedNameNotType("protocol", namespace, name, useSpan)
	}
	panic("unreachable")
}

func (s *SchemaSet) resolveConst(
	namespace string,
	name string,
	importSpan syntax.Span,
	useSpan syntax.Span,
) (schema_idl.Const, error) {
	var zero schema_idl.Const
	decls := s.decls[namespace]
	decl, ok := decls[name]
	if !ok {
		return zero, errImportNameNotFound(namespace, name, importSpan)
	}
	if decl.conflict {
		return zero, errImportNameDefinitionConflict(namespace, name, importSpan)
	}

	var gotType string
	switch decl.type_ {
	case declType_CONST:
		return decl.value.(schema_idl.Const), nil
	case declType_ENUM:
		gotType = "enum"
	case declType_STRUCT:
		gotType = "struct"
	case declType_MESSAGE:
		gotType = "message"
	case declType_UNION:
		gotType = "union"
	case declType_PROTOCOL:
		gotType = "protocol"
	default:
		panic("unreachable")
	}
	return zero, errImportedNameNotConst(gotType, namespace, name, useSpan)
}

type mergedDecl struct {
	type_    declType
	enumType schema_idl.Type
	value    interface{}
	conflict bool
}

func canUnifyMergedDecls(a, b *mergedDecl) bool {
	if a.conflict || b.conflict {
		return false
	}
	if a.type_ != b.type_ {
		return false
	}
	if a.enumType != b.enumType {
		return false
	}
	switch a.type_ {
	case declType_UNKNOWN:
		return false
	case declType_CONST:
		return a.value.(schema_idl.Const) == b.value.(schema_idl.Const)
	case declType_ENUM:
		return a.value.(schema_idl.Enum) == b.value.(schema_idl.Enum)
	case declType_STRUCT:
		return a.value.(schema_idl.Struct) == b.value.(schema_idl.Struct)
	case declType_MESSAGE:
		return a.value.(schema_idl.Message) == b.value.(schema_idl.Message)
	case declType_UNION:
		return a.value.(schema_idl.Union) == b.value.(schema_idl.Union)
	case declType_PROTOCOL:
		return a.value.(schema_idl.Protocol) == b.value.(schema_idl.Protocol)
	default:
		return false
	}
}

func Merge(schemas []schema_idl.Schema) (*SchemaSet, error) {
	set := func(decls map[string]*mergedDecl, k string, v *mergedDecl) {
		if prev, conflict := decls[k]; conflict {
			if !canUnifyMergedDecls(v, prev) {
				decls[k] = &mergedDecl{
					conflict: true,
				}
			}
			return
		}
		decls[k] = v
	}

	declsByNs := make(map[string]map[string]*mergedDecl)
	for _, schema := range schemas {
		decls := make(map[string]*mergedDecl)
		for _, const_ := range schema.Consts().Iter() {
			set(decls, const_.Name(), &mergedDecl{
				type_: declType_CONST,
				value: const_,
			})
		}
		for _, enum := range schema.Enums().Iter() {
			set(decls, enum.Name(), &mergedDecl{
				type_:    declType_ENUM,
				enumType: enum.Type(),
				value:    enum,
			})
		}
		for _, struct_ := range schema.Structs().Iter() {
			set(decls, struct_.Name(), &mergedDecl{
				type_: declType_STRUCT,
				value: struct_,
			})
		}
		for _, message := range schema.Messages().Iter() {
			set(decls, message.Name(), &mergedDecl{
				type_: declType_MESSAGE,
				value: message,
			})
		}
		for _, union := range schema.Unions().Iter() {
			set(decls, union.Name(), &mergedDecl{
				type_: declType_UNION,
				value: union,
			})
		}
		for _, protocol := range schema.Protocols().Iter() {
			set(decls, protocol.Name(), &mergedDecl{
				type_: declType_PROTOCOL,
				value: protocol,
			})
		}

		ns := schema.Namespace()
		if prevDecls, ok := declsByNs[ns]; ok {
			for _, name := range slices.Sorted(maps.Keys(decls)) {
				if prev, conflict := prevDecls[name]; conflict {
					if !canUnifyMergedDecls(decls[name], prev) {
						prevDecls[name] = &mergedDecl{
							conflict: true,
						}
					}
				} else {
					prevDecls[name] = decls[name]
				}
			}
		} else {
			declsByNs[ns] = decls
		}
	}

	return &SchemaSet{
		decls: declsByNs,
	}, nil
}
