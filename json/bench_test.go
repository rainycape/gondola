// Copyright 2011 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Large data benchmark.
// The JSON data is a summary of agl's changes in the
// go, webkit, and chromium open source projects.
// We benchmark converting between the JSON form
// and in-memory data structures.

package json

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

type codeResponse struct {
	Tree     *codeNode `json:"tree"`
	Username string    `json:"username"`
}

type codeNode struct {
	Name     string      `json:"name"`
	Kids     []*codeNode `json:"kids"`
	CLWeight float64     `json:"cl_weight"`
	Touches  int         `json:"touches"`
	MinT     int64       `json:"min_t"`
	MaxT     int64       `json:"max_t"`
	MeanT    int64       `json:"mean_t"`
}

// This is used to test decoding when the passed in
// struct has methods, since there are additional
// type assertion to perform in that case.
type codeResponseMethod struct {
	Tree     *codeNodeMethod `json:"tree"`
	Username string          `json:"username"`
}

func (c *codeResponseMethod) F() int {
	return 0
}

type codeNodeMethod struct {
	codeNode
}

func (c *codeNodeMethod) F() int {
	return 0
}

type SliceStruct struct {
	Position int   `json:"position"`
	Data     []int `json:"data"`
}

var codeJSON []byte
var codeJSONSlice []byte
var codeStruct codeResponse

func readData(filename string, obj interface{}) []byte {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		panic(err)
	}
	data, err := ioutil.ReadAll(gz)
	if err != nil {
		panic(err)
	}

	if err := Unmarshal(data, obj); err != nil {
		panic(fmt.Errorf("unmarshal %T: %s", obj, err))
	}
	enc, err := Marshal(obj)
	if err != nil {
		panic(fmt.Errorf("marshal %T: %s", obj, err))
	}
	if !bytes.Equal(data, enc) {
		println("different lengths", len(data), len(enc))
		for i := 0; i < len(data) && i < len(enc); i++ {
			if data[i] != enc[i] {
				println("re-marshal: changed at byte", i)
				println("orig: ", string(data[i-10:i+10]))
				println("new: ", string(enc[i-10:i+10]))
				break
			}
		}
		panic("re-marshal code.json: different result")
	}
	return data
}

func codeInit() {
	codeJSON = readData("testdata/code.json.gz", &codeStruct)
	var s []*SliceStruct
	codeJSONSlice = readData("testdata/slice.json.gz", &s)
}

func BenchmarkCodeEncoder(b *testing.B) {
	if codeJSON == nil {
		b.StopTimer()
		codeInit()
		b.StartTimer()
	}
	enc := NewEncoder(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		if err := enc.Encode(&codeStruct); err != nil {
			b.Fatal("Encode:", err)
		}
	}
	b.SetBytes(int64(len(codeJSON)))
}

func BenchmarkCodeMarshal(b *testing.B) {
	if codeJSON == nil {
		b.StopTimer()
		codeInit()
		b.StartTimer()
	}
	for i := 0; i < b.N; i++ {
		if _, err := Marshal(&codeStruct); err != nil {
			b.Fatal("Marshal:", err)
		}
	}
	b.SetBytes(int64(len(codeJSON)))
}

func initializeCodeBenchmark(b *testing.B) {
	if codeJSON == nil {
		codeInit()
	}
	b.ResetTimer()
}

func BenchmarkCodeDecoder(b *testing.B) {
	initializeCodeBenchmark(b)
	var buf bytes.Buffer
	dec := NewDecoder(&buf)
	var r codeResponse
	for i := 0; i < b.N; i++ {
		buf.Write(codeJSON)
		// hide EOF
		buf.WriteByte('\n')
		buf.WriteByte('\n')
		buf.WriteByte('\n')
		if err := dec.Decode(&r); err != nil {
			b.Fatal("Decode:", err)
		}
	}
	b.SetBytes(int64(len(codeJSON)))
}

func BenchmarkCodeUnmarshal(b *testing.B) {
	initializeCodeBenchmark(b)
	for i := 0; i < b.N; i++ {
		var r codeResponse
		if err := Unmarshal(codeJSON, &r); err != nil {
			b.Fatal("Unmmarshal:", err)
		}
	}
	b.SetBytes(int64(len(codeJSON)))
}

func BenchmarkCodeUnmarshalSlice(b *testing.B) {
	initializeCodeBenchmark(b)
	var s []*SliceStruct
	for i := 0; i < b.N; i++ {
		if err := Unmarshal(codeJSONSlice, &s); err != nil {
			b.Fatal("Unmmarshal slice:", err)
		}
	}
	b.SetBytes(int64(len(codeJSONSlice)))
}

func BenchmarkCodeUnmarshalInterface(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var r interface{}
		if err := Unmarshal(codeJSON, &r); err != nil {
			b.Fatal("Unmmarshal:", err)
		}
	}
	b.SetBytes(int64(len(codeJSON)))
}

func BenchmarkCodeUnmarshalMethod(b *testing.B) {
	initializeCodeBenchmark(b)
	for i := 0; i < b.N; i++ {
		var r codeResponseMethod
		if err := Unmarshal(codeJSON, &r); err != nil {
			b.Fatal("Unmmarshal:", err)
		}
	}
	b.SetBytes(int64(len(codeJSON)))
}

func BenchmarkCodeUnmarshalReuse(b *testing.B) {
	initializeCodeBenchmark(b)
	var r codeResponse
	for i := 0; i < b.N; i++ {
		if err := Unmarshal(codeJSON, &r); err != nil {
			b.Fatal("Unmmarshal:", err)
		}
	}
	b.SetBytes(int64(len(codeJSON)))
}

func BenchmarkUnmarshalString(b *testing.B) {
	data := []byte(`"hello, world"`)
	var s string

	for i := 0; i < b.N; i++ {
		if err := Unmarshal(data, &s); err != nil {
			b.Fatal("Unmarshal:", err)
		}
	}
}

func BenchmarkUnmarshalFloat64(b *testing.B) {
	var f float64
	data := []byte(`3.14`)

	for i := 0; i < b.N; i++ {
		if err := Unmarshal(data, &f); err != nil {
			b.Fatal("Unmarshal:", err)
		}
	}
}

func BenchmarkUnmarshalInt64(b *testing.B) {
	var x int64
	data := []byte(`3`)

	for i := 0; i < b.N; i++ {
		if err := Unmarshal(data, &x); err != nil {
			b.Fatal("Unmarshal:", err)
		}
	}
}
