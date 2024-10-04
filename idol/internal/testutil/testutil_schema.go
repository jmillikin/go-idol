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
	"cmp"
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"testing"

	"go.idol-lang.org/idol/syntax"
)

type SchemaError struct {
	Key     string
	Code    uint32
	Message string
	Pattern *regexp.Regexp
}

func LoadSchemaErrors(testdata fs.FS) (map[string]*SchemaError, error) {
	type raw struct {
		Code    uint32 `json:"code"`
		Message string `json:"message"`
		Pattern string `json:"message_pattern"`
	}

	jsonData, err := fs.ReadFile(testdata, "diagnostics/schema_errors.json")
	if err != nil {
		return nil, err
	}

	var rawErrors map[string]raw
	decoder := json.NewDecoder(bytes.NewReader(jsonData))
	decoder.UseNumber()
	if err := decoder.Decode(&rawErrors); err != nil {
		return nil, err
	}

	out := make(map[string]*SchemaError, len(rawErrors))
	codes := make(map[uint32]struct{}, len(rawErrors))
	for key, raw := range rawErrors {
		if key[0] == '_' {
			if raw.Code != 0 {
				if _, conflict := codes[raw.Code]; conflict {
					return nil, fmt.Errorf("duplicate schema error code %d", raw.Code)
				}
				codes[raw.Code] = struct{}{}
			}
			continue
		}

		if raw.Code == 0 {
			return nil, fmt.Errorf("schema error %q has no error code", key)
		}
		if _, conflict := codes[raw.Code]; conflict {
			return nil, fmt.Errorf("duplicate schema error code %d", raw.Code)
		}
		codes[raw.Code] = struct{}{}

		var pattern *regexp.Regexp
		if raw.Pattern != "" {
			pattern, err = regexp.Compile(raw.Pattern)
			if err != nil {
				return nil, err
			}
		}
		out[key] = &SchemaError{
			Key:     key,
			Code:    raw.Code,
			Message: raw.Message,
			Pattern: pattern,
		}
	}

	return out, nil
}

type SchemaWarning struct {
	Key     string
	Code    uint32
	Message string
	Pattern *regexp.Regexp
}

func LoadSchemaWarnings(testdata fs.FS) (map[string]*SchemaWarning, error) {
	type raw struct {
		Code    uint32 `json:"code"`
		Message string `json:"message"`
		Pattern string `json:"message_pattern"`
	}

	jsonData, err := fs.ReadFile(testdata, "diagnostics/schema_warnings.json")
	if err != nil {
		return nil, err
	}

	var rawWarnings map[string]raw
	decoder := json.NewDecoder(bytes.NewReader(jsonData))
	decoder.UseNumber()
	if err := decoder.Decode(&rawWarnings); err != nil {
		return nil, err
	}

	out := make(map[string]*SchemaWarning, len(rawWarnings))
	codes := make(map[uint32]struct{}, len(rawWarnings))
	for key, raw := range rawWarnings {
		if key[0] == '_' {
			if raw.Code != 0 {
				if _, conflict := codes[raw.Code]; conflict {
					return nil, fmt.Errorf("duplicate schema warning code %d", raw.Code)
				}
				codes[raw.Code] = struct{}{}
			}
			continue
		}

		if raw.Code == 0 {
			return nil, fmt.Errorf("schema warning %q has no error code", key)
		}
		if _, conflict := codes[raw.Code]; conflict {
			return nil, fmt.Errorf("duplicate schema warning code %d", raw.Code)
		}
		codes[raw.Code] = struct{}{}

		var pattern *regexp.Regexp
		if raw.Pattern != "" {
			pattern, err = regexp.Compile("(?i)" + raw.Pattern)
			if err != nil {
				return nil, err
			}
		}
		out[key] = &SchemaWarning{
			Key:     key,
			Code:    raw.Code,
			Message: raw.Message,
			Pattern: pattern,
		}
	}

	return out, nil
}

type ExpectedError struct {
	SchemaError
	Span syntax.Span
}

func LoadExpectedErrors(
	t *testing.T,
	schemaErrors map[string]*SchemaError,
	testdata fs.FS,
	jsonPath string,
) []*ExpectedError {
	t.Helper()

	jsonData, err := fs.ReadFile(testdata, jsonPath)
	if err != nil {
		t.Fatal(err)
	}

	type expectedErrors struct {
		Errors []struct {
			Error string `json:"error"`
			Span  struct {
				Start uint32 `json:"start"`
				Len   uint32 `json:"len"`
			} `json:"error_span"`
		} `json:"errors"`
	}

	var raw expectedErrors
	if err := json.Unmarshal(jsonData, &raw); err != nil {
		t.Fatal(err)
	}

	var out []*ExpectedError
	for _, raw := range raw.Errors {
		if err, ok := schemaErrors[raw.Error]; !ok {
			t.Fatalf("unknown schema error name %q", raw.Error)
		} else {
			out = append(out, &ExpectedError{
				SchemaError: *err,
				Span:        syntax.NewSpan(raw.Span.Start, raw.Span.Len),
			})
		}
	}

	slices.SortFunc(out, func(a, b *ExpectedError) int {
		if x := cmp.Compare(a.Span.Start(), b.Span.Start()); x != 0 {
			return x
		}
		return cmp.Compare(a.Code, b.Code)
	})
	return out
}

type ExpectedWarning struct {
	SchemaWarning
	Span syntax.Span
}

func LoadExpectedWarnings(
	t *testing.T,
	schemaWarnings map[string]*SchemaWarning,
	testdata fs.FS,
	jsonPath string,
) []*ExpectedWarning {
	t.Helper()

	jsonData, err := fs.ReadFile(testdata, jsonPath)
	if err != nil {
		t.Fatal(err)
	}

	type expectedWarnings struct {
		Warnings []struct {
			Warning string `json:"warning"`
			Span    struct {
				Start uint32 `json:"start"`
				Len   uint32 `json:"len"`
			} `json:"warning_span"`
		} `json:"warnings"`
	}

	var raw expectedWarnings
	if err := json.Unmarshal(jsonData, &raw); err != nil {
		t.Fatal(err)
	}

	var out []*ExpectedWarning
	for _, raw := range raw.Warnings {
		if err, ok := schemaWarnings[raw.Warning]; !ok {
			t.Fatalf("unknown schema warning name %q", raw.Warning)
		} else {
			out = append(out, &ExpectedWarning{
				SchemaWarning: *err,
				Span:          syntax.NewSpan(raw.Span.Start, raw.Span.Len),
			})
		}
	}

	slices.SortFunc(out, func(a, b *ExpectedWarning) int {
		if x := cmp.Compare(a.Span.Start(), b.Span.Start()); x != 0 {
			return x
		}
		return cmp.Compare(a.Code, b.Code)
	})
	return out
}
