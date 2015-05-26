// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2012 The Go Authors.  All rights reserved.
// https://github.com/golang/protobuf
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// Verify that the compiler output for test.proto is unchanged.

package testdata

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// sum returns in string form (for easy comparison) the SHA-1 hash of the named file.
func sum(t *testing.T, name string) string {
	data, err := ioutil.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("sum(%q): length is %d", name, len(data))
	hash := sha1.New()
	_, err = hash.Write(data)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("% x", hash.Sum(nil))
}

func run(t *testing.T, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
}

func TestGolden(t *testing.T) {
	// Compute the original checksum.
	goldenSum := sum(t, "my_test/test.pb.go")
	// Run the proto compiler.
	run(t, "protoc", "--go_out="+os.TempDir(), "my_test/test.proto")
	newFile := filepath.Join(os.TempDir(), "my_test/test.pb.go")
	defer os.Remove(newFile)
	// Compute the new checksum.
	newSum := sum(t, newFile)
	// Verify
	if newSum != goldenSum {
		run(t, "diff", "-u", "my_test/test.pb.go", newFile)
		t.Fatal("Code generated by protoc-gen-go has changed; update test.pb.go")
	}
}
