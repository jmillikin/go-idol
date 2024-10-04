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
)

type ParseOption interface {
	apply(*ParseOptions)
}

func Parse(src []uint8, opts ...ParseOption) (*Schema, error) {
	return NewParseOptions(opts...).ParseSchema(src)
}

type ParseOptions struct {
	saveSpaces   bool
	saveNewlines bool
	saveComments bool
}

func NewParseOptions(opts ...ParseOption) *ParseOptions {
	return &ParseOptions{
		// TODO
		saveSpaces:   true,
		saveNewlines: true,
		saveComments: true,
	}
}

func (opts *ParseOptions) ParseSchema(src []uint8) (*Schema, error) {
	ctx, err := newParseCtx[Schema](opts, src)
	if err != nil {
		return nil, err
	}
	return parseSchema(ctx)
}

func (opts *ParseOptions) ParseNamespace(src []uint8) (*Namespace, error) {
	ctx, err := newParseCtx[Namespace](opts, src)
	if err != nil {
		return nil, err
	}
	return parseNamespace(ctx)
}

func (opts *ParseOptions) ParseImport(src []uint8) (*Import, error) {
	ctx, err := newParseCtx[Import](opts, src)
	if err != nil {
		return nil, err
	}
	return parseImport(ctx)
}

func (opts *ParseOptions) ParseExport(src []uint8) (*Export, error) {
	ctx, err := newParseCtx[Export](opts, src)
	if err != nil {
		return nil, err
	}
	return parseExport(ctx)
}

func (opts *ParseOptions) ParseOptions(src []uint8) (*Options, error) {
	ctx, err := newParseCtx[Options](opts, src)
	if err != nil {
		return nil, err
	}
	return parseOptions(ctx)
}

func (opts *ParseOptions) ParseConst(src []uint8) (*Const, error) {
	ctx, err := newParseCtx[Const](opts, src)
	if err != nil {
		return nil, err
	}
	return parseConst(ctx)
}

func (opts *ParseOptions) ParseEnum(src []uint8) (*Enum, error) {
	ctx, err := newParseCtx[Enum](opts, src)
	if err != nil {
		return nil, err
	}
	return parseEnum(ctx)
}

func (opts *ParseOptions) ParseStruct(src []uint8) (*Struct, error) {
	ctx, err := newParseCtx[Struct](opts, src)
	if err != nil {
		return nil, err
	}
	return parseStruct(ctx)
}

func (opts *ParseOptions) ParseMessage(src []uint8) (*Message, error) {
	ctx, err := newParseCtx[Message](opts, src)
	if err != nil {
		return nil, err
	}
	return parseMessage(ctx)
}

func (opts *ParseOptions) ParseUnion(src []uint8) (*Union, error) {
	ctx, err := newParseCtx[Union](opts, src)
	if err != nil {
		return nil, err
	}
	return parseUnion(ctx)
}

func (opts *ParseOptions) ParseProtocol(src []uint8) (*Protocol, error) {
	ctx, err := newParseCtx[Protocol](opts, src)
	if err != nil {
		return nil, err
	}
	return parseProtocol(ctx)
}

type parseCtx[T any] struct {
	src        []uint8
	opts       *ParseOptions
	tokens     *Tokens
	childNodes []Node
	haveToken  bool
	token      Token
	err        error
	consumed   uint32
	offset     uint32
}

func newParseCtx[T any](opts *ParseOptions, src []uint8) (*parseCtx[T], error) {
	tokens, err := NewTokens(src)
	if err != nil {
		return nil, err
	}
	return &parseCtx[T]{
		src:    src,
		opts:   opts,
		tokens: tokens,
	}, nil
}

func (ctx *parseCtx[T]) ensureToken() error {
	if ctx.err != nil {
		return ctx.err
	}
	if ctx.haveToken {
		return nil
	}
	if err := ctx.tokens.Next(&ctx.token); err != nil {
		ctx.err = err
		return ctx.err
	}
	ctx.haveToken = true
	return nil
}

func (ctx *parseCtx[T]) readToken() []uint8 {
	return ctx.src[:ctx.token.Len]
}

func (ctx *parseCtx[T]) consumeToken(child Node) {
	ctx.src = ctx.src[ctx.token.Len:]
	ctx.consumed += uint32(ctx.token.Len)
	ctx.offset += uint32(ctx.token.Len)
	ctx.haveToken = false
	if child != nil {
		ctx.childNodes = append(ctx.childNodes, child)
	}
}

