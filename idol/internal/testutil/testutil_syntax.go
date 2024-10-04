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

package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"go.idol-lang.org/idol/syntax"
)

type SyntaxError struct {
	code    uint32
	message string
	pattern *regexp.Regexp
}

func (err *SyntaxError) Code() uint32 {
	return err.code
}

func (err *SyntaxError) Message() string {
	return err.message
}

func (err *SyntaxError) MessagePattern() *regexp.Regexp {
	return err.pattern
}

func LoadSyntaxErrors(testdata fs.FS) (map[string]*SyntaxError, error) {
	type syntaxError struct {
		Code    uint32 `json:"code"`
		Message string `json:"message"`
		Pattern string `json:"message_pattern"`
	}

	jsonData, err := fs.ReadFile(testdata, "diagnostics/syntax_errors.json")
	if err != nil {
		return nil, err
	}

	var rawErrors map[string]syntaxError
	decoder := json.NewDecoder(bytes.NewReader(jsonData))
	decoder.UseNumber()
	if err := decoder.Decode(&rawErrors); err != nil {
		return nil, err
	}

	out := make(map[string]*SyntaxError, len(rawErrors))
	codes := make(map[uint32]struct{}, len(rawErrors))
	for key, raw := range rawErrors {
		if key[0] == '_' {
			if raw.Code != 0 {
				if _, conflict := codes[raw.Code]; conflict {
					return nil, fmt.Errorf("duplicate syntax error code %d", raw.Code)
				}
				codes[raw.Code] = struct{}{}
			}
			continue
		}

		if raw.Code == 0 {
			return nil, fmt.Errorf("syntax error %q has no error code", key)
		}
		if _, conflict := codes[raw.Code]; conflict {
			return nil, fmt.Errorf("duplicate syntax error code %d", raw.Code)
		}
		codes[raw.Code] = struct{}{}

		var pattern *regexp.Regexp
		if raw.Pattern != "" {
			pattern, err = regexp.Compile("(?i)" + raw.Pattern)
			if err != nil {
				return nil, err
			}
		}
		out[key] = &SyntaxError{
			code:    raw.Code,
			message: raw.Message,
			pattern: pattern,
		}
	}

	return out, nil
}

func DumpJSON(node syntax.Node) []byte {
	var buf bytes.Buffer
	dumpJSON(&buf, node, 0)
	return buf.Bytes()
}

func quoteJSON(s string) []byte {
	quoted, _ := json.Marshal(s)
	return quoted
}

func dumpJSON(buf *bytes.Buffer, node syntax.Node, indent int) {
	ty := fmt.Sprintf("%T", node)
	var nameBuf strings.Builder
	for ii, c := range strings.TrimPrefix(ty, "*syntax.") {
		if c >= 'A' && c <= 'Z' {
			if ii > 0 {
				nameBuf.WriteRune('-')
			}
			nameBuf.WriteRune(c + ('a' - 'A'))
		} else {
			nameBuf.WriteRune(c)
		}
	}
	buf.WriteString(strings.Repeat("    ", indent))
	buf.WriteString("{")
	buf.Write(quoteJSON(nameBuf.String()))
	buf.WriteString(": {\n")
	dumpSpanJSON(buf, node.Span(), indent+1)

	switch node := node.(type) {
	case *syntax.Space, *syntax.Newline, *syntax.Sigil, *syntax.Keyword:
		var unparseBuf bytes.Buffer
		node.UnparseTo(&unparseBuf)
		buf.WriteString(",\n")
		buf.WriteString(strings.Repeat("    ", indent+1))
		buf.WriteString("\"unparse\": ")
		buf.Write(quoteJSON(unparseBuf.String()))
		buf.WriteString("")
	case *syntax.Comment:
		buf.WriteString(",\n")
		buf.WriteString(strings.Repeat("    ", indent+1))
		buf.WriteString(`"text": `)
		buf.Write(quoteJSON(node.Text()))
	case *syntax.Ident:
		buf.WriteString(",\n")
		buf.WriteString(strings.Repeat("    ", indent+1))
		buf.WriteString(`"value": `)
		buf.Write(quoteJSON(node.Get()))
	case *syntax.IntLit:
		buf.WriteString(",\n")
		buf.WriteString(strings.Repeat("    ", indent+1))
		buf.WriteString(`"value": `)
		if value, ok := node.GetInt64(); ok {
			buf.WriteString(fmt.Sprintf("%v", value))
		} else {
			value, _ := node.GetUint64()
			buf.WriteString(fmt.Sprintf("%v", value))
		}
	case *syntax.TextLit:
		buf.WriteString(",\n")
		buf.WriteString(strings.Repeat("    ", indent+1))
		buf.WriteString(`"value": `)

		// FIXME
		var value bytes.Buffer
		node.UnparseTo(&value)
		s := value.String()
		buf.Write(quoteJSON(s[1 : len(s)-1]))
	default:
	}

	firstChild := true
	for child := range node.ChildNodes() {
		if firstChild {
			buf.WriteString(",\n")
			buf.WriteString(strings.Repeat("    ", indent+1))
			buf.WriteString("\"child-nodes\": [\n")
		} else {
			buf.WriteString(",\n")
		}
		firstChild = false
		dumpJSON(buf, child, indent+2)
	}
	if !firstChild {
		buf.WriteString("\n")
		buf.WriteString(strings.Repeat("    ", indent+1))
		buf.WriteString("]")
	}
	buf.WriteString("}}")
}

func dumpSpanJSON(buf *bytes.Buffer, span syntax.Span, indent int) {
	buf.WriteString(strings.Repeat("    ", indent))
	buf.WriteString(fmt.Sprintf(
		`"span": {"start": %d, "len": %d}`,
		span.Start(),
		span.Len(),
	))
}
