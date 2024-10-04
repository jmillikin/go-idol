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
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"unsafe"

	"go.idol-lang.org/idol"
	"go.idol-lang.org/idol/codegen_idl"
)

var buffers = make(map[*uint8][]uint8)

//go:export idol_codegen_allocate
func idolCodegenAllocate(len uint32) *uint8 {
	if len > math.MaxInt32 {
		return nil
	}
	buf := make([]uint8, int(len))
	ptr := unsafe.SliceData(buf)
	buffers[ptr] = buf
	return ptr
}

//go:export idol_codegen_deallocate
func idolCodegenDeallocate(ptr *uint8) {
	delete(buffers, ptr)
}

//go:export idol_codegen_generate/go
func idolCodegenGenerateGo(requestPtr *uint8, responsePtrPtr **uint8) uint8 {
	responseBuilder := &codegen_idl.CodegenResponse__Builder{}
	requestLen := binary.LittleEndian.Uint32(unsafe.Slice(requestPtr, 4))
	requestBuf := unsafe.Slice(requestPtr, requestLen)

	if err := idol.Decode[codegen_idl.CodegenRequest](
		&idol.DecodeCtx{},
		requestBuf,
	); err != nil {
		var stderrBuf strings.Builder
		fmt.Fprintf(&stderrBuf, "Encode[CodegenResponse]: %v", err)
		responseBuilder.Error.Set(stderrBuf.String())
		response, _ := idol.Encode(&idol.EncodeCtx{}, responseBuilder)
		responsePtr := unsafe.SliceData(response)
		buffers[responsePtr] = response
		*responsePtrPtr = responsePtr
		return 1
	}

	requestStr := unsafe.String(requestPtr, requestLen)
	request := *(*codegen_idl.CodegenRequest)(unsafe.Pointer(&requestStr))

	c := codegen{
		schema:        request.Schema(),
		dependencies:  request.Dependencies().Collect(),
		pluginOptions: request.PluginOptions().Collect(),
	}

	if err := c.emitSchema(); err != nil {
		responseBuilder.Error.Set(fmt.Sprintf("%v", err))
		response, _ := idol.Encode(&idol.EncodeCtx{}, responseBuilder)
		responsePtr := unsafe.SliceData(response)
		buffers[responsePtr] = response
		*responsePtrPtr = responsePtr
		return 1
	}

	outputBuilder := &codegen_idl.OutputFile__Builder{}
	outputBuilder.Path.Set(strings.Split(c.goPackage+".go", "/"))
	outputBuilder.Content.SetBytes(c.output)
	responseBuilder.OutputFiles.Add(outputBuilder)

	response, err := idol.Encode(&idol.EncodeCtx{}, responseBuilder)
	if err != nil {
		var stderrBuf strings.Builder
		fmt.Fprintf(&stderrBuf, "Encode[CodegenResponse]: %v", err)
		responseBuilder.Error.Set(stderrBuf.String())
		response, _ := idol.Encode(&idol.EncodeCtx{}, responseBuilder)
		responsePtr := unsafe.SliceData(response)
		buffers[responsePtr] = response
		*responsePtrPtr = responsePtr
		return 1
	}

	responsePtr := unsafe.SliceData(response)
	buffers[responsePtr] = response
	*responsePtrPtr = responsePtr
	return 0
}