func (ctx *parseCtx[T]) tokenSpan() Span {
	return Span{
		start: ctx.offset,
		len:   uint32(ctx.token.Len),
	}
}

func (ctx *parseCtx[T]) loop(yield func(struct{}) bool) {
	if ctx.err != nil {
		return
	}
	for {
		consumed := ctx.consumed
		if !yield(struct{}{}) {
			return
		}
		if ctx.err != nil {
			return
		}
		if consumed == ctx.consumed {
			return
		}
	}
}

func (ctx *parseCtx[T]) space() {
	if err := ctx.ensureToken(); err != nil {
		return
	}
	if ctx.token.Kind != T_SPACE {
		return
	}
	ctx.consumeSpace()
}

func (ctx *parseCtx[T]) consumeSpace() {
	if !ctx.opts.saveSpaces {
		ctx.consumeToken(nil)
		return
	}

	tokenBytes := ctx.readToken()
	var token string
	if bytes.Equal(tokenBytes, []uint8{' '}) {
		token = " "
	} else {
		token = string(tokenBytes)
	}
	ctx.consumeToken(&Space{
		raw:   token,
		start: ctx.offset,
	})
}

func (ctx *parseCtx[T]) comments() {
	for _ = range ctx.loop {
		if err := ctx.ensureToken(); err != nil {
			return
		}
		switch ctx.token.Kind {
		case T_SPACE:
			ctx.consumeSpace()
		case T_NEWLINE:
			var child *Newline
			if ctx.opts.saveNewlines {
				child = &Newline{
					crlf:  ctx.token.Len == 2,
					start: ctx.offset,
				}
			}
			ctx.consumeToken(child)
		case T_COMMENT:
			var child *Comment
			if ctx.opts.saveComments {
				child = &Comment{
					raw:   string(ctx.readToken()),
					start: ctx.offset,
				}
			}
			ctx.consumeToken(child)
		default:
			return
		}
	}
}

func (ctx *parseCtx[T]) suffixComment() {
	// TODO
}

func (ctx *parseCtx[T]) sigil(kind TokenKind) {
	if err := ctx.ensureToken(); err != nil {
		return
	}
	if ctx.token.Kind != kind {
		ctx.err = errExpectedSigil(
			kind,
			ctx.token.Kind,
			string(ctx.readToken()),
			ctx.tokenSpan(),
		)
		return
	}
	ctx.consumeToken(&Sigil{
		raw:   ctx.src[0],
		start: ctx.offset,
	})
}

func (ctx *parseCtx[T]) trySigil(kind TokenKind) bool {
	if err := ctx.ensureToken(); err != nil {
		return false
	}
	if ctx.token.Kind != kind {
		return false
	}
	ctx.consumeToken(&Sigil{
		raw:   ctx.src[0],
		start: ctx.offset,
	})
	return true
}

func (ctx *parseCtx[T]) tryKeyword(keyword string) bool {
	if err := ctx.ensureToken(); err != nil {
		return false
	}
	if ctx.token.Kind != T_IDENT {
		return false
	}
	if string(ctx.readToken()) != keyword {
		return false
	}
	ctx.consumeToken(&Keyword{
		raw:   keyword,
		start: ctx.offset,
	})
	return true
}

func (ctx *parseCtx[T]) ident() *Ident {
	if err := ctx.ensureToken(); err != nil {
		return nil
	}
	token := string(ctx.readToken())
	if ctx.token.Kind != T_IDENT {
		ctx.err = errExpectedIdent(ctx.token.Kind, token, ctx.tokenSpan())
		return nil
	}
	ident := &Ident{
		raw:   token,
		start: ctx.offset,
	}
	ctx.consumeToken(ident)
	return ident
}

func (ctx *parseCtx[T]) int() *IntLit {
	if err := ctx.ensureToken(); err != nil {
		return nil
	}
	token := string(ctx.readToken())

	switch ctx.token.Kind {
	case T_INT_LIT, T_BIN_INT_LIT, T_OCT_INT_LIT, T_DEC_INT_LIT, T_HEX_INT_LIT:
	default:
		ctx.err = errExpectedIntLit(ctx.token.Kind, token, ctx.tokenSpan())
		return nil
	}

	intNode, err := newIntLit(token, ctx.token.Kind, ctx.offset)
	if err != nil {
		ctx.err = err
		return nil
	}
	ctx.consumeToken(intNode)
	return intNode
}

