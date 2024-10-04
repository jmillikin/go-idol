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
	"encoding/binary"
	"maps"
	"math"
	"slices"
	"sort"
	"strings"

	"go.idol-lang.org/idol"
	"go.idol-lang.org/idol/schema_idl"
	"go.idol-lang.org/idol/syntax"
)

const (
	maxFloat32 = 2 << 23
	maxFloat64 = 2 << 52
)

type declType uint8

const (
	declType_UNKNOWN declType = iota
	declType_CONST
	declType_ENUM
	declType_STRUCT
	declType_MESSAGE
	declType_UNION
	declType_PROTOCOL
)

var builtinTypes = map[string]schema_idl.Type{
	"bool":   schema_idl.Type_BOOL,
	"u8":     schema_idl.Type_U8,
	"i8":     schema_idl.Type_I8,
	"u16":    schema_idl.Type_U16,
	"i16":    schema_idl.Type_I16,
	"u32":    schema_idl.Type_U32,
	"i32":    schema_idl.Type_I32,
	"u64":    schema_idl.Type_U64,
	"i64":    schema_idl.Type_I64,
	"f32":    schema_idl.Type_F32,
	"f64":    schema_idl.Type_F64,
	"text":   schema_idl.Type_TEXT,
	"asciz":  schema_idl.Type_ASCIZ,
	"handle": schema_idl.Type_HANDLE,
}

type CompileOption interface {
	apply(*CompileOptions)
}

type compileOption func(*CompileOptions)

func (f compileOption) apply(opts *CompileOptions) { f(opts) }

type CompileOptions struct {
	deps       *SchemaSet
	sourcePath []string
}

func WithDependencies(dependencies *SchemaSet) CompileOption {
	return compileOption(func(opts *CompileOptions) {
		opts.deps = dependencies
	})
}

func WithSourcePath(sourcePath []string) CompileOption {
	return compileOption(func(opts *CompileOptions) {
		opts.sourcePath = sourcePath
	})
}

type CompileResult struct {
	schema *schema_idl.Schema__Builder

	Errors   []*Error
	Warnings []*Warning
}

func (r *CompileResult) Schema() (schema_idl.Schema, error) {
	// TODO: build directly to a decoded message.
	var buf bytes.Buffer
	if err := idol.EncodeTo(nil, r.schema, &buf); err != nil {
		return schema_idl.Schema{}, err
	}
	return idol.DecodeAs[schema_idl.Schema](nil, buf.Bytes())
}

func (r *CompileResult) SchemaBuilder() *schema_idl.Schema__Builder {
	return r.schema
}

