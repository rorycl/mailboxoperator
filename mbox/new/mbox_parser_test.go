package main

import (
	"fmt"
	"io"
	"os"
	"testing"
)

func sliceFile(f *os.File, s, e int64) error {
	_, err := f.Seek(s, 0)
	if err != nil {
		return fmt.Errorf("seek err %s", err)
	}
	contents := make([]byte, e-s)
	_, err = f.Read(contents)
	if err != nil {
		return fmt.Errorf("read err %s", err)
	}
	o, err := os.Create("/tmp/sliced.eml")
	if err != nil {
		return fmt.Errorf("create err %s", err)
	}
	defer o.Close()
	_, err = o.Write(contents)
	if err != nil {
		return fmt.Errorf("create err %s", err)
	}
	return nil
}

func TestParser(t *testing.T) {
	filer, err := os.Open("/tmp/netatalk")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	mr := NewMboxReader(filer)
	counter := 0
	atEOF := false
	for {
		r, err := mr.NextMessage()
		if err != nil && err != io.EOF {
			fmt.Println(err)
			t.Fatal(err)
		}
		if err == io.EOF {
			atEOF = true
		}
		x, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		if counter == 55 {
			err := os.WriteFile("/tmp/output.eml", x, 0644)
			if err != nil {
				t.Fatal(err)
			}
		}
		counter++
		if atEOF {
			break
		}
	}
	fmt.Println(counter)

	/*
		lastEmail := offsets[0]
		sliceFile(filer, int64(lastEmail.start), int64(lastEmail.end))
		fmt.Println(len(offsets))
	*/
}