func (ctx *parseCtx[T]) text() *TextLit {
	if err := ctx.ensureToken(); err != nil {
		return nil
	}
	token := string(ctx.readToken())

	if ctx.token.Kind != T_TEXT_LIT {
		ctx.err = errExpectedTextLit(ctx.token.Kind, token, ctx.tokenSpan())
		return nil
	}
	textNode, err := newTextLit(token, ctx.offset, ctx.token.flags)
	if err != nil {
		ctx.err = err
		return nil
	}
	ctx.consumeToken(textNode)
	return textNode
}

func (ctx *parseCtx[T]) finish(
	build func(span Span, childNodes []Node) *T,
) (*T, error) {
	if ctx.err != nil {
		return nil, ctx.err
	}
	span := Span{
		start: ctx.offset - ctx.consumed,
		len:   ctx.consumed,
	}
	return build(span, ctx.childNodes), nil
}

func parseChild[P any, C any, PtrC interface {
	*C
	Node
}](
	ctx *parseCtx[P],
	parseChildFn func(*parseCtx[C]) (PtrC, error),
) (*C, bool) {
	if ctx.err != nil {
		return nil, false
	}
	childCtx := &parseCtx[C]{
		src:       ctx.src,
		opts:      ctx.opts,
		tokens:    ctx.tokens,
		haveToken: ctx.haveToken,
		token:     ctx.token,
		offset:    ctx.offset,
	}
	child, err := parseChildFn(childCtx)
	if err != nil {
		ctx.err = err
		return nil, false
	}

	ctx.haveToken = childCtx.haveToken
	ctx.token = childCtx.token

	if childCtx.consumed == 0 {
		return nil, false
	}
	ctx.src = ctx.src[childCtx.consumed:]
	ctx.consumed += childCtx.consumed
	ctx.offset = childCtx.offset
	ctx.childNodes = append(ctx.childNodes, child)
	return child, true
}

func parseSchema(ctx *parseCtx[Schema]) (*Schema, error) {
	ctx.comments()
	parseChild(ctx, parseNamespace)

	for _ = range ctx.loop {
		ctx.comments()
		parseChild(ctx, parseImport)
	}

	for _ = range ctx.loop {
		ctx.comments()
		parseChild(ctx, parseExport)
	}

	for _ = range ctx.loop {
		ctx.comments()
		parseChild(ctx, parseOptions)
	}

	for _ = range ctx.loop {
		ctx.comments()

		if ctx.token.Kind == T_EOF {
			break
		}

		decorators := parseDecorators(ctx)

		var ok bool
		{
			var decl *Const
			if decl, ok = parseChild(ctx, parseConst); ok {
				setDecorators(decl, decorators)
			}
		}
		if !ok && ctx.err == nil {
			var decl *Enum
			if decl, ok = parseChild(ctx, parseEnum); ok {
				setDecorators(decl, decorators)
			}
		}
		if !ok && ctx.err == nil {
			var decl *Struct
			if decl, ok = parseChild(ctx, parseStruct); ok {
				setDecorators(decl, decorators)
			}
		}
		if !ok && ctx.err == nil {
			var decl *Message
			if decl, ok = parseChild(ctx, parseMessage); ok {
				setDecorators(decl, decorators)
			}
		}
		if !ok && ctx.err == nil {
			var decl *Union
			if decl, ok = parseChild(ctx, parseUnion); ok {
				setDecorators(decl, decorators)
			}
		}
		if !ok && ctx.err == nil {
			var decl *Protocol
			if decl, ok = parseChild(ctx, parseProtocol); ok {
				setDecorators(decl, decorators)
			}
		}
		if ctx.err != nil {
			return nil, ctx.err
		}
		if !ok {
			token := string(ctx.readToken())
			span := ctx.tokenSpan()
			if ctx.token.Kind == T_IDENT {
				return nil, errUnknownDeclaration(token, span)
			}
			return nil, errExpectedDeclaration(ctx.token.Kind, token, span)
		}
	}

	return ctx.finish(func(span Span, childNodes []Node) *Schema {
		return &Schema{
			span:       span,
			childNodes: childNodes,
		}
	})
}

func parseNamespace(ctx *parseCtx[Namespace]) (*Namespace, error) {
	if !ctx.tryKeyword("namespace") {
		return nil, errExpectedKeywordNamespace(
			ctx.token.Kind,
			string(ctx.readToken()),
			ctx.tokenSpan(),
		)
	}
	ctx.space()
	namespace := ctx.text()
	ctx.suffixComment()

	return ctx.finish(func(span Span, childNodes []Node) *Namespace {
		return &Namespace{
			span:       span,
			childNodes: childNodes,
			namespace:  namespace,
		}
	})
}

