package mbox

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"testing"
)

func TestMbox(t *testing.T) {

	mboxes := []string{"testdata/golang.mbox", "testdata/golang.mbox.bz2"}

	for _, mailbox := range mboxes {
		md, err := NewMbox(mailbox)
		if err != nil {
			t.Fatal(err)
		}

		counter := 0
		firstFiveLines := ""
		for {
			m, r, err := md.NextReader()
			if err != nil && err == io.EOF {
				break
			}
			if err != nil && err != io.EOF {
				t.Fatal(err)
			}
			counter++
			firstFiveLines = m.Path + "\n"
			b := bufio.NewReader(r)
			for i := 0; i < 5; i++ {
				line, _, err := b.ReadLine()
				if err != nil {
					t.Fatal(err)
				}
				firstFiveLines += string(line) + "\n"
			}
		}

		if got, want := counter, 2; got != want {
			t.Errorf("counter got %d want %d", got, want)
		}

		// check contents of last file
		lastFileHeader := fmt.Sprintf(`
%s
From golang-nuts+bncBAABB5477SUAMGQE2GXYLLA@googlegroups.com Thu Oct  5 20:35:22 2023
Return-path: <golang-nuts+bncBAABB5477SUAMGQE2GXYLLA@googlegroups.com>
Envelope-to: example@test.com
Delivery-date: Thu, 05 Oct 2023 19:35:22 +0000
Received: from mail-oo1-f59.google.com ([209.85.161.59])`, mailbox)

		if got, want := strings.TrimSpace(firstFiveLines), strings.TrimSpace(lastFileHeader); got != want {
			t.Errorf("last file header error\ngot\n%s\nwant\n%s", got, want)
		}
	}
}

func TestMissingMbox(t *testing.T) {
	if _, err := NewMbox("testdata/null"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected fs ErrNotExistErr, got %s", err)
	}
}

func TestInvalidMbox(t *testing.T) {
	_, err := NewMbox("testdata/empty")
	if err == nil || !errors.Is(err, io.EOF) {
		t.Fatal("expected EOF", err)
	}
	fmt.Println(err)
}
