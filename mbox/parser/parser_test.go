package parser

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
)

type mboxReader interface {
	NextMessage() (io.Reader, error)
}

// drain drains a MboxReader
func drain(mr mboxReader, t *testing.T) (counter int) {
	var atEOF bool
	for {
		r, err := mr.NextMessage()
		if err != nil && err != io.EOF {
			t.Fatal(err)
		}
		if err == io.EOF {
			atEOF = true
		}
		_, err = io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		counter++
		if atEOF {
			break
		}
	}
	return counter
}

// mailExtractor extracts an email from an mbox
func mailExtractor(mr mboxReader, emailNo int, t *testing.T) []byte {
	var atEOF bool
	counter := 0
	for {
		r, err := mr.NextMessage()
		if err != nil && err != io.EOF {
			t.Fatal(err)
		}
		if err == io.EOF {
			atEOF = true
		}
		if counter == emailNo {
			contents, err := io.ReadAll(r)
			if err != nil {
				t.Fatal(err)
			}
			return contents
		}
		counter++
		if atEOF {
			break
		}
	}
	t.Fatalf("fall through error; perhaps emailNo is wrong (counter %d, emailNo %d)", counter, emailNo)
	return nil
}

func TestFileParserMailARC(t *testing.T) {
	tests := []struct {
		file string
		no   int
	}{
		{
			file: "testdata/mailarc-1.txt",
			no:   16,
		},
		{
			file: "testdata/mailarc-2.txt",
			no:   5,
		},
		{
			file: "testdata/mailarc-3.txt",
			no:   17,
		},
		{
			file: "testdata/mailarc-1-dos.txt", // dos
			no:   16,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("test_%d", i), func(t *testing.T) {
			filer, err := os.Open(tt.file)
			if err != nil {
				t.Fatal(err)
			}

			mr := NewMboxFileReader(filer)
			counter := drain(mr, t)
			if got, want := counter, tt.no; got != want {
				t.Errorf("got %d want %d emails", got, want)
			}

		})
	}
}

func TestFileParserExtractEmail(t *testing.T) {
	tests := []struct {
		file      string
		emailNo   int
		firstLine string
		byteLen   int
	}{
		{
			file:      "testdata/mailarc-1.txt",
			emailNo:   5,
			firstLine: "From someone@imagecraft.com  Fri Jul 10 18:59:26 1998",
			byteLen:   1848,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("test_%d", i), func(t *testing.T) {
			filer, err := os.Open(tt.file)
			if err != nil {
				t.Fatal(err)
			}

			mr := NewMboxFileReader(filer)
			contents := mailExtractor(mr, tt.emailNo, t)
			firstLine, _, found := bytes.Cut(contents, []byte(string("\n")))
			if !found {
				t.Fatal("no newline found in contents")
			}
			if got, want := string(firstLine), tt.firstLine; got != want {
				t.Errorf("got %s want %s firstline for email %d", got, want, tt.emailNo)
			}
			if got, want := len(contents), tt.byteLen; got != want {
				// os.WriteFile("/tmp/tmp.mbox", contents, 0644)
				t.Errorf("got %d want %d bytes for email %d", got, want, tt.emailNo)
			}
		})
	}
}

func TestIOParserMailARC(t *testing.T) {
	tests := []struct {
		file string
		no   int
	}{
		{
			file: "testdata/mailarc-1.txt",
			no:   16,
		},
		{
			file: "testdata/mailarc-2.txt",
			no:   5,
		},
		{
			file: "testdata/mailarc-3.txt",
			no:   17,
		},
		{
			file: "testdata/mailarc-1-dos.txt", // dos
			no:   16,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("test_%d", i), func(t *testing.T) {
			filer, err := os.Open(tt.file)
			if err != nil {
				t.Fatal(err)
			}

			mr := NewMboxIOReader(filer)
			counter := drain(mr, t)
			if got, want := counter, tt.no; got != want {
				t.Errorf("got %d want %d emails", got, want)
			}

		})
	}
}

/*
// sliceFile is a helper func for writing part of a file to outPath.
func sliceFile(f *os.File, s, e int64, outPath string) error {
	_, err := f.Seek(s, 0)
	if err != nil {
		return fmt.Errorf("seek err %s", err)
	}
	contents := make([]byte, e-s)
	_, err = f.Read(contents)
	if err != nil {
		return fmt.Errorf("read err %s", err)
	}
	o, err := os.Create(outPath)
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
*/