func parseImport(ctx *parseCtx[Import]) (*Import, error) {
	if !ctx.tryKeyword("import") {
		return nil, nil
	}
	ctx.space()
	namespace := ctx.text()
	ctx.space()

	var importAs *Ident
	var importNames []*Ident
	if ctx.tryKeyword("as") {
		ctx.space()
		importAs = ctx.ident()
	} else {
		ctx.sigil(T_OPEN_CURL)
		ctx.comments()
		for _ = range ctx.loop {
			if ctx.trySigil(T_CLOSE_CURL) {
				break
			}
			importNames = append(importNames, ctx.ident())
			ctx.suffixComment()
			ctx.comments()
		}
	}
	ctx.suffixComment()

	return ctx.finish(func(span Span, childNodes []Node) *Import {
		return &Import{
			span:        span,
			childNodes:  childNodes,
			namespace:   namespace,
			importAs:    importAs,
			importNames: importNames,
		}
	})
}

func parseExport(ctx *parseCtx[Export]) (*Export, error) {
	if !ctx.tryKeyword("export") {
		return nil, nil
	}
	ctx.space()

	var exportAs exportAs
	var exportNames []*ExportName
	if err := ctx.ensureToken(); err != nil {
		return nil, err
	}
	if ctx.token.Kind == T_IDENT {
		exportName, _ := parseChild(ctx, parseExportName)
		ctx.space()
		if !ctx.tryKeyword("as") {
			return nil, errExpectedKeywordAs(
				ctx.token.Kind,
				string(ctx.readToken()),
				ctx.tokenSpan(),
			)
		}
		ctx.space()
		name := ctx.ident()
		exportAs.exportName = exportName
		exportAs.name = name
	} else {
		ctx.sigil(T_OPEN_CURL)
		ctx.comments()
		for _ = range ctx.loop {
			if ctx.trySigil(T_CLOSE_CURL) {
				break
			}
			name, _ := parseChild(ctx, parseExportName)
			ctx.suffixComment()
			ctx.comments()
			exportNames = append(exportNames, name)
		}
	}
	ctx.suffixComment()

	return ctx.finish(func(span Span, childNodes []Node) *Export {
		return &Export{
			span:        span,
			childNodes:  childNodes,
			exportAs:    exportAs,
			exportNames: exportNames,
		}
	})
}

func parseTypeName(ctx *parseCtx[TypeName]) (*TypeName, error) {
	if err := ctx.ensureToken(); err != nil {
		return nil, err
	}
	if ctx.token.Kind != T_IDENT {
		return nil, errExpectedTypeName(
			ctx.token.Kind,
			string(ctx.readToken()),
			ctx.tokenSpan(),
		)
	}

	var scope, name *Ident
	name = ctx.ident()
	if ctx.trySigil(T_DOT) {
		scope = name
		name = ctx.ident()
	}
	return ctx.finish(func(span Span, childNodes []Node) *TypeName {
		return &TypeName{
			span:       span,
			childNodes: childNodes,
			scope:      scope,
			name:       name,
		}
	})
}

func parseValueName(ctx *parseCtx[ValueName]) (*ValueName, error) {
	if err := ctx.ensureToken(); err != nil {
		return nil, err
	}
	if ctx.token.Kind != T_IDENT {
		return nil, errExpectedValueName(
			ctx.token.Kind,
			string(ctx.readToken()),
			ctx.tokenSpan(),
		)
	}

	var scope, name *Ident
	name = ctx.ident()
	if ctx.trySigil(T_DOT) {
		scope = name
		name = ctx.ident()
	}
	return ctx.finish(func(span Span, childNodes []Node) *ValueName {
		return &ValueName{
			span:       span,
			childNodes: childNodes,
			scope:      scope,
			name:       name,
		}
	})
}

func parseExportName(ctx *parseCtx[ExportName]) (*ExportName, error) {
	if err := ctx.ensureToken(); err != nil {
		return nil, err
	}
	if ctx.token.Kind != T_IDENT {
		return nil, errExpectedExportName(
			ctx.token.Kind,
			string(ctx.readToken()),
			ctx.tokenSpan(),
		)
	}

	var scope, name *Ident
	name = ctx.ident()
	if ctx.trySigil(T_DOT) {
		scope = name
		name = ctx.ident()
	}
	return ctx.finish(func(span Span, childNodes []Node) *ExportName {
		return &ExportName{
			span:       span,
			childNodes: childNodes,
			scope:      scope,
			name:       name,
		}
	})
}

