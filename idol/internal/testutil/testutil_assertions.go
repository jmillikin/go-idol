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
	"regexp"
	"slices"
	"testing"

	"github.com/pmezard/go-difflib/difflib"
)

func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("Expected (err != nil), got: nil")
	}
}

func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Expected (err == nil), got: %v", err)
	}
}

func ExpectNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Expected (err == nil), got: %v", err)
	}
}

func ExpectTrue(t *testing.T, cond bool) {
	t.Helper()
	if !cond {
		t.Errorf("Expected (true), got: %v", cond)
	}
}

func ExpectFalse(t *testing.T, cond bool) {
	t.Helper()
	if cond {
		t.Errorf("Expected (false), got: %v", cond)
	}
}

func ExpectEq[T comparable](t *testing.T, want, got T) {
	t.Helper()
	if want != got {
		t.Errorf("Expected %v, got: %v", want, got)
	}
}

func ExpectBytesEq(t *testing.T, want, got []byte) {
	t.Helper()
	if !bytes.Equal(want, got) {
		t.Errorf("Expected %#v, got: %#v", want, got)
	}
}

func ExpectSliceEq[E comparable, S ~[]E](t *testing.T, want, got S) {
	t.Helper()
	if !slices.Equal(want, got) {
		t.Errorf("Expected %#v, got: %#v", want, got)
	}
}

func ExpectMatch[P *regexp.Regexp | string](t *testing.T, want P, got string) {
	t.Helper()
	var pattern *regexp.Regexp
	if p, ok := any(want).(*regexp.Regexp); ok {
		pattern = p
	} else {
		pattern = regexp.MustCompile(any(want).(string))
	}
	if !pattern.MatchString(got) {
		t.Errorf("Expected (match %q), got: %q", pattern.String(), got)
	}
}

func ExpectNoDiff(t *testing.T, a, b string) {
	t.Helper()
	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:       difflib.SplitLines(a),
		B:       difflib.SplitLines(b),
		Context: 5,
	})
	if diff != "" {
		t.Error(diff)
	}
}
