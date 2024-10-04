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

package main

import (
	"path/filepath"
	"slices"
)

func splitPath(path string) []string {
	var out []string
	for {
		dir, file := filepath.Split(path)
		if dir == "" {
			out = append(out, file)
			slices.Reverse(out)
			return out
		}
		out = append(out, file)
		path = dir[:len(dir)-1]
	}
}