func parseOptions(ctx *parseCtx[Options]) (*Options, error) {
	if !ctx.tryKeyword("options") {
		return nil, nil
	}
	ctx.space()

	var schema *TypeName
	if ctx.trySigil(T_COLON) {
		ctx.space()
		schema, _ = parseChild(ctx, parseTypeName)
		ctx.space()
	}

	var options []*OptionsOption
	ctx.sigil(T_OPEN_CURL)
	ctx.comments()
	for _ = range ctx.loop {
		if ctx.trySigil(T_CLOSE_CURL) {
			break
		}
		option, _ := parseChild(ctx, parseOptionsOption)
		options = append(options, option)
		ctx.suffixComment()
		ctx.comments()
	}

	return ctx.finish(func(span Span, childNodes []Node) *Options {
		return &Options{
			span:       span,
			childNodes: childNodes,
			schema:     schema,
			options:    options,
		}
	})
}

func parseOptionsOption(
	ctx *parseCtx[OptionsOption],
) (*OptionsOption, error) {
	name, _ := parseChild(ctx, parseOptionName)
	ctx.space()
	ctx.sigil(T_EQ)
	ctx.space()
	value := parseOptionValue(ctx)

	return ctx.finish(func(span Span, childNodes []Node) *OptionsOption {
		return &OptionsOption{
			span:       span,
			childNodes: childNodes,
			name:       name,
			value:      value,
		}
	})
}

func parseOption(ctx *parseCtx[Option]) (*Option, error) {
	if !ctx.trySigil(T_OPEN_CURL) {
		return nil, nil
	}
	ctx.space()
	name, _ := parseChild(ctx, parseOptionName)
	ctx.space()

	var value Node
	if ctx.trySigil(T_EQ) {
		ctx.space()
		value = parseOptionValue(ctx)
	}

	ctx.space()
	ctx.sigil(T_CLOSE_CURL)

	return ctx.finish(func(span Span, childNodes []Node) *Option {
		return &Option{
			span:       span,
			childNodes: childNodes,
			name:       name,
			value:      value,
		}
	})
}

func parseOptionName(ctx *parseCtx[OptionName]) (*OptionName, error) {
	if err := ctx.ensureToken(); err != nil {
		return nil, err
	}
	if ctx.token.Kind != T_IDENT {
		return nil, errExpectedOptionName(
			ctx.token.Kind,
			string(ctx.readToken()),
			ctx.tokenSpan(),
		)
	}

	ctx.ident()
	for _ = range ctx.loop {
		if ctx.trySigil(T_DOT) {
			ctx.ident()
		} else {
			break
		}
	}
	return ctx.finish(func(span Span, childNodes []Node) *OptionName {
		return &OptionName{
			span:       span,
			childNodes: childNodes,
		}
	})
}

func setDecorators[T any](node *T, decorators []*Decorator) {
	type setter interface {
		setDecorators([]*Decorator)
	}
	if node != nil {
		var iface interface{} = node
		iface.(setter).setDecorators(decorators)
	}
}

func parseDecorators[T any](ctx *parseCtx[T]) []*Decorator {
	var decorators []*Decorator
	for _ = range ctx.loop {
		if decorator, ok := parseChild(ctx, parseDecorator); ok {
			decorators = append(decorators, decorator)
			ctx.comments()
		}
	}
	return decorators
}

func parseDecorator(ctx *parseCtx[Decorator]) (*Decorator, error) {
	if !ctx.trySigil(T_AT) {
		return nil, nil
	}

	var value Node
	value, ok := parseChild(ctx, parseOptions)
	if !ok && ctx.err == nil {
		value, ok = parseChild(ctx, parseOption)
	}
	if ctx.err != nil {
		return nil, ctx.err
	}
	if !ok {
		token := string(ctx.readToken())
		return nil, errUnknownDecorator(token, ctx.tokenSpan())
	}

	return ctx.finish(func(span Span, childNodes []Node) *Decorator {
		return &Decorator{
			span:       span,
			childNodes: childNodes,
			value:      value,
		}
	})
}

func parseConst(ctx *parseCtx[Const]) (*Const, error) {
	if !ctx.tryKeyword("const") {
		return nil, nil
	}
	ctx.space()
	name := ctx.ident()
	ctx.space()
	ctx.sigil(T_COLON)
	ctx.space()
	typeName, _ := parseChild(ctx, parseTypeName)
	ctx.space()
	ctx.sigil(T_EQ)
	ctx.space()
	value := parseConstValue(ctx)

	return ctx.finish(func(span Span, childNodes []Node) *Const {
		return &Const{
			span:       span,
			childNodes: childNodes,
			name:       name,
			typeName:   typeName,
			value:      value,
		}
	})
}

