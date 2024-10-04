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
	"fmt"

	"go.idol-lang.org/idol/syntax"
)

type Warning struct {
	code    uint32
	message string
	span    syntax.Span
}

func (w *Warning) String() string {
	return fmt.Sprintf("W%d: %s", w.code, w.message)
}

func (w *Warning) Code() uint32 {
	return w.code
}

func (w *Warning) Message() string {
	return w.message
}

func (w *Warning) Span() syntax.Span {
	return w.span
}

func warnEmptyImport(ns string, span syntax.Span) *Warning {
	return &Warning{
		code:    4000,
		message: fmt.Sprintf("Import from namespace %q is empty", ns),
		span:    span,
	}
}

func warnUnusedImport(ns, name string, span syntax.Span) *Warning {
	return &Warning{
		code:    4001,
		message: fmt.Sprintf("Import '%s' from namespace %q is unused", name, ns),
		span:    span,
	}
}

func warnUnusedImportAs(ns, as string, span syntax.Span) *Warning {
	return &Warning{
		code:    4002,
		message: fmt.Sprintf("Import of namespace %q (as %q) is unused", ns, as),
		span:    span,
	}
}

func warnDuplicateImport(ns, name string, span syntax.Span) *Warning {
	return &Warning{
		code:    4003,
		message: fmt.Sprintf("Duplicate import '%s' from namespace %q", name, ns),
		span:    span,
	}
}

func warnDuplicateImportAs(ns, as string, span syntax.Span) *Warning {
	return &Warning{
		code:    4004,
		message: fmt.Sprintf("Duplicate import of namespace %q (as %q)", ns, as),
		span:    span,
	}
}

func warnEmptyExport(span syntax.Span) *Warning {
	return &Warning{
		code:    4005,
		message: "Export is empty",
		span:    span,
	}
}

func warnDuplicateExport(nameNode *syntax.ExportName) *Warning {
	typeName := fmtTypeName(nameNode)
	return &Warning{
		code:    4006,
		message: fmt.Sprintf("Duplicate export of '%s'", typeName),
		span:    nameNode.Span(),
	}
}

func warnExportAsSameName(
	nameNode *syntax.ExportName,
	as string,
	span syntax.Span,
) *Warning {
	name := fmtTypeName(nameNode)
	return &Warning{
		code:    4007,
		message: fmt.Sprintf("Export of '%s' as '%s' (same name)", name, as),
		span:    span,
	}
}

func warnExportLocalDecl(name string, span syntax.Span) *Warning {
	return &Warning{
		code:    4008,
		message: fmt.Sprintf("Export of local declaration '%s' has no effect", name),
		span:    span,
	}
}

func warnDeclShadowsBuiltin(name string, span syntax.Span) *Warning {
	return &Warning{
		code:    4009,
		message: fmt.Sprintf("Local declaration '%s' shadows builtin", name),
		span:    span,
	}
}

func warnDuplicateOption(name string, node syntax.Node) *Warning {
	return &Warning{
		code:    4010,
		message: fmt.Sprintf("Duplicate option '%s' with same value", name),
		span:    node.Span(),
	}
}

func warnOptionNameNotFound(name string, node syntax.Node) *Warning {
	return &Warning{
		code:    4011,
		message: fmt.Sprintf("Option name '%s' not found in schema", name),
		span:    node.Span(),
	}
}

func fmtTypeName(node interface {
	Scope() *syntax.Ident
	Name() *syntax.Ident
}) string {
	typeName := node.Name().Get()
	if scope := node.Scope(); scope != nil {
		return scope.Get() + "." + typeName
	}
	return typeName
}
