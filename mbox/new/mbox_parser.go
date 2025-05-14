package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
)

type MboxEmail struct {
	start, end int
}

// lineSplitter is a bufio.Split function which is like bufio.SplitLines
// but does not remove "\n" or any preceeding "\r" characters.
func lineSplitter(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		// This is a full newline-terminated line.
		return i + 1, data[0 : i+1], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func lastLineIsNull(b []byte) bool {
	b = bytes.Trim(b, "\r\n ")
	return len(b) == 0
}

// postMarkRegex marks the
//
//	jane@example.com Wed Oct 5 01:06:54 2000
var postMarkRegex *regexp.Regexp = regexp.MustCompile(`^From\s([^ ]+@[^ ]+) (.*\d{2,4})`)

// email header line (not a run-on line)
var emailHeaderLine *regexp.Regexp = regexp.MustCompile(`^([A-Za-z-]+: [^ ]+|>From.*:.*[0-9]{2,4})`)

func scan(filer *os.File) {
	scanner := bufio.NewScanner(filer)
	scanner.Split(lineSplitter)

	counter, start := 0, 0
	lastLine := []byte{}
	justInserted := false

	for scanner.Scan() {
		by := scanner.Bytes()
		previousCounter := counter
		counter += len(by)

		// If an allOffSets entry has been made for the previous line,
		// ensure that this line is an email header line (key: value) or
		// a ">From" line (for older mbox formats), else undo the
		// last insert into allOffsets.
		if justInserted {
			if !emailHeaderLine.Match(by) {
				allOffSets = allOffSets[:len(allOffSets)-1]
			}
			justInserted = false
		}

		// if the line starts with from and the last line is null and it
		// meets the (rough) postmarkregex.
		fmt.Print(string(lastLine))
		if postMarkRegex.Match(by) && lastLineIsNull(lastLine) {
			if previousCounter > 0 {
				allOffSets = append(allOffSets, MboxEmail{start, previousCounter})
				justInserted = true
			}
			start = previousCounter
		}
		copy(lastLine, by)
	}
	allOffSets = append(allOffSets, MboxEmail{start, counter})
	fmt.Println(counter)
}

var allOffSets []MboxEmail = []MboxEmail{}

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

func main() {
	filer, err := os.Open("/tmp/netatalk")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	scan(filer)
	/*
		for i, v := range allOffSets {
			fmt.Println(i, v.start, v.end)
		}
	*/

	// grab second last
	lastEmail := allOffSets[0]
	sliceFile(filer, int64(lastEmail.start), int64(lastEmail.end))
	fmt.Println(len(allOffSets))
}