func parseConstValue[T any](ctx *parseCtx[T]) Node {
	node := parseValue(ctx)
	if node == nil && ctx.err == nil {
		ctx.err = errExpectedConstValue(
			ctx.token.Kind,
			string(ctx.readToken()),
			ctx.tokenSpan(),
		)
	}
	return node
}

func parseOptionValue[T any](ctx *parseCtx[T]) Node {
	node := parseValue(ctx)
	if node == nil && ctx.err == nil {
		ctx.err = errExpectedOptionValue(
			ctx.token.Kind,
			string(ctx.readToken()),
			ctx.tokenSpan(),
		)
	}
	return node
}

func parseValue[T any](ctx *parseCtx[T]) Node {
	if err := ctx.ensureToken(); err != nil {
		return nil
	}
	switch ctx.token.Kind {
	case T_INT_LIT, T_BIN_INT_LIT, T_OCT_INT_LIT, T_DEC_INT_LIT, T_HEX_INT_LIT:
		if child := ctx.int(); child != nil {
			return child
		}
	case T_TEXT_LIT:
		if child := ctx.text(); child != nil {
			return child
		}
	case T_DOT:
		if child, _ := parseChild(ctx, parseEnumRef); child != nil {
			return child
		}
	case T_IDENT:
		if child, ok := parseChild(ctx, parseValueName); ok {
			return child
		}
	}
	return nil
}

func parseEnumRef(ctx *parseCtx[EnumRef]) (*EnumRef, error) {
	ctx.sigil(T_DOT)
	name := ctx.ident()
	return ctx.finish(func(span Span, childNodes []Node) *EnumRef {
		return &EnumRef{
			span:       span,
			childNodes: childNodes,
			name:       name,
		}
	})
}

func parseEnum(ctx *parseCtx[Enum]) (*Enum, error) {
	if !ctx.tryKeyword("enum") {
		return nil, nil
	}
	ctx.space()
	name := ctx.ident()
	ctx.space()
	ctx.sigil(T_COLON)
	ctx.space()
	type_ := ctx.ident()
	ctx.space()

	var items []*EnumItem
	ctx.sigil(T_OPEN_CURL)
	ctx.comments()
	for _ = range ctx.loop {
		if ctx.trySigil(T_CLOSE_CURL) {
			break
		}
		decorators := parseDecorators(ctx)
		item, _ := parseChild(ctx, parseEnumItem)
		setDecorators(item, decorators)
		items = append(items, item)
		ctx.suffixComment()
		ctx.comments()
	}
	ctx.suffixComment()

	return ctx.finish(func(span Span, childNodes []Node) *Enum {
		return &Enum{
			span:       span,
			childNodes: childNodes,
			name:       name,
			type_:      type_,
			items:      items,
		}
	})
}

func parseEnumItem(ctx *parseCtx[EnumItem]) (*EnumItem, error) {
	name := ctx.ident()
	ctx.space()
	ctx.sigil(T_EQ)
	ctx.space()

	var value Node
	if err := ctx.ensureToken(); err != nil {
		return nil, err
	}
	if ctx.token.Kind == T_DOT {
		value, _ = parseChild(ctx, parseEnumRef)
	} else if ctx.token.Kind == T_IDENT {
		value, _ = parseChild(ctx, parseValueName)
	} else {
		value = ctx.int()
	}
	ctx.suffixComment()

	return ctx.finish(func(span Span, childNodes []Node) *EnumItem {
		return &EnumItem{
			span:       span,
			childNodes: childNodes,
			name:       name,
			value:      value,
		}
	})
}

func parseStruct(ctx *parseCtx[Struct]) (*Struct, error) {
	if !ctx.tryKeyword("struct") {
		return nil, nil
	}
	ctx.space()
	name := ctx.ident()
	ctx.space()

	ctx.sigil(T_OPEN_CURL)
	ctx.comments()
	var fields []*StructField
	for _ = range ctx.loop {
		if ctx.trySigil(T_CLOSE_CURL) {
			break
		}
		decorators := parseDecorators(ctx)
		field, _ := parseChild(ctx, parseStructField)
		setDecorators(field, decorators)
		fields = append(fields, field)
		ctx.suffixComment()
		ctx.comments()
	}
	ctx.suffixComment()

	return ctx.finish(func(span Span, childNodes []Node) *Struct {
		return &Struct{
			span:       span,
			childNodes: childNodes,
			name:       name,
			fields:     fields,
		}
	})
}