func (r *CompileResult) EncodedSchema() (string, error) {
	var buf strings.Builder
	if err := idol.EncodeTo(nil, r.schema, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func Compile(parsedSchema *syntax.Schema, opts ...CompileOption) CompileResult {
	return NewCompileOptions(opts...).Compile(parsedSchema)
}

func NewCompileOptions(opts ...CompileOption) *CompileOptions {
	compileOptions := &CompileOptions{}
	for _, opt := range opts {
		opt.apply(compileOptions)
	}
	return compileOptions
}

func (opts *CompileOptions) Compile(parsedSchema *syntax.Schema) CompileResult {
	c := compiler{
		opts:   opts,
		nodes:  newSchemaNodes(parsedSchema),
		schema: &schema_idl.Schema__Builder{},
	}
	if len(opts.sourcePath) > 0 {
		c.schema.SourcePath.Set(opts.sourcePath)
	}
	c.compileSchema()
	if len(c.errors) > 0 {
		return CompileResult{
			Errors:   c.errors,
			Warnings: c.warnings,
		}
	}
	return CompileResult{
		schema:   c.schema,
		Warnings: c.warnings,
	}
}

type schemaNodes struct {
	namespace *syntax.Namespace
	imports   []*syntax.Import
	exports   []*syntax.Export
	options   []*syntax.Options

	consts    []*syntax.Const
	enums     []*syntax.Enum
	structs   []*syntax.Struct
	messages  []*syntax.Message
	unions    []*syntax.Union
	protocols []*syntax.Protocol
}

func newSchemaNodes(parsedSchema *syntax.Schema) *schemaNodes {
	nodes := &schemaNodes{}
	for node := range parsedSchema.ChildNodes() {
		switch node := node.(type) {
		case *syntax.Namespace:
			nodes.namespace = node
		case *syntax.Import:
			nodes.imports = append(nodes.imports, node)
		case *syntax.Export:
			nodes.exports = append(nodes.exports, node)
		case *syntax.Options:
			nodes.options = append(nodes.options, node)
		case *syntax.Const:
			nodes.consts = append(nodes.consts, node)
		case *syntax.Enum:
			nodes.enums = append(nodes.enums, node)
		case *syntax.Struct:
			nodes.structs = append(nodes.structs, node)
		case *syntax.Message:
			nodes.messages = append(nodes.messages, node)
		case *syntax.Union:
			nodes.unions = append(nodes.unions, node)
		case *syntax.Protocol:
			nodes.protocols = append(nodes.protocols, node)
		default:
		}
	}
	return nodes
}

type compiler struct {
	opts     *CompileOptions
	nodes    *schemaNodes
	schema   *schema_idl.Schema__Builder
	errors   []*Error
	warnings []*Warning

	// Set by registerImports()
	imports            []*importCtx
	importedNames      map[string]*importedName
	importsByNamespace map[string]*importCtx
	importsByAlias     map[string]*importCtx

	// Set by registerDecls()
	decls       []*declInfo
	declsByName map[string]*declInfo
}

type importCtx struct {
	namespace     string
	usedNames     map[string]struct{}
	unusedAliases map[string]*syntax.Import
}

type importedName struct {
	ictx *importCtx
	node *syntax.Ident
	used bool
}

type typeInfo struct {
	type_    schema_idl.Type
	typeName string
	imported any
	decl     *declInfo
}

type declInfo struct {
	node     declNode
	enumType schema_idl.Type

	// Set by registerConstType()
	constType *typeInfo

	// Set by compileConst()
	constValue []byte

	// Set by compileEnum()
	enumValues map[string]uint64
}

type constInfo struct {
	type_    schema_idl.Type
	typeName string
	value    []byte
}

type declNode interface {
	syntax.Node
	Name() *syntax.Ident
}

type exportInfo struct {
	type_    schema_idl.ExportType
	typeName string
	imported any
}

type builtinOptionsSchema uint8

const (
	_OPTS_NOT_BUILTIN builtinOptionsSchema = iota
	_OPTS_SCHEMA
	_OPTS_CONST
	_OPTS_ENUM
	_OPTS_ENUM_ITEM
	_OPTS_STRUCT
	_OPTS_STRUCT_FIELD
	_OPTS_MESSAGE
	_OPTS_MESSAGE_FIELD
	_OPTS_UNION
	_OPTS_UNION_FIELD
	_OPTS_PROTOCOL
	_OPTS_PROTOCOL_RPC
	_OPTS_PROTOCOL_EVENT
)

func (t *typeInfo) isImported() bool {
	return strings.Contains(t.typeName, "\x1F")
}

func (c *compiler) err(err error) {
	c.errors = append(c.errors, err.(*Error))
}

func (c *compiler) warn(warning *Warning) {
	c.warnings = append(c.warnings, warning)
}

func (c *compiler) compileSchema() {
	namespace, err := checkNamespace(c.nodes.namespace.Namespace())
	if err != nil {
		c.err(err)
	}
	c.schema.Namespace.Set(namespace)

	c.registerImports()
	c.registerDecls()
	c.compileExports()
	if opts := c.compileSchemaOptions(); opts != nil {
		c.schema.Options.Set(opts)
	}
	c.compileDecls()
	c.compileImports()
}

func (c *compiler) registerImports() {
	c.importedNames = make(map[string]*importedName)
	c.importsByNamespace = make(map[string]*importCtx)
	c.importsByAlias = make(map[string]*importCtx)
	for _, node := range c.nodes.imports {
		c.registerImport(node)
	}
}

func (c *compiler) registerImport(node *syntax.Import) {
	namespace, err := checkNamespace(node.Namespace())
	if err != nil {
		c.err(err)
	}

	if c.opts.deps == nil {
		c.opts.deps = &SchemaSet{
			decls: make(map[string]map[string]*mergedDecl),
		}
	}

	importOk := true
	var importDecls map[string]*mergedDecl
	if isCodegenOptions(namespace) {
		importDecls = make(map[string]*mergedDecl)
	} else {
		var ok bool
		importDecls, ok = c.opts.deps.decls[namespace]
		if !ok {
			c.err(errImportNamespaceNotFound(
				namespace,
				node.Namespace().Span(),
			))
			importDecls = make(map[string]*mergedDecl)
			c.opts.deps.decls[namespace] = importDecls
			importOk = false
		}
	}

	ictx, ok := c.importsByNamespace[namespace]
	if !ok {
		ictx = &importCtx{
			namespace:     namespace,
			usedNames:     make(map[string]struct{}),
			unusedAliases: make(map[string]*syntax.Import),
		}
		c.imports = append(c.imports, ictx)
		c.importsByNamespace[namespace] = ictx
	}

	if alias := node.ImportAs(); alias != nil {
		alias := alias.Get()
		if prevImport, conflict := c.importsByAlias[alias]; conflict {
			if prevImport == ictx {
				c.warn(warnDuplicateImportAs(namespace, alias, node.Span()))
			} else {
				c.err(errImportAsConflict(
					prevImport.namespace,
					namespace,
					alias,
					node.Span(),
				))
			}
		} else {
			c.importsByAlias[alias] = ictx
			ictx.unusedAliases[alias] = node
		}
		return
	}

	hasNames := false
	for nameNode := range node.ImportNames() {
		hasNames = true
		name := nameNode.Get()
		if prevImport, conflict := c.importedNames[name]; conflict {
			if prevImport.ictx == ictx {
				c.warn(warnDuplicateImport(
					namespace,
					name,
					nameNode.Span(),
				))
			} else {
				c.err(errImportNameConflict(
					prevImport.ictx.namespace,
					namespace,
					name,
					nameNode.Span(),
				))
			}
		} else {
			if importOk {
				if _, ok := importDecls[name]; !ok {
					c.err(errImportNameNotFound(
						namespace,
						name,
						nameNode.Span(),
					))
					importDecls[name] = &mergedDecl{}
				}
			} else {
				importDecls[name] = &mergedDecl{}
			}
			c.importedNames[name] = &importedName{
				ictx: ictx,
				node: nameNode,
			}
		}
	}

	if !hasNames {
		c.warn(warnEmptyImport(namespace, node.Span()))
	}
}

func (c *compiler) registerDecls() {
	c.declsByName = make(map[string]*declInfo)
	for _, node := range c.nodes.enums {
		if declInfo := c.registerDecl(node); declInfo != nil {
			c.registerEnumType(node, declInfo)
		}
	}
	for _, node := range c.nodes.structs {
		c.registerDecl(node)
	}
	for _, node := range c.nodes.messages {
		c.registerDecl(node)
	}
	for _, node := range c.nodes.unions {
		c.registerDecl(node)
	}
	for _, node := range c.nodes.protocols {
		c.registerDecl(node)
	}

	for _, node := range c.nodes.consts {
		if declInfo := c.registerDecl(node); declInfo != nil {
			c.registerConstType(node, declInfo)
		}
	}
}

func (c *compiler) registerDecl(node declNode) *declInfo {
	declInfo := &declInfo{node: node}
	name := node.Name().Get()
	if prevDecl, conflict := c.declsByName[name]; conflict {
		c.err(errDeclNameConflict(node, prevDecl.node))
	} else {
		c.declsByName[name] = declInfo
	}
	if importedName, conflict := c.importedNames[name]; conflict {
		namespace := importedName.ictx.namespace
		c.err(errDeclNameConflictsWithImport(node, namespace))
	}
	if ictx, conflict := c.importsByAlias[name]; conflict {
		c.err(errDeclNameConflictsWithImportAs(node, ictx.namespace))
	}
	if _, shadow := builtinTypes[name]; shadow {
		c.warn(warnDeclShadowsBuiltin(name, node.Name().Span()))
	}
	c.decls = append(c.decls, declInfo)
	return declInfo
}

func (c *compiler) compileImports() {
	for _, name := range c.importedNames {
		if !name.used {
			c.warn(warnUnusedImport(
				name.ictx.namespace,
				name.node.Get(),
				name.node.Span(),
			))
		}
	}

	imports := make([]*importCtx, 0, len(c.imports))
	for _, ictx := range c.imports {
		for _, alias := range slices.Sorted(maps.Keys(ictx.unusedAliases)) {
			importNode := ictx.unusedAliases[alias]
			c.warn(warnUnusedImportAs(ictx.namespace, alias, importNode.Span()))
		}

		if len(ictx.usedNames) > 0 {
			imports = append(imports, ictx)
		}
	}

	slices.SortFunc(imports, func(a, b *importCtx) int {
		return strings.Compare(a.namespace, b.namespace)
	})

	for _, ictx := range imports {
		names := slices.Collect(maps.Keys(ictx.usedNames))
		sort.Strings(names)
		c.schema.Imports.Add(build(func(b *schema_idl.Import__Builder) {
			b.Namespace.Set(ictx.namespace)
			b.Names.Set(names)
		}))
	}
}

func (c *compiler) compileExports() {
	type exportKey struct {
		type_    schema_idl.ExportType
		typeName string
		exportAs string
	}
	exportDupes := make(map[exportKey]struct{})
	for _, node := range c.nodes.exports {
		var exportNames []*syntax.ExportName
		var exportAs string
		redundantExportAs := false
		if name, exportAsNode := node.ExportAs(); name != nil {
			exportNames = append(exportNames, name)
			exportAs = exportAsNode.Get()
			if name.Name().Get() == exportAs {
				c.warn(warnExportAsSameName(name, exportAs, node.Span()))
				redundantExportAs = true
			}
		} else {
			for name := range node.ExportNames() {
				exportNames = append(exportNames, name)
			}
			if len(exportNames) == 0 {
				c.warn(warnEmptyExport(node.Span()))
				continue
			}
		}

		for _, name := range exportNames {
			resolved, err := c.resolveExport(name)
			if err != nil {
				c.err(err)
			}
			if resolved == nil {
				continue
			}
			key := exportKey{
				type_:    resolved.type_,
				typeName: resolved.typeName,
				exportAs: exportAs,
			}
			if redundantExportAs {
				key.exportAs = ""
			}
			if _, dupe := exportDupes[key]; dupe {
				c.warn(warnDuplicateExport(name))
				continue
			} else {
				exportDupes[key] = struct{}{}
			}
			c.schema.Exports.Add(build(func(b *schema_idl.Export__Builder) {
				b.Type.Set(resolved.type_)
				b.TypeName.Set(resolved.typeName)
				b.ExportAs.Set(exportAs)
			}))
			c.registerExportedDecl(resolved, exportAs)
		}
	}
}

func (c *compiler) registerExportedDecl(declInfo *exportInfo, exportAs string) {
	switch decl := declInfo.imported.(type) {
	case schema_idl.Const:
		cloned := idol.Clone(decl).Self().(*schema_idl.Const__Builder)
		if len(exportAs) != 0 {
			cloned.Name.Set(exportAs)
		}
		c.schema.Consts.Add(cloned)
	case schema_idl.Enum:
		cloned := idol.Clone(decl).Self().(*schema_idl.Enum__Builder)
		if len(exportAs) != 0 {
			cloned.Name.Set(exportAs)
		}
		c.schema.Enums.Add(cloned)
	case schema_idl.Struct:
		cloned := idol.Clone(decl).Self().(*schema_idl.Struct__Builder)
		if len(exportAs) != 0 {
			cloned.Name.Set(exportAs)
		}
		c.schema.Structs.Add(cloned)
	case schema_idl.Message:
		cloned := idol.Clone(decl).Self().(*schema_idl.Message__Builder)
		if len(exportAs) != 0 {
			cloned.Name.Set(exportAs)
		}
		c.schema.Messages.Add(cloned)
	case schema_idl.Union:
		cloned := idol.Clone(decl).Self().(*schema_idl.Union__Builder)
		if len(exportAs) != 0 {
			cloned.Name.Set(exportAs)
		}
		c.schema.Unions.Add(cloned)
	case schema_idl.Protocol:
		cloned := idol.Clone(decl).Self().(*schema_idl.Protocol__Builder)
		if len(exportAs) != 0 {
			cloned.Name.Set(exportAs)
		}
		c.schema.Protocols.Add(cloned)
	default:
		panic("unreachable")
	}
}

func (c *compiler) compileSchemaOptions() *schema_idl.SchemaOptions__Builder {
	var b *schema_idl.SchemaOptions__Builder
	ctx := newOptionsCtx()
	for _, options := range c.nodes.options {
		var schema *optionsSchema
		if schemaNode := options.Schema(); schemaNode != nil {
			schema = c.resolveOptionsSchema(schemaNode)
			if schema == nil {
				continue
			}
		} else {
			schema = &optionsSchema{
				builtin: _OPTS_SCHEMA,
			}
		}
		for option := range options.Options() {
			updateFn := c.compileOption(ctx, schema, option)
			if updateFn != nil {
				if b == nil {
					b = &schema_idl.SchemaOptions__Builder{}
				}
				updateFn(b)
			}
		}
	}
	if len(ctx.uninterpreted.builders) > 0 {
		if b == nil {
			b = &schema_idl.SchemaOptions__Builder{}
		}
		for _, ub := range ctx.uninterpreted.builders {
			b.Uninterpreted.Add(ub)
		}
	}
	return b
}

type optionsCtx struct {
	uninterpreted *uninterpretedOptions
	seen          map[string]map[string][]uint8
}

func newOptionsCtx() *optionsCtx {
	return &optionsCtx{
		uninterpreted: &uninterpretedOptions{
			bySchema: make(map[string]*schema_idl.UninterpretedOptions__Builder),
		},
	}
}

func (ctx *optionsCtx) checkConflict(
	schema, name string,
	value []uint8,
) (bool, bool) {
	if ctx.seen == nil {
		ctx.seen = make(map[string]map[string][]uint8)
	}
	opts, ok := ctx.seen[schema]
	if !ok {
		opts = make(map[string][]uint8)
		ctx.seen[schema] = opts
	}
	if prev, conflict := opts[name]; conflict {
		if bytes.Equal(value, prev) {
			return false, false
		}
		return true, false
	}
	opts[name] = value
	return false, true
}

func (ctx *optionsCtx) getUninterpretedOptionsBuilder(
	schema string,
) *schema_idl.UninterpretedOptions__Builder {
	if b, ok := ctx.uninterpreted.bySchema[schema]; ok {
		return b
	}
	b := &schema_idl.UninterpretedOptions__Builder{}
	if schema != "" {
		b.SchemaType.Set(schema_idl.Type_MESSAGE)
		b.SchemaTypeName.Set(schema)
	}
	ctx.uninterpreted.bySchema[schema] = b
	ctx.uninterpreted.builders = append(ctx.uninterpreted.builders, b)
	return b
}

type uninterpretedOptions struct {
	builders []*schema_idl.UninterpretedOptions__Builder
	bySchema map[string]*schema_idl.UninterpretedOptions__Builder
}

type optionsSchema struct {
	typeName string
	imported *schema_idl.Message
	builtin  builtinOptionsSchema
}

func (s *optionsSchema) isCodegenOptions() bool {
	return strings.HasPrefix(s.typeName, "idol/codegen-options/")
}

func (c *compiler) resolveOptionsSchema(name *syntax.TypeName) *optionsSchema {
	resolved, err := c.resolveType(name, &resolverOptions{
		forOptionsSchema: true,
	})
	if err != nil {
		c.err(err)
		return nil
	}
	if resolved.type_ != schema_idl.Type_MESSAGE {
		c.err(errOptionsSchemaMustBeMessage(name, resolved.type_))
		return nil
	}
	if !resolved.isImported() {
		c.err(errOptionsSchemaMustBeImported(name))
		return nil
	}
	var imported *schema_idl.Message
	if resolved.imported != nil {
		importedMsg := resolved.imported.(schema_idl.Message)
		imported = &importedMsg
	}
	return &optionsSchema{
		typeName: resolved.typeName,
		imported: imported,
	}
}

type optionsUpdater func(any)

func (c *compiler) compileOption(
	ctx *optionsCtx,
	schema *optionsSchema,
	option interface {
		syntax.Node
		Name() *syntax.OptionName
		Value() syntax.Node
	},
) optionsUpdater {
	nameNode := option.Name()
	var nameBuf bytes.Buffer
	nameNode.UnparseTo(&nameBuf)
	name := nameBuf.String()

	var optType *typeInfo
	if !schema.isCodegenOptions() {
		optType = c.resolveOptionType(schema, name, nameNode)
	}

	if optType != nil && schema.builtin != _OPTS_NOT_BUILTIN {
		return c.compileBuiltinOption(schema.builtin, name, option)
	}

	optsBuilder := ctx.getUninterpretedOptionsBuilder(schema.typeName)
	optBuilder := &schema_idl.UninterpretedOption__Builder{}
	optBuilder.Name.Set(name)

	var value []uint8
	if optType == nil {
		var valueBuf bytes.Buffer
		option.Value().UnparseTo(&valueBuf)
		value = valueBuf.Bytes()
	} else {
		var err error
		value, err = c.compileValue(option, optType, option.Value())
		if err != nil {
			c.err(err)
			return nil
		}
		optBuilder.Type.Set(optType.type_)
	}

	if conflict, ok := ctx.checkConflict(schema.typeName, name, value); !ok {
		if conflict {
			c.err(errOptionNameConflict(name, option))
			return nil
		}
		c.warn(warnDuplicateOption(name, option))
		return nil
	}

	optBuilder.Value.SetBytes(value)
	optsBuilder.Options.Add(optBuilder)
	return nil
}

func (c *compiler) resolveOptionType(
	schema *optionsSchema,
	name string,
	nameNode *syntax.OptionName,
) *typeInfo {
	switch schema.builtin {
	case _OPTS_NOT_BUILTIN:
	case _OPTS_MESSAGE_FIELD:
		if name == "optional" {
			return &typeInfo{
				type_: schema_idl.Type_BOOL,
			}
		}
		c.warn(warnOptionNameNotFound(name, nameNode))
		return nil
	default:
		c.warn(warnOptionNameNotFound(name, nameNode))
		return nil
	}

	if schema.imported == nil {
		c.warn(warnOptionNameNotFound(name, nameNode))
		return nil
	}

	schemaMsg := schema.imported
	schemaNS, _, _ := strings.Cut(schema.typeName, "\x1F")
	namePart := name
	for strings.Contains(namePart, ".") {
		var nextField string
		nextField, namePart, _ = strings.Cut(namePart, ".")
		ok := false
		for _, field := range schemaMsg.Fields().Iter() {
			if string(field.Name()) != nextField {
				continue
			}

			var fieldTypeNS, fieldTypeName string
			if ns, localName, ok := strings.Cut(field.TypeName(), "\x1F"); ok {
				fieldTypeNS = ns
				fieldTypeName = localName
			} else {
				fieldTypeNS = schemaNS
				fieldTypeName = field.TypeName()
			}

			resolved, err := c.resolveType2(fieldTypeNS, fieldTypeName, nameNode)
			if err != nil {
				c.err(err)
				return nil
			}
			if resolved.type_ != schema_idl.Type_MESSAGE {
				c.err(errOptionsNameThroughNonMessage(
					nameNode,
					resolved.type_,
					resolved.typeName,
				))
				return nil
			}
			fieldType := resolved.imported.(schema_idl.Message)
			schemaMsg = &fieldType
			schemaNS = fieldTypeNS
			ok = true
			break
		}
		if !ok {
			c.warn(warnOptionNameNotFound(name, nameNode))
			return nil
		}
	}

	for _, field := range schemaMsg.Fields().Iter() {
		if string(field.Name()) != namePart {
			continue
		}
		fieldType := &typeInfo{
			type_:    field.Type(),
			typeName: field.TypeName(),
		}
		if !fieldType.canCompileValue() {
			c.err(errOptionTypeInvalid(field.Type(), field.TypeName()))
			break
		}
		return fieldType
	}

	c.warn(warnOptionNameNotFound(name, nameNode))
	return nil
}

func (c *compiler) compileBuiltinOption(
	schema builtinOptionsSchema,
	name string,
	option interface {
		syntax.Node
		Name() *syntax.OptionName
		Value() syntax.Node
	},
) optionsUpdater {
	valueNode := option.Value()
	switch schema {
	case _OPTS_MESSAGE_FIELD:
		if name == "optional" {
			var value bool
			if valueNode == nil {
				value = true
			} else {
				enumRef, ok := valueNode.(*syntax.EnumRef)
				if !ok {
					optType := &typeInfo{
						type_: schema_idl.Type_BOOL,
					}
					c.err(errValueTypeMismatch(option, optType, valueNode))
					return nil
				}
				name := enumRef.Name().Get()
				if name == "true" {
					value = true
				}
				if name != "false" {
					c.err(errInvalidBoolValue(enumRef))
				}
			}
			if !value {
				return nil
			}
			return optionsUpdater(func(optsBuilder any) {
				b := optsBuilder.(*schema_idl.MessageFieldOptions__Builder)
				b.Optional.Set(true)
			})
		}
	}
	panic("unreachable")
}

func compileDecorators[T any](
	c *compiler,
	builtinSchema builtinOptionsSchema,
	decoratedNode interface {
		Decorators() []*syntax.Decorator
	},
) (*T, *uninterpretedOptions) {
	var builder *T
	ctx := newOptionsCtx()
	for _, decorator := range decoratedNode.Decorators() {
		if options := decorator.GetOptions(); options != nil {
			var schema *optionsSchema
			if schemaNode := options.Schema(); schemaNode != nil {
				schema = c.resolveOptionsSchema(schemaNode)
				if schema == nil {
					continue
				}
			} else {
				schema = &optionsSchema{
					builtin: builtinSchema,
				}
			}
			for option := range options.Options() {
				updateFn := c.compileOption(ctx, schema, option)
				if updateFn != nil {
					if builder == nil {
						var zero T
						builder = &zero
					}
					updateFn(builder)
				}
			}
		}

		if option := decorator.GetOption(); option != nil {
			schema := &optionsSchema{
				builtin: builtinSchema,
			}
			updateFn := c.compileOption(ctx, schema, option)
			if updateFn != nil {
				if builder == nil {
					var zero T
					builder = &zero
				}
				updateFn(builder)
			}
		}
	}
	return builder, ctx.uninterpreted
}

func (c *compiler) compileDecls() {
	for _, decl := range c.decls {
		if node, ok := decl.node.(*syntax.Const); ok {
			if b := c.compileConst(decl, node, false); b != nil {
				c.schema.Consts.Add(b)
			}
		}
	}
	for _, decl := range c.decls {
		if node, ok := decl.node.(*syntax.Enum); ok {
			if b := c.compileEnum(decl, node); b != nil {
				c.schema.Enums.Add(b)
			}
		}
	}
	for _, decl := range c.decls {
		if node, ok := decl.node.(*syntax.Const); ok {
			if b := c.compileConst(decl, node, true); b != nil {
				c.schema.Consts.Add(b)
			}
		}
	}

	for _, decl := range c.decls {
		if node, ok := decl.node.(*syntax.Struct); ok {
			c.schema.Structs.Add(c.compileStruct(node))
		} else if node, ok := decl.node.(*syntax.Message); ok {
			c.schema.Messages.Add(c.compileMessage(node))
		} else if node, ok := decl.node.(*syntax.Union); ok {
			c.schema.Unions.Add(c.compileUnion(node))
		} else if node, ok := decl.node.(*syntax.Protocol); ok {
			c.schema.Protocols.Add(c.compileProtocol(node))
		}
	}
}

func (c *compiler) registerConstType(
	node *syntax.Const,
	declInfo *declInfo,
) {
	typeInfo, err := c.resolveType(node.TypeName(), nil)
	if err != nil {
		c.err(err)
		return
	}
	declInfo.constType = typeInfo
}

func (c *compiler) compileConst(
	declInfo *declInfo,
	node *syntax.Const,
	compilingEnumConsts bool,
) *schema_idl.Const__Builder {
	b := &schema_idl.Const__Builder{}
	b.Name.Set(node.Name().Get())
	if opts := c.compileConstOptions(node); opts != nil {
		b.Options.Set(opts)
	}
	typeInfo := declInfo.constType
	if typeInfo == nil {
		if compilingEnumConsts {
			return nil
		}
		return b
	}
	if compilingEnumConsts != typeInfo.isEnum() {
		return nil
	}
	b.Type.Set(typeInfo.type_)
	b.TypeName.Set(typeInfo.typeName)
	if !typeInfo.canCompileValue() {
		c.err(errConstTypeInvalid(node.TypeName()))
		return b
	}
	if value, err := c.compileValue(node, typeInfo, node.Value()); err == nil {
		b.Value.SetBytes(value)
		declInfo.constValue = value
	} else {
		c.err(err)
	}
	return b
}

func (c *compiler) compileConstOptions(
	node *syntax.Const,
) *schema_idl.ConstOptions__Builder {
	type T = schema_idl.ConstOptions__Builder
	b, uninterpreted := compileDecorators[T](c, _OPTS_CONST, node)
	if len(uninterpreted.builders) > 0 {
		if b == nil {
			b = &schema_idl.ConstOptions__Builder{}
		}
		for _, ub := range uninterpreted.builders {
			b.Uninterpreted.Add(ub)
		}
	}
	return b
}

func (t *typeInfo) canCompileValue() bool {
	switch t.type_ {
	case schema_idl.Type_BOOL:
	case schema_idl.Type_U8:
	case schema_idl.Type_I8:
	case schema_idl.Type_U16:
	case schema_idl.Type_I16:
	case schema_idl.Type_U32:
	case schema_idl.Type_I32:
	case schema_idl.Type_U64:
	case schema_idl.Type_I64:
	case schema_idl.Type_F32:
	case schema_idl.Type_F64:
	case schema_idl.Type_ASCIZ:
	case schema_idl.Type_TEXT:
	default:
		return false
	}
	return true
}

func (t *typeInfo) isEnum() bool {
	if t.typeName == "" {
		return false
	}
	switch t.type_ {
	case schema_idl.Type_U8:
	case schema_idl.Type_I8:
	case schema_idl.Type_U16:
	case schema_idl.Type_I16:
	case schema_idl.Type_U32:
	case schema_idl.Type_I32:
	case schema_idl.Type_U64:
	case schema_idl.Type_I64:
	default:
		return false
	}
	return true
}

func (c *compiler) compileValue(
	dst syntax.Node,
	valueType *typeInfo,
	valueNode syntax.Node,
) ([]byte, error) {
	if valueNode, ok := valueNode.(*syntax.ValueName); ok {
		return c.compileNamedValue(dst, valueType, valueNode)
	}
	if valueType.isEnum() {
		enumRef, ok := valueNode.(*syntax.EnumRef)
		if !ok {
			return nil, errValueTypeMismatch(dst, valueType, valueNode)
		}
		return c.compileEnumRefValue(dst, valueType, valueNode, enumRef.Name().Get())
	}
	switch valueType.type_ {
	case schema_idl.Type_BOOL:
		enumRef, ok := valueNode.(*syntax.EnumRef)
		if !ok {
			return nil, errValueTypeMismatch(dst, valueType, valueNode)
		}
		name := enumRef.Name().Get()
		if name == "true" {
			return []uint8{1}, nil
		}
		if name == "false" {
			return []uint8{0}, nil
		}
		return nil, errInvalidBoolValue(enumRef)
	case schema_idl.Type_U8:
		intLit, ok := valueNode.(*syntax.IntLit)
		if !ok {
			return nil, errValueTypeMismatch(dst, valueType, valueNode)
		}
		if value, ok := intLit.GetUint8(); ok {
			return []uint8{uint8(value)}, nil
		}
		return nil, errValueOutOfRange(valueType.type_, intLit)
	case schema_idl.Type_I8:
		intLit, ok := valueNode.(*syntax.IntLit)
		if !ok {
			return nil, errValueTypeMismatch(dst, valueType, valueNode)
		}
		if value, ok := intLit.GetInt8(); ok {
			return []uint8{uint8(value)}, nil
		}
		return nil, errValueOutOfRange(valueType.type_, intLit)
	case schema_idl.Type_U16:
		intLit, ok := valueNode.(*syntax.IntLit)
		if !ok {
			return nil, errValueTypeMismatch(dst, valueType, valueNode)
		}
		if value, ok := intLit.GetUint16(); ok {
			tmp := make([]uint8, 2)
			binary.LittleEndian.PutUint16(tmp, value)
			return tmp, nil
		}
		return nil, errValueOutOfRange(valueType.type_, intLit)
	case schema_idl.Type_I16:
		intLit, ok := valueNode.(*syntax.IntLit)
		if !ok {
			return nil, errValueTypeMismatch(dst, valueType, valueNode)
		}
		if value, ok := intLit.GetInt16(); ok {
			tmp := make([]uint8, 2)
			binary.LittleEndian.PutUint16(tmp, uint16(value))
			return tmp, nil
		}
		return nil, errValueOutOfRange(valueType.type_, intLit)
	case schema_idl.Type_U32:
		intLit, ok := valueNode.(*syntax.IntLit)
		if !ok {
			return nil, errValueTypeMismatch(dst, valueType, valueNode)
		}
		if value, ok := intLit.GetUint32(); ok {
			tmp := make([]uint8, 4)
			binary.LittleEndian.PutUint32(tmp, value)
			return tmp, nil
		}
		return nil, errValueOutOfRange(valueType.type_, intLit)
	case schema_idl.Type_I32:
		intLit, ok := valueNode.(*syntax.IntLit)
		if !ok {
			return nil, errValueTypeMismatch(dst, valueType, valueNode)
		}
		if value, ok := intLit.GetInt32(); ok {
			tmp := make([]uint8, 4)
			binary.LittleEndian.PutUint32(tmp, uint32(value))
			return tmp, nil
		}
		return nil, errValueOutOfRange(valueType.type_, intLit)
	case schema_idl.Type_U64:
		intLit, ok := valueNode.(*syntax.IntLit)
		if !ok {
			return nil, errValueTypeMismatch(dst, valueType, valueNode)
		}
		if value, ok := intLit.GetUint64(); ok {
			tmp := make([]uint8, 8)
			binary.LittleEndian.PutUint64(tmp, value)
			return tmp, nil
		}
		return nil, errValueOutOfRange(valueType.type_, intLit)
	case schema_idl.Type_I64:
		intLit, ok := valueNode.(*syntax.IntLit)
		if !ok {
			return nil, errValueTypeMismatch(dst, valueType, valueNode)
		}
		if value, ok := intLit.GetInt64(); ok {
			tmp := make([]uint8, 8)
			binary.LittleEndian.PutUint64(tmp, uint64(value))
			return tmp, nil
		}
		return nil, errValueOutOfRange(valueType.type_, intLit)
	case schema_idl.Type_F32:
		intLit, ok := valueNode.(*syntax.IntLit)
		if !ok {
			return nil, errValueTypeMismatch(dst, valueType, valueNode)
		}
		if i64, ok := intLit.GetInt64(); ok {
			if i64 > maxFloat32 || i64 < -maxFloat32 {
				return nil, errValueOutOfRange(valueType.type_, intLit)
			}
			tmp := make([]uint8, 4)
			binary.LittleEndian.PutUint32(tmp, math.Float32bits(float32(i64)))
			return tmp, nil
		}
		u64, _ := intLit.GetUint64()
		if u64 > maxFloat32 {
			return nil, errValueOutOfRange(valueType.type_, intLit)
		}
		tmp := make([]uint8, 4)
		binary.LittleEndian.PutUint32(tmp, math.Float32bits(float32(u64)))
		return tmp, nil
	case schema_idl.Type_F64:
		intLit, ok := valueNode.(*syntax.IntLit)
		if !ok {
			return nil, errValueTypeMismatch(dst, valueType, valueNode)
		}
		if i64, ok := intLit.GetInt64(); ok {
			if i64 > maxFloat64 || i64 < -maxFloat64 {
				return nil, errValueOutOfRange(valueType.type_, intLit)
			}
			tmp := make([]uint8, 8)
			binary.LittleEndian.PutUint64(tmp, math.Float64bits(float64(i64)))
			return tmp, nil
		}
		u64, _ := intLit.GetUint64()
		if u64 > maxFloat64 {
			return nil, errValueOutOfRange(valueType.type_, intLit)
		}
		tmp := make([]uint8, 8)
		binary.LittleEndian.PutUint64(tmp, math.Float64bits(float64(u64)))
		return tmp, nil
	case schema_idl.Type_ASCIZ:
		textLit, ok := valueNode.(*syntax.TextLit)
		if !ok {
			return nil, errValueTypeMismatch(dst, valueType, valueNode)
		}
		if v, ok := textLit.GetAsciz(); ok {
			return append([]byte(v), 0), nil
		}
		return nil, errInvalidAscizValue(textLit)
	case schema_idl.Type_TEXT:
		textLit, ok := valueNode.(*syntax.TextLit)
		if !ok {
			return nil, errValueTypeMismatch(dst, valueType, valueNode)
		}
		if v, ok := textLit.GetText(); ok {
			return []byte(v), nil
		}
		return nil, errInvalidTextValue(textLit)
	default:
		panic("unreachable")
	}
}

func (c *compiler) compileNamedValue(
	dst syntax.Node,
	valueType *typeInfo,
	name *syntax.ValueName,
) ([]uint8, error) {
	const_, err := c.resolveConst(name)
	if err != nil {
		return nil, err
	}

	if valueType.isEnum() {
		var valueTypeNS, valueTypeName string
		if ns, localName, ok := strings.Cut(valueType.typeName, "\x1F"); ok {
			valueTypeNS = ns
			valueTypeName = localName
		} else {
			valueTypeName = valueType.typeName
		}

		var constTypeNS, constTypeName string
		if ns, localName, ok := strings.Cut(const_.typeName, "\x1F"); ok {
			constTypeNS = ns
			constTypeName = localName
		} else {
			constTypeName = const_.typeName
		}

		if valueTypeNS != constTypeNS || valueTypeName != constTypeName {
			// FIXME: update errValueTypeMismatch to show namespaces when
			// there's a type error between enums (or constants?).
			return nil, errValueTypeMismatch(dst, valueType, name)
		}
	}

	if const_.type_ != valueType.type_ {
		// FIXME: It would be helpful to include the actual type (vs just name)
		// of the constant.
		return nil, errValueTypeMismatch(dst, valueType, name)
	}

	v := const_.value
	switch const_.type_ {
	case schema_idl.Type_BOOL:
		if len(v) != 1 || (v[0] != 0x00 && v[0] != 0x01) {
			return nil, errImportedConstantCorrupt()
		}
		return v, nil
	case schema_idl.Type_U8, schema_idl.Type_I8:
		if len(v) != 1 {
			return nil, errImportedConstantCorrupt()
		}
		return v, nil
	case schema_idl.Type_U16, schema_idl.Type_I16:
		if len(v) != 2 {
			return nil, errImportedConstantCorrupt()
		}
		return v, nil
	case schema_idl.Type_U32, schema_idl.Type_I32, schema_idl.Type_F32:
		if len(v) != 4 {
			return nil, errImportedConstantCorrupt()
		}
		return v, nil
	case schema_idl.Type_U64, schema_idl.Type_I64, schema_idl.Type_F64:
		if len(v) != 8 {
			return nil, errImportedConstantCorrupt()
		}
		return v, nil
	case schema_idl.Type_ASCIZ, schema_idl.Type_TEXT:
		// TODO: verify value is valid asciz / text
		//
		// asciz: no internal NUL, ends in NUL
		// text: valid utf8, no internal NUL, ends in NUL
		return v, nil
	default:
		// panic("unreachable")
		return nil, &Error{message: "TODO: compileNamedValue err not type??"}
	}

	return nil, &Error{message: "TODO: compileNamedValue not implemented"}
}

func (c *compiler) compileEnumRefValue(
	dst syntax.Node,
	valueType *typeInfo,
	valueNode syntax.Node,
	name string,
) ([]uint8, error) {
	var enumValues map[string]uint64
	if valueType.imported != nil {
		// FIXME verify valueType imported Enum is the same type as what the
		// const/option expects?
		//
		// need to compare namespaces, etc

		enumValues = make(map[string]uint64)
		// FIXME: is this cast correct? Should it have an error fallback for
		// non-enum ... whatevers?
		for _, item := range valueType.imported.(schema_idl.Enum).Items().Iter() {
			enumValues[item.Name()] = item.Value()
		}
	} else {
		enumValues = valueType.decl.enumValues
	}

	value, ok := enumValues[name]
	if !ok {
		return nil, errEnumRefNotFound()
	}

	span := valueNode.Span()
	switch valueType.type_ {
	case schema_idl.Type_U8:
		if value > math.MaxUint8 {
			return nil, errValueOutOfRange2(valueType.type_, value, span)
		}
		return []uint8{uint8(value)}, nil
	case schema_idl.Type_I8:
		value := int64(value)
		if value < math.MinInt8 || value > math.MaxInt8 {
			return nil, errValueOutOfRange2(valueType.type_, value, span)
		}
		return []uint8{uint8(uint64(value))}, nil
	case schema_idl.Type_U16:
		if value > math.MaxUint16 {
			return nil, errValueOutOfRange2(valueType.type_, value, span)
		}
		tmp := make([]uint8, 2)
		binary.LittleEndian.PutUint16(tmp, uint16(value))
		return tmp, nil
	case schema_idl.Type_I16:
		value := int64(value)
		if value < math.MinInt16 || value > math.MaxInt16 {
			return nil, errValueOutOfRange2(valueType.type_, value, span)
		}
		tmp := make([]uint8, 2)
		binary.LittleEndian.PutUint16(tmp, uint16(uint64(value)))
		return tmp, nil
	case schema_idl.Type_U32:
		if value > math.MaxUint32 {
			return nil, errValueOutOfRange2(valueType.type_, value, span)
		}
		tmp := make([]uint8, 4)
		binary.LittleEndian.PutUint32(tmp, uint32(value))
		return tmp, nil
	case schema_idl.Type_I32:
		value := int64(value)
		if value < math.MinInt32 || value > math.MaxInt32 {
			return nil, errValueOutOfRange2(valueType.type_, value, span)
		}
		tmp := make([]uint8, 4)
		binary.LittleEndian.PutUint32(tmp, uint32(uint32(value)))
		return tmp, nil
	case schema_idl.Type_U64:
		if value > math.MaxUint64 {
			return nil, errValueOutOfRange2(valueType.type_, value, span)
		}
		tmp := make([]uint8, 8)
		binary.LittleEndian.PutUint64(tmp, value)
		return tmp, nil
	case schema_idl.Type_I64:
		value := int64(value)
		if value < math.MinInt64 || value > math.MaxInt64 {
			return nil, errValueOutOfRange2(valueType.type_, value, span)
		}
		tmp := make([]uint8, 8)
		binary.LittleEndian.PutUint64(tmp, uint64(value))
		return tmp, nil
	default:
		panic("unreachable")
	}
}

func (c *compiler) registerEnumType(
	node *syntax.Enum,
	declInfo *declInfo,
) {
	switch node.Type().Get() {
	case "u8":
		declInfo.enumType = schema_idl.Type_U8
	case "i8":
		declInfo.enumType = schema_idl.Type_I8
	case "u16":
		declInfo.enumType = schema_idl.Type_U16
	case "i16":
		declInfo.enumType = schema_idl.Type_I16
	case "u32":
		declInfo.enumType = schema_idl.Type_U32
	case "i32":
		declInfo.enumType = schema_idl.Type_I32
	case "u64":
		declInfo.enumType = schema_idl.Type_U64
	case "i64":
		declInfo.enumType = schema_idl.Type_I64
	default:
		c.err(errEnumTypeInvalid(node.Type()))
	}
}

func (c *compiler) compileEnum(
	declInfo *declInfo,
	node *syntax.Enum,
) *schema_idl.Enum__Builder {
	type pendingAlias struct {
		name       string
		targetName string
		idx        uint32
	}

	enumOpts := c.compileEnumOptions(node)

	valuesByName := make(map[string]uint64)
	aliases := make(map[string]string)
	names := make(map[string]struct{})
	namesByValue := make(map[uint64]string)
	var pendingAliases []pendingAlias

	var items []*schema_idl.EnumItem__Builder
	for _, item := range node.Items() {
		itemBuilder := &schema_idl.EnumItem__Builder{}
		itemName := item.Name().Get()
		itemBuilder.Name.Set(itemName)
		if opts := c.compileEnumItemOptions(item); opts != nil {
			itemBuilder.Options.Set(opts)
		}

		if _, conflict := names[itemName]; conflict {
			var prevValue *uint64
			var prevAlias string
			if prev, ok := valuesByName[itemName]; ok {
				prevValue = &prev
			} else if prev, ok := aliases[itemName]; ok {
				prevAlias = prev
			}
			c.err(errEnumItemNameConflict(
				declInfo.enumType, prevValue, prevAlias, item.Name(),
			))
		}
		names[itemName] = struct{}{}

		var value uint64
		var isAlias bool
		switch valueNode := item.Value().(type) {
		case *syntax.IntLit:
			var ok bool
			switch declInfo.enumType {
			case schema_idl.Type_U8:
				var v uint8
				if v, ok = valueNode.GetUint8(); ok {
					value = uint64(v)
				}
			case schema_idl.Type_U16:
				var v uint16
				if v, ok = valueNode.GetUint16(); ok {
					value = uint64(v)
				}
			case schema_idl.Type_U32:
				var v uint32
				if v, ok = valueNode.GetUint32(); ok {
					value = uint64(v)
				}
			case schema_idl.Type_U64:
				value, ok = valueNode.GetUint64()
			case schema_idl.Type_I8:
				var v int8
				if v, ok = valueNode.GetInt8(); ok {
					value = uint64(uint8(v))
				}
			case schema_idl.Type_I16:
				var v int16
				if v, ok = valueNode.GetInt16(); ok {
					value = uint64(uint16(v))
				}
			case schema_idl.Type_I32:
				var v int32
				if v, ok = valueNode.GetInt32(); ok {
					value = uint64(uint32(v))
				}
			case schema_idl.Type_I64:
				var v int64
				if v, ok = valueNode.GetInt64(); ok {
					value = uint64(v)
				}
			}
			if ok {
				valuesByName[itemName] = value
				if prevName, conflict := namesByValue[value]; conflict {
					c.err(errEnumItemValueConflict(
						declInfo.enumType, value, itemName, prevName,
						item.Value(),
					))
				}
				namesByValue[value] = itemName
			} else {
				c.err(errValueOutOfRange(declInfo.enumType, valueNode))
			}
		case *syntax.EnumRef:
			isAlias = true
			targetName := valueNode.Name().Get()
			if targetValue, ok := valuesByName[targetName]; ok {
				value = targetValue
			} else {
				pendingAliases = append(pendingAliases, pendingAlias{
					name:       itemName,
					targetName: targetName,
					idx:        uint32(len(items)),
				})
			}
			aliases[itemName] = targetName
		case *syntax.ValueName:
			var err error
			value, err = c.resolveEnumItemValue(valueNode, declInfo.enumType)
			if err == nil {
				valuesByName[itemName] = value
				if prevName, conflict := namesByValue[value]; conflict {
					c.err(errEnumItemValueConflict(
						declInfo.enumType, value, itemName, prevName,
						item.Value(),
					))
				}
				namesByValue[value] = itemName
			} else {
				c.err(err)
			}
		default:
			c.err(errEnumValueNotOk())
		}

		itemBuilder.Value.Set(value)
		itemBuilder.IsAlias.Set(isAlias)
		items = append(items, itemBuilder)
	}

	for _, alias := range pendingAliases {
		if targetValue, ok := valuesByName[alias.targetName]; ok {
			items[alias.idx].Value.Set(targetValue)
		} else {
			c.err(errEnumAliasTargetNotFound())
		}
	}

	declInfo.enumValues = valuesByName
	return build(func(b *schema_idl.Enum__Builder) {
		b.Name.Set(node.Name().Get())
		b.Type.Set(declInfo.enumType)
		if enumOpts != nil {
			b.Options.Set(enumOpts)
		}
		for _, item := range items {
			b.Items.Add(item)
		}
	})
}

func (c *compiler) compileEnumOptions(
	node *syntax.Enum,
) *schema_idl.EnumOptions__Builder {
	type T = schema_idl.EnumOptions__Builder
	b, uninterpreted := compileDecorators[T](c, _OPTS_ENUM, node)
	if len(uninterpreted.builders) > 0 {
		if b == nil {
			b = &schema_idl.EnumOptions__Builder{}
		}
		for _, ub := range uninterpreted.builders {
			b.Uninterpreted.Add(ub)
		}
	}
	return b
}

func (c *compiler) compileEnumItemOptions(
	node *syntax.EnumItem,
) *schema_idl.EnumItemOptions__Builder {
	type T = schema_idl.EnumItemOptions__Builder
	b, uninterpreted := compileDecorators[T](c, _OPTS_ENUM_ITEM, node)
	if len(uninterpreted.builders) > 0 {
		if b == nil {
			b = &schema_idl.EnumItemOptions__Builder{}
		}
		for _, ub := range uninterpreted.builders {
			b.Uninterpreted.Add(ub)
		}
	}
	return b
}

func (c *compiler) resolveEnumItemValue(
	name *syntax.ValueName,
	enumType schema_idl.Type,
) (uint64, error) {
	const_, err := c.resolveConst(name)
	if err != nil {
		return 0, err
	}
	if const_.typeName != "" {
		return 0, &Error{message: "TODO: resolveEnumItemValue const must be plain builtin value (not '" + const_.typeName + "')"}
	}

	signed := false
	var value uint64
	switch const_.type_ {
	case schema_idl.Type_I8:
		signed = true
		fallthrough
	case schema_idl.Type_U8:
		if len(const_.value) != 1 {
			return 0, &Error{message: "TODO invalid const value length"}
		}
		value = uint64(const_.value[0])
	case schema_idl.Type_I16:
		signed = true
		fallthrough
	case schema_idl.Type_U16:
		if len(const_.value) != 2 {
			return 0, &Error{message: "TODO invalid const value length"}
		}
		value = uint64(binary.LittleEndian.Uint16(const_.value))
	case schema_idl.Type_I32:
		signed = true
		fallthrough
	case schema_idl.Type_U32:
		if len(const_.value) != 4 {
			return 0, &Error{message: "TODO invalid const value length"}
		}
		value = uint64(binary.LittleEndian.Uint32(const_.value))
	case schema_idl.Type_I64:
		signed = true
		fallthrough
	case schema_idl.Type_U64:
		if len(const_.value) != 8 {
			return 0, &Error{message: "TODO invalid const value length"}
		}
		value = binary.LittleEndian.Uint64(const_.value)
	default:
		return 0, &Error{message: "TODO: resolveEnumItemValue const can't be assigned to enum value"}
	}

	switch enumType {
	case schema_idl.Type_U8:
		if signed && int64(value) < 0 {
			return 0, errValueOutOfRange2(enumType, int64(value), name.Span())
		}
		if value > math.MaxUint8 {
			return 0, errValueOutOfRange2(enumType, value, name.Span())
		}
	case schema_idl.Type_I8:
		value := int64(value)
		if value < math.MinInt8 || value > math.MaxInt8 {
			return 0, errValueOutOfRange2(enumType, value, name.Span())
		}
	case schema_idl.Type_U16:
		if signed && int64(value) < 0 {
			return 0, errValueOutOfRange2(enumType, int64(value), name.Span())
		}
		if value > math.MaxUint16 {
			return 0, errValueOutOfRange2(enumType, value, name.Span())
		}
	case schema_idl.Type_I16:
		value := int64(value)
		if value < math.MinInt16 || value > math.MaxInt16 {
			return 0, errValueOutOfRange2(enumType, value, name.Span())
		}
	case schema_idl.Type_U32:
		if signed && int64(value) < 0 {
			return 0, errValueOutOfRange2(enumType, int64(value), name.Span())
		}
		if value > math.MaxUint32 {
			return 0, errValueOutOfRange2(enumType, value, name.Span())
		}
	case schema_idl.Type_I32:
		value := int64(value)
		if value < math.MinInt32 || value > math.MaxInt32 {
			return 0, errValueOutOfRange2(enumType, value, name.Span())
		}
	case schema_idl.Type_U64:
		if signed && int64(value) < 0 {
			return 0, errValueOutOfRange2(enumType, int64(value), name.Span())
		}
		if value > math.MaxUint64 {
			return 0, errValueOutOfRange2(enumType, value, name.Span())
		}
	case schema_idl.Type_I64:
		value := int64(value)
		if value < math.MinInt64 || value > math.MaxInt64 {
			return 0, errValueOutOfRange2(enumType, value, name.Span())
		}
	default:
		panic("unreachable")
	}

	return value, nil
}

func (c *compiler) compileStruct(
	node *syntax.Struct,
) *schema_idl.Struct__Builder {
	b := &schema_idl.Struct__Builder{}
	b.Name.Set(node.Name().Get())
	if opts := c.compileStructOptions(node); opts != nil {
		b.Options.Set(opts)
	}

	fieldsByName := make(map[string]*syntax.StructField)
	for _, field := range node.Fields() {
		b.Fields.Add(c.compileStructField(field, fieldsByName))
	}
	if len(fieldsByName) == 0 {
		c.err(errStructEmpty(node.Name().Get(), node.Span()))
	}
	return b
}

func (c *compiler) compileStructOptions(
	node *syntax.Struct,
) *schema_idl.StructOptions__Builder {
	type T = schema_idl.StructOptions__Builder
	b, uninterpreted := compileDecorators[T](c, _OPTS_STRUCT, node)
	if len(uninterpreted.builders) > 0 {
		if b == nil {
			b = &schema_idl.StructOptions__Builder{}
		}
		for _, ub := range uninterpreted.builders {
			b.Uninterpreted.Add(ub)
		}
	}
	return b
}

func (c *compiler) compileStructField(
	node *syntax.StructField,
	fieldsByName map[string]*syntax.StructField,
) *schema_idl.StructField__Builder {
	b := &schema_idl.StructField__Builder{}
	if opts := c.compileStructFieldOptions(node); opts != nil {
		b.Options.Set(opts)
	}

	fieldName := node.Name().Get()
	b.Name.Set(fieldName)
	if prev, dupe := fieldsByName[fieldName]; dupe {
		c.err(errFieldNameConflict("Struct", prev, node))
	} else {
		fieldsByName[fieldName] = node
	}

	fieldType := node.FieldType()
	resolved, err := c.resolveType(fieldType.TypeName(), nil)
	if err != nil {
		c.err(err)
		resolved = &typeInfo{}
	}
	b.Type.Set(resolved.type_)
	b.TypeName.Set(resolved.typeName)

	arrayLen := c.checkFieldArrayLen(fieldType, true)
	b.ArrayLen.Set(arrayLen)

	return b
}

func (c *compiler) compileStructFieldOptions(
	node *syntax.StructField,
) *schema_idl.StructFieldOptions__Builder {
	type T = schema_idl.StructFieldOptions__Builder
	b, uninterpreted := compileDecorators[T](c, _OPTS_STRUCT, node)
	if len(uninterpreted.builders) > 0 {
		if b == nil {
			b = &schema_idl.StructFieldOptions__Builder{}
		}
		for _, ub := range uninterpreted.builders {
			b.Uninterpreted.Add(ub)
		}
	}
	return b
}

func (c *compiler) compileMessage(
	node *syntax.Message,
) *schema_idl.Message__Builder {
	b := &schema_idl.Message__Builder{}
	b.Name.Set(node.Name().Get())
	if opts := c.compileMessageOptions(node); opts != nil {
		b.Options.Set(opts)
	}

	fieldsByTag := make(map[uint16]*syntax.MessageField)
	fieldsByName := make(map[string]*syntax.MessageField)
	for _, field := range node.Fields() {
		b.Fields.Add(c.compileMessageField(field, fieldsByTag, fieldsByName))
	}
	return b
}

func (c *compiler) compileMessageOptions(
	node *syntax.Message,
) *schema_idl.MessageOptions__Builder {
	type T = schema_idl.MessageOptions__Builder
	b, uninterpreted := compileDecorators[T](c, _OPTS_MESSAGE, node)
	if len(uninterpreted.builders) > 0 {
		if b == nil {
			b = &schema_idl.MessageOptions__Builder{}
		}
		for _, ub := range uninterpreted.builders {
			b.Uninterpreted.Add(ub)
		}
	}
	return b
}

func (c *compiler) compileMessageField(
	node *syntax.MessageField,
	fieldsByTag map[uint16]*syntax.MessageField,
	fieldsByName map[string]*syntax.MessageField,
) *schema_idl.MessageField__Builder {
	b := &schema_idl.MessageField__Builder{}
	if options := c.compileMessageFieldOptions(node); options != nil {
		b.Options.Set(options)
	}

	fieldName := node.Name().Get()
	b.Name.Set(fieldName)
	if prev, dupe := fieldsByName[fieldName]; dupe {
		c.err(errFieldNameConflict("Message", prev, node))
	} else {
		fieldsByName[fieldName] = node
	}

	tag, tagOk := c.checkFieldTag(node)
	if tagOk {
		if prev, dupe := fieldsByTag[tag]; dupe {
			c.err(errFieldTagConflict("Message", tag, prev, node))
		} else {
			fieldsByTag[tag] = node
		}
		b.Tag.Set(tag)
	}

	fieldType := node.FieldType()
	resolved, err := c.resolveType(fieldType.TypeName(), nil)
	if err != nil {
		c.err(err)
		resolved = &typeInfo{}
	}
	b.Type.Set(resolved.type_)
	b.TypeName.Set(resolved.typeName)

	arrayLen := c.checkFieldArrayLen(fieldType, false)
	b.ArrayLen.Set(arrayLen)

	return b
}

func (c *compiler) compileMessageFieldOptions(
	node *syntax.MessageField,
) *schema_idl.MessageFieldOptions__Builder {
	type T = schema_idl.MessageFieldOptions__Builder
	b, uninterpreted := compileDecorators[T](c, _OPTS_MESSAGE_FIELD, node)
	if len(uninterpreted.builders) > 0 {
		if b == nil {
			b = &schema_idl.MessageFieldOptions__Builder{}
		}
		for _, ub := range uninterpreted.builders {
			b.Uninterpreted.Add(ub)
		}
	}
	return b
}

func (c *compiler) compileUnion(
	node *syntax.Union,
) *schema_idl.Union__Builder {
	b := &schema_idl.Union__Builder{}
	b.Name.Set(node.Name().Get())
	if opts := c.compileUnionOptions(node); opts != nil {
		b.Options.Set(opts)
	}

	fieldsByTag := make(map[uint16]*syntax.UnionField)
	fieldsByName := make(map[string]*syntax.UnionField)
	for _, field := range node.Fields() {
		b.Fields.Add(c.compileUnionField(field, fieldsByTag, fieldsByName))
	}
	return b
}

func (c *compiler) compileUnionOptions(
	node *syntax.Union,
) *schema_idl.UnionOptions__Builder {
	type T = schema_idl.UnionOptions__Builder
	b, uninterpreted := compileDecorators[T](c, _OPTS_UNION, node)
	if len(uninterpreted.builders) > 0 {
		if b == nil {
			b = &schema_idl.UnionOptions__Builder{}
		}
		for _, ub := range uninterpreted.builders {
			b.Uninterpreted.Add(ub)
		}
	}
	return b
}

func (c *compiler) compileUnionField(
	node *syntax.UnionField,
	fieldsByTag map[uint16]*syntax.UnionField,
	fieldsByName map[string]*syntax.UnionField,
) *schema_idl.UnionField__Builder {
	b := &schema_idl.UnionField__Builder{}
	if options := c.compileUnionFieldOptions(node); options != nil {
		b.Options.Set(options)
	}

	fieldName := node.Name().Get()
	b.Name.Set(fieldName)
	if prev, dupe := fieldsByName[fieldName]; dupe {
		c.err(errFieldNameConflict("Union", prev, node))
	} else {
		fieldsByName[fieldName] = node
	}

	tag, tagOk := c.checkFieldTag(node)
	if tagOk {
		if prev, dupe := fieldsByTag[tag]; dupe {
			c.err(errFieldTagConflict("Message", tag, prev, node))
		} else {
			fieldsByTag[tag] = node
		}
		b.Tag.Set(tag)
	}

	fieldType := node.FieldType()
	resolved, err := c.resolveType(fieldType.TypeName(), nil)
	if err != nil {
		c.err(err)
		resolved = &typeInfo{}
	}
	b.Type.Set(resolved.type_)
	b.TypeName.Set(resolved.typeName)

	arrayLen := c.checkFieldArrayLen(fieldType, false)
	b.ArrayLen.Set(arrayLen)

	return b
}

func (c *compiler) compileUnionFieldOptions(
	node *syntax.UnionField,
) *schema_idl.UnionFieldOptions__Builder {
	type T = schema_idl.UnionFieldOptions__Builder
	b, uninterpreted := compileDecorators[T](c, _OPTS_UNION_FIELD, node)
	if len(uninterpreted.builders) > 0 {
		if b == nil {
			b = &schema_idl.UnionFieldOptions__Builder{}
		}
		for _, ub := range uninterpreted.builders {
			b.Uninterpreted.Add(ub)
		}
	}
	return b
}

func (c *compiler) checkFieldTag(
	field interface {
		syntax.Node
		Tag() *syntax.Tag
	},
) (uint16, bool) {
	tagNode := field.Tag()
	tag, ok := tagNode.Value().GetUint16()
	if !ok || tag == 0 {
		c.err(errFieldTagOutOfRange(field, tagNode))
		return 0, false
	}
	return tag, true
}

func (c *compiler) checkFieldArrayLen(
	fieldType *syntax.FieldType,
	isStructField bool,
) uint32 {
	if !fieldType.IsArray() {
		return 0
	}
	arrayLenNode := fieldType.ArrayLen()
	if arrayLenNode == nil {
		if isStructField {
			c.err(errStructFieldUnsizedArray())
			return 0
		}
		return math.MaxUint32
	}

	arrayLen, ok := arrayLenNode.GetUint32()
	if !ok {
		c.err(errArrayLenNotU32())
		return 0
	}
	if arrayLen == 0 {
		c.err(errArrayLenZero())
	} else if arrayLen == math.MaxUint32 {
		c.err(errArrayLenMaxU32())
	}
	return arrayLen
}

func (c *compiler) compileProtocol(
	node *syntax.Protocol,
) *schema_idl.Protocol__Builder {
	b := &schema_idl.Protocol__Builder{}
	b.Name.Set(node.Name().Get())

	if options := c.compileProtocolOptions(node); options != nil {
		b.Options.Set(options)
	}

	itemsByName := make(map[string]syntax.Node)
	itemsByTag := make(map[uint64]syntax.Node)
	for rpc := range node.Rpcs() {
		b.Rpcs.Add(c.compileProtocolRpc(rpc, itemsByName, itemsByTag))
	}
	for event := range node.Events() {
		b.Events.Add(c.compileProtocolEvent(event, itemsByName, itemsByTag))
	}
	return b
}

func (c *compiler) compileProtocolOptions(
	node *syntax.Protocol,
) *schema_idl.ProtocolOptions__Builder {
	type T = schema_idl.ProtocolOptions__Builder
	b, uninterpreted := compileDecorators[T](c, _OPTS_PROTOCOL, node)
	if len(uninterpreted.builders) > 0 {
		if b == nil {
			b = &schema_idl.ProtocolOptions__Builder{}
		}
		for _, ub := range uninterpreted.builders {
			b.Uninterpreted.Add(ub)
		}
	}
	return b
}

func (c *compiler) compileProtocolRpc(
	node *syntax.ProtocolRpc,
	itemsByName map[string]syntax.Node,
	itemsByTag map[uint64]syntax.Node,
) *schema_idl.ProtocolRpc__Builder {
	b := &schema_idl.ProtocolRpc__Builder{}
	if opts := c.compileProtocolRpcOptions(node); opts != nil {
		b.Options.Set(opts)
	}

	name := node.Name().Get()
	b.Name.Set(name)
	if prev, conflict := itemsByName[name]; conflict {
		c.err(errProtocolItemNameConflict(node, prev))
	} else {
		itemsByName[name] = node
	}

	var tag uint64
	hasTag := false
	if tagNode := node.Tag(); tagNode != nil {
		var ok bool
		if tag, ok = tagNode.Value().GetUint64(); ok {
			b.Tag.Set(tag)
			hasTag = true
		} else {
			c.err(errProtocolTagOutOfRange(tagNode))
		}
	}

	if hasTag {
		if prev, conflict := itemsByTag[tag]; conflict {
			c.err(errProtocolItemTagConflict(node, prev))
		} else {
			itemsByTag[tag] = node
		}
	}

	requestType, err := c.resolveType(node.RequestType(), nil)
	if err != nil {
		c.err(err)
		requestType = &typeInfo{}
	}

	b.RequestType.Set(requestType.type_)
	b.RequestTypeName.Set(requestType.typeName)
	b.RequestIsStream.Set(node.RequestIsStream())

	var responseType *typeInfo
	if t := node.ResponseType(); t != nil {
		responseType, err = c.resolveType(t, nil)
		if err != nil {
			c.err(err)
			responseType = &typeInfo{}
		}
	} else {
		responseType = &typeInfo{}
	}

	b.ResponseType.Set(responseType.type_)
	b.ResponseTypeName.Set(responseType.typeName)
	b.ResponseIsStream.Set(node.ResponseIsStream())

	return b
}

func (c *compiler) compileProtocolRpcOptions(
	node *syntax.ProtocolRpc,
) *schema_idl.ProtocolRpcOptions__Builder {
	type T = schema_idl.ProtocolRpcOptions__Builder
	b, uninterpreted := compileDecorators[T](c, _OPTS_PROTOCOL_RPC, node)
	if len(uninterpreted.builders) > 0 {
		if b == nil {
			b = &schema_idl.ProtocolRpcOptions__Builder{}
		}
		for _, ub := range uninterpreted.builders {
			b.Uninterpreted.Add(ub)
		}
	}
	return b
}

func (c *compiler) compileProtocolEvent(
	node *syntax.ProtocolEvent,
	itemsByName map[string]syntax.Node,
	itemsByTag map[uint64]syntax.Node,
) *schema_idl.ProtocolEvent__Builder {
	b := &schema_idl.ProtocolEvent__Builder{}
	if opts := c.compileProtocolEventOptions(node); opts != nil {
		b.Options.Set(opts)
	}

	name := node.Name().Get()
	b.Name.Set(name)
	if prev, conflict := itemsByName[name]; conflict {
		c.err(errProtocolItemNameConflict(node, prev))
	} else {
		itemsByName[name] = node
	}

	var tag uint64
	hasTag := false
	if tagNode := node.Tag(); tagNode != nil {
		var ok bool
		if tag, ok = tagNode.Value().GetUint64(); ok {
			b.Tag.Set(tag)
			hasTag = true
		} else {
			c.err(errProtocolTagOutOfRange(tagNode))
		}
	}

	if hasTag {
		if prev, conflict := itemsByTag[tag]; conflict {
			c.err(errProtocolItemTagConflict(node, prev))
		} else {
			itemsByTag[tag] = node
		}
	}

	payloadType, err := c.resolveType(node.PayloadType(), nil)
	if err != nil {
		c.err(err)
		payloadType = &typeInfo{}
	}

	b.PayloadType.Set(payloadType.type_)
	b.PayloadTypeName.Set(payloadType.typeName)

	return b
}

func (c *compiler) compileProtocolEventOptions(
	node *syntax.ProtocolEvent,
) *schema_idl.ProtocolEventOptions__Builder {
	type T = schema_idl.ProtocolEventOptions__Builder
	b, uninterpreted := compileDecorators[T](c, _OPTS_PROTOCOL_EVENT, node)
	if len(uninterpreted.builders) > 0 {
		if b == nil {
			b = &schema_idl.ProtocolEventOptions__Builder{}
		}
		for _, ub := range uninterpreted.builders {
			b.Uninterpreted.Add(ub)
		}
	}
	return b
}

type resolverOptions struct {
	forOptionsSchema bool
}

func isCodegenOptions(namespace string) bool {
	return strings.HasPrefix(namespace, "idol/codegen-options/")
}

func (o *resolverOptions) synthesizeCodegenOptions(namespace string) bool {
	if !o.forOptionsSchema {
		return false
	}
	return strings.HasPrefix(namespace, "idol/codegen-options/")
}

func (c *compiler) resolveType(
	node *syntax.TypeName,
	opts *resolverOptions,
) (*typeInfo, error) {
	name := node.Name().Get()
	if opts == nil {
		opts = &resolverOptions{}
	}

	if scope := node.Scope(); scope != nil {
		ictx, ok := c.importsByAlias[scope.Get()]
		if !ok {
			return nil, errImportAsNotFound(scope.Get(), scope.Span())
		}
		delete(ictx.unusedAliases, scope.Get())

		if opts.synthesizeCodegenOptions(ictx.namespace) {
			ictx.usedNames[name] = struct{}{}
			return &typeInfo{
				type_:    schema_idl.Type_MESSAGE,
				typeName: ictx.namespace + "\x1F" + name,
			}, nil
		}

		decl, err := c.opts.deps.resolveType(
			ictx.namespace,
			name,
			node.Span(),
			node.Span(),
		)
		if err != nil {
			return nil, err
		}
		ictx.usedNames[name] = struct{}{}
		return &typeInfo{
			type_:    decl.type_,
			typeName: ictx.namespace + "\x1F" + name,
			imported: decl.imported,
		}, nil
	}

	if decl, ok := c.declsByName[name]; ok {
		switch decl.node.(type) {
		case *syntax.Enum:
			return &typeInfo{
				type_:    decl.enumType,
				typeName: name,
				decl:     decl,
			}, nil
		case *syntax.Struct:
			return &typeInfo{
				type_:    schema_idl.Type_STRUCT,
				typeName: name,
				decl:     decl,
			}, nil
		case *syntax.Message:
			return &typeInfo{
				type_:    schema_idl.Type_MESSAGE,
				typeName: name,
				decl:     decl,
			}, nil
		case *syntax.Union:
			return &typeInfo{
				type_:    schema_idl.Type_UNION,
				typeName: name,
				decl:     decl,
			}, nil
		case *syntax.Const:
			return nil, errResolvedDeclNotType()
		case *syntax.Protocol:
			return nil, errResolvedDeclNotType()
		default:
			panic("unreachable")
		}
	}

	if importedName, ok := c.importedNames[name]; ok {
		ictx := importedName.ictx

		if opts.synthesizeCodegenOptions(ictx.namespace) {
			importedName.used = true
			ictx.usedNames[name] = struct{}{}
			return &typeInfo{
				type_:    schema_idl.Type_MESSAGE,
				typeName: ictx.namespace + "\x1F" + name,
			}, nil
		}

		decl, err := c.opts.deps.resolveType(
			ictx.namespace,
			name,
			importedName.node.Span(),
			node.Span(),
		)
		if err != nil {
			return nil, err
		}
		importedName.used = true
		ictx.usedNames[name] = struct{}{}
		return &typeInfo{
			type_:    decl.type_,
			typeName: ictx.namespace + "\x1F" + name,
			imported: decl.imported,
		}, nil
	}

	if type_, ok := builtinTypes[name]; ok {
		return &typeInfo{
			type_: type_,
		}, nil
	}

	return nil, errTypeNameNotFound()
}

func (c *compiler) resolveType2(
	namespace, name string,
	node syntax.Node,
) (*typeInfo, error) {
	decl, err := c.opts.deps.resolveType(
		namespace,
		name,
		node.Span(),
		node.Span(),
	)
	if err != nil {
		return nil, err
	}
	return &typeInfo{
		type_:    decl.type_,
		typeName: namespace + "\x1F" + name,
		imported: decl.imported,
	}, nil
}

func (c *compiler) resolveConst(
	node *syntax.ValueName,
) (*constInfo, error) {
	name := node.Name().Get()

	if scope := node.Scope(); scope != nil {
		ictx, ok := c.importsByAlias[scope.Get()]
		if !ok {
			return nil, errImportAsNotFound(scope.Get(), scope.Span())
		}
		delete(ictx.unusedAliases, scope.Get())

		resolved, err := c.opts.deps.resolveConst(
			ictx.namespace,
			name,
			node.Span(),
			node.Span(),
		)
		if err != nil {
			return nil, err
		}
		ictx.usedNames[name] = struct{}{}

		typeName := resolved.TypeName()
		if !strings.Contains(typeName, "\x1F") {
			typeName = ictx.namespace + "\x1F" + typeName
		}
		return &constInfo{
			type_:    resolved.Type(),
			typeName: typeName,
			value:    resolved.Value().Collect(),
		}, nil
	}

	if importedName, ok := c.importedNames[name]; ok {
		ictx := importedName.ictx
		resolved, err := c.opts.deps.resolveConst(
			ictx.namespace,
			name,
			importedName.node.Span(),
			node.Span(),
		)
		if err != nil {
			return nil, err
		}
		importedName.used = true
		ictx.usedNames[name] = struct{}{}

		typeName := resolved.TypeName()
		if !strings.Contains(typeName, "\x1F") {
			typeName = ictx.namespace + "\x1F" + typeName
		}
		return &constInfo{
			type_:    resolved.Type(),
			typeName: typeName,
			value:    resolved.Value().Collect(),
		}, nil
	}

	if decl, ok := c.declsByName[name]; ok {
		switch decl.node.(type) {
		case *syntax.Enum:
			return nil, errResolvedDeclNotConst()
		case *syntax.Struct:
			return nil, errResolvedDeclNotConst()
		case *syntax.Message:
			return nil, errResolvedDeclNotConst()
		case *syntax.Union:
			return nil, errResolvedDeclNotConst()
		case *syntax.Const:
			if decl.constValue == nil {
				return nil, &Error{message: "TODO: resolveConst not compiled yet"}
			}
			return &constInfo{
				type_:    decl.constType.type_,
				typeName: decl.constType.typeName,
				//typeName: decl.node.Name().Get(),
				value: decl.constValue,
			}, nil
		case *syntax.Protocol:
			return nil, errResolvedDeclNotConst()
		default:
			panic("unreachable")
		}
	}

	return nil, errValueNameNotFound()
}

func (c *compiler) resolveExport(node *syntax.ExportName) (*exportInfo, error) {
	name := node.Name().Get()

	if scope := node.Scope(); scope != nil {
		ictx, ok := c.importsByAlias[scope.Get()]
		if !ok {
			return nil, errImportAsNotFound(scope.Get(), scope.Span())
		}
		delete(ictx.unusedAliases, scope.Get())

		decl, err := c.opts.deps.resolveExport(
			ictx.namespace,
			name,
			node.Span(),
		)
		if err != nil {
			return nil, err
		}
		ictx.usedNames[name] = struct{}{}
		decl.typeName = ictx.namespace + "\x1F" + name
		return decl, nil
	}

	if importedName, ok := c.importedNames[name]; ok {
		ictx := importedName.ictx
		decl, err := c.opts.deps.resolveExport(
			ictx.namespace,
			name,
			node.Span(),
		)
		if err != nil {
			return nil, err
		}
		importedName.used = true
		ictx.usedNames[name] = struct{}{}
		decl.typeName = ictx.namespace + "\x1F" + name
		return decl, nil
	}

	if _, ok := c.declsByName[name]; ok {
		c.warn(warnExportLocalDecl(name, node.Span()))
		return nil, nil
	}

	return nil, errExportNameNotFound()
}

func checkNamespace(node *syntax.TextLit) (string, error) {
	namespace, ok := node.GetText()

	invalid := func() (string, error) {
		return namespace, errInvalidNamespace(namespace, node.Span())
	}
	if !ok || namespace == "" {
		return invalid()
	}
	for _, c := range namespace {
		if (c >= 0x20 && c <= 0x7E) || c >= 0x80 || c == 0x09 {
			continue
		}
		return invalid()
	}
	return namespace, nil
}

func build[T any](f func(*T)) *T {
	t := new(T)
	f(t)
	return t
}
