//go:build failingTests
// +build failingTests

package mbox

import (
	"io"
	"testing"
)

func TestMboxFrom(t *testing.T) {

	mbox := "testdata/from.mbox"

	md, err := NewMbox(mbox)
	if err != nil {
		t.Fatal(err)
	}

	counter := 0
	for {
		_, r, err := md.NextReader()
		if err != nil && err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		counter++
	}

	if got, want := counter, 2; got != want {
		t.Errorf("counter got %d want %d", got, want)
	}

}