func parseStructField(ctx *parseCtx[StructField]) (*StructField, error) {
	name := ctx.ident()
	ctx.space()
	ctx.sigil(T_COLON)
	ctx.space()
	fieldType, _ := parseChild(ctx, parseFieldType)
	ctx.suffixComment()

	return ctx.finish(func(span Span, childNodes []Node) *StructField {
		return &StructField{
			span:       span,
			childNodes: childNodes,
			name:       name,
			fieldType:  fieldType,
		}
	})
}

func parseMessage(ctx *parseCtx[Message]) (*Message, error) {
	if !ctx.tryKeyword("message") {
		return nil, nil
	}
	ctx.space()
	name := ctx.ident()
	ctx.space()

	var fields []*MessageField
	ctx.sigil(T_OPEN_CURL)
	ctx.comments()
	for _ = range ctx.loop {
		if ctx.trySigil(T_CLOSE_CURL) {
			break
		}
		decorators := parseDecorators(ctx)
		field, _ := parseChild(ctx, parseMessageField)
		setDecorators(field, decorators)
		fields = append(fields, field)
		ctx.suffixComment()
		ctx.comments()
	}
	ctx.suffixComment()

	return ctx.finish(func(span Span, childNodes []Node) *Message {
		return &Message{
			span:       span,
			childNodes: childNodes,
			name:       name,
			fields:     fields,
		}
	})
}

func parseMessageField(ctx *parseCtx[MessageField]) (*MessageField, error) {
	name := ctx.ident()
	ctx.space()
	tag, _ := parseChild(ctx, parseFieldTag)
	ctx.space()
	ctx.sigil(T_COLON)
	ctx.space()
	fieldType, _ := parseChild(ctx, parseFieldType)
	ctx.suffixComment()

	return ctx.finish(func(span Span, childNodes []Node) *MessageField {
		return &MessageField{
			span:       span,
			childNodes: childNodes,
			name:       name,
			tag:        tag,
			fieldType:  fieldType,
		}
	})
}

func parseUnion(ctx *parseCtx[Union]) (*Union, error) {
	if !ctx.tryKeyword("union") {
		return nil, nil
	}
	ctx.space()
	name := ctx.ident()
	ctx.space()

	var fields []*UnionField
	ctx.sigil(T_OPEN_CURL)
	ctx.comments()
	for _ = range ctx.loop {
		if ctx.trySigil(T_CLOSE_CURL) {
			break
		}
		decorators := parseDecorators(ctx)
		field, _ := parseChild(ctx, parseUnionField)
		setDecorators(field, decorators)
		fields = append(fields, field)
		ctx.suffixComment()
		ctx.comments()
	}
	ctx.suffixComment()

	return ctx.finish(func(span Span, childNodes []Node) *Union {
		return &Union{
			span:       span,
			childNodes: childNodes,
			name:       name,
			fields:     fields,
		}
	})
}

func parseUnionField(ctx *parseCtx[UnionField]) (*UnionField, error) {
	name := ctx.ident()
	ctx.space()
	tag, _ := parseChild(ctx, parseFieldTag)
	ctx.space()
	ctx.sigil(T_COLON)
	ctx.space()
	fieldType, _ := parseChild(ctx, parseFieldType)
	ctx.suffixComment()

	return ctx.finish(func(span Span, childNodes []Node) *UnionField {
		return &UnionField{
			span:       span,
			childNodes: childNodes,
			name:       name,
			tag:        tag,
			fieldType:  fieldType,
		}
	})
}

func parseFieldTag(ctx *parseCtx[Tag]) (*Tag, error) {
	ctx.sigil(T_AT)
	ctx.space()
	value := ctx.int()
	return ctx.finish(func(span Span, childNodes []Node) *Tag {
		return &Tag{
			span:       span,
			childNodes: childNodes,
			value:      value,
		}
	})
}

func parseProtocolTag(ctx *parseCtx[Tag]) (*Tag, error) {
	if !ctx.trySigil(T_AT) {
		return nil, nil
	}
	ctx.space()
	value := ctx.int()
	return ctx.finish(func(span Span, childNodes []Node) *Tag {
		return &Tag{
			span:       span,
			childNodes: childNodes,
			value:      value,
		}
	})
}

func parseFieldType(ctx *parseCtx[FieldType]) (*FieldType, error) {
	typeName, _ := parseChild(ctx, parseTypeName)

	isArray := false
	var arrayLen *IntLit
	if ctx.trySigil(T_OPEN_SQUARE) {
		isArray = true
		ctx.space()
		if !ctx.trySigil(T_CLOSE_SQUARE) {
			arrayLen = ctx.int()
			ctx.space()
			ctx.sigil(T_CLOSE_SQUARE)
		}
	}

	return ctx.finish(func(span Span, childNodes []Node) *FieldType {
		return &FieldType{
			span:       span,
			childNodes: childNodes,
			typeName:   typeName,
			isArray:    isArray,
			arrayLen:   arrayLen,
		}
	})
}

func parseProtocol(ctx *parseCtx[Protocol]) (*Protocol, error) {
	if !ctx.tryKeyword("protocol") {
		return nil, nil
	}
	ctx.space()
	name := ctx.ident()
	ctx.space()

	var rpcs []*ProtocolRpc
	var events []*ProtocolEvent
	ctx.sigil(T_OPEN_CURL)
	ctx.comments()
	for _ = range ctx.loop {
		if ctx.trySigil(T_CLOSE_CURL) {
			break
		}
		decorators := parseDecorators(ctx)

		rpc, ok := parseChild(ctx, parseProtocolRpc)
		if ok {
			setDecorators(rpc, decorators)
			rpcs = append(rpcs, rpc)
		}
		if !ok && ctx.err == nil {
			var event *ProtocolEvent
			event, ok = parseChild(ctx, parseProtocolEvent)
			if ok {
				setDecorators(event, decorators)
				events = append(events, event)
			}
		}
		if ctx.err != nil {
			return nil, ctx.err
		}
		if !ok {
			return nil, errExpectedProtocolItem(
				ctx.token.Kind,
				string(ctx.readToken()),
				ctx.tokenSpan(),
			)
		}

		ctx.suffixComment()
		ctx.comments()
	}
	ctx.suffixComment()

	return ctx.finish(func(span Span, childNodes []Node) *Protocol {
		return &Protocol{
			span:       span,
			childNodes: childNodes,
			name:       name,
			rpcs:       rpcs,
			events:     events,
		}
	})
}

func parseProtocolRpc(ctx *parseCtx[ProtocolRpc]) (*ProtocolRpc, error) {
	if !ctx.tryKeyword("rpc") {
		return nil, nil
	}
	ctx.space()
	name := ctx.ident()
	ctx.space()
	tag, _ := parseChild(ctx, parseProtocolTag)
	ctx.sigil(T_OPEN_PAREN)
	ctx.space()
	requestType, _ := parseChild(ctx, parseTypeName)
	ctx.space()
	var requestIsStream bool
	if ctx.tryKeyword("stream") {
		requestIsStream = true
		ctx.space()
	}
	ctx.sigil(T_CLOSE_PAREN)

	ctx.space()
	ctx.sigil(T_COLON)
	ctx.space()

	var responseIsStream bool
	var responseType *TypeName
	if ctx.trySigil(T_OPEN_PAREN) {
		ctx.space()

		if err := ctx.ensureToken(); err != nil {
			return nil, err
		}
		if ctx.token.Kind == T_IDENT {
			responseType, _ = parseChild(ctx, parseTypeName)
			ctx.space()
			if ctx.tryKeyword("stream") {
				responseIsStream = true
				ctx.space()
			}
		}
		ctx.sigil(T_CLOSE_PAREN)
	} else {
		responseType, _ = parseChild(ctx, parseTypeName)
	}

	return ctx.finish(func(span Span, childNodes []Node) *ProtocolRpc {
		return &ProtocolRpc{
			span:             span,
			childNodes:       childNodes,
			name:             name,
			tag:              tag,
			requestType:      requestType,
			requestIsStream:  requestIsStream,
			responseType:     responseType,
			responseIsStream: responseIsStream,
		}
	})
}

func parseProtocolEvent(ctx *parseCtx[ProtocolEvent]) (*ProtocolEvent, error) {
	if !ctx.tryKeyword("event") {
		return nil, nil
	}
	ctx.space()
	name := ctx.ident()
	ctx.space()
	tag, _ := parseChild(ctx, parseProtocolTag)
	ctx.sigil(T_COLON)
	ctx.space()
	typeName, _ := parseChild(ctx, parseTypeName)

	return ctx.finish(func(span Span, childNodes []Node) *ProtocolEvent {
		return &ProtocolEvent{
			span:        span,
			childNodes:  childNodes,
			name:        name,
			tag:         tag,
			payloadType: typeName,
		}
	})
}
