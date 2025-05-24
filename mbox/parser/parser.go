// Package parser implementats an mbox parser using "postmark" lines
// separating RFC 2822 format messages, as set out in `man mbox` on most
// unix/linux systems. The man page sets out the postmark line
// definition as follows:
//
//	A postmark line consists of the four characters "From", followed
//	by a space character, followed by the message's envelope sender
//	address, followed by whitespace, and followed by a time stamp.
//	This line is often called From_ line.
//
//	The sender address is expected to be addr-spec as defined in
//	RFC2822 3.4.1. The date is expected to be date-time as output by
//	asctime(3).  For compatibility reasons with legacy software,
//	two-digit years greater than or equal to 70 should be interpreted
//	as the years 1970+, while two-digit years less than 70 should be
//	interpreted as the years 2000-2069. Software reading files in
//	this format should also be prepared to accept non-numeric
//	timezone information such as "CET DST" for Central European Time,
//	daylight saving time.
//
//	Example:
//
//	  From example@example.com Fri Jun 23 02:56:55 2000
//
// To differentiate between lines that look like "From_" separator lines
// and similarly structured content in emails, the program ensures that
// each  postmark line is either preceeded by no lines or an empty line,
// and is followed by either an email header line or (as a special case)
// a ">From" header added from broken older mailing list software.
//
// The parser works with both dos and unix line endings.
//
// No attempt is made to escape bodies of emails depending on the
// mailbox type (be it mboxo, mboxcl, etc.).
package parser

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"regexp"
)

// postMarkRegexp marks the absolute essentials of a "postmark" line
// defining the start of a email entry in a mailbox (irrespective of the
// type of mailbox such as mboxo, mboxcl etc.).
//
//	From jane@example.com Wed Oct 5 01:06:54 2000
//
// grepmail uses https://metacpan.org/pod/Mail::Mbox::MessageParser,
// which has an mbox message "start" pattern defined (in perl) as
//
//	'from_pattern' => q/(?mx)^
//	 (From\s
//	   # Skip names, months, days
//	   (?> [^:\n]+ )
//	   # Match time
//	   (?: :\d\d){1,2}
//	   # Match time zone (EST), hour shift (+0500), and-or year
//	   (?: \s+ (?: [A-Z]{2,6} | [+-]?\d{4} ) ){1,3}
//	   # smail compatibility
//	   (\sremote\sfrom\s.*)?
//	 )/,
//
// This is a simplified version of that pattern, where nsl means
// "non-space letter":
//
//	^From\s        : starting "From "
//	([^ ]+@[^ ]+)  : (email) more than 0 nsl, @, more than 0 nsl, space
//	.*             : filler
//	(:\d\d{1,2})   : (time) one or two :nn time fields
//	.*             : filler
//	([A-Z]{2,6}|
//		  \d{2,4}) : time zone, hour shift or year
var postMarkRegexp *regexp.Regexp = regexp.MustCompile(
	`^From\s([^ ]+@[^ ]+).*(:\d\d){1,2}.*([A-Z]{2,6}|\d{2,4})`,
)

// emailHeaderLineRegexp is a regexp matching a key: value email header line
// or ">From" line from older mailing programs (eg sourceforge). In the
// program it checks that the postmark line is followed by a line with
// this pattern.
var emailHeaderLineRegexp *regexp.Regexp = regexp.MustCompile(
	`^([A-Za-z-]+: [^ ]+|>From.*:.*[0-9]{2,4})`,
)

// fileOffsets are pairs of start/end file byte position markers
type fileOffsets struct {
	start, end int64
}

// MboxFileReader reads emails from an mbox file. This mode of reading
// requires seeking and is an efficient way of reading bytes from a
// large, uncompressed, mbox file.
type MboxFileReader struct {
	file             *os.File
	scanner          *bufio.Scanner
	start            int64 // start position in the file
	counter          int64
	lastLine         []byte
	justInserted     bool
	offsets          []fileOffsets
	atEOF            bool
	msgStart, msgEnd int64 // start and end positions of last message in file
	total            int
}

// NewMboxFileReader creates a new MboxFileReader.
func NewMboxFileReader(file *os.File) *MboxFileReader {
	scanner := bufio.NewScanner(file)
	scanner.Split(lineSplitter)
	mr := &MboxFileReader{
		file:     file,
		scanner:  scanner,
		lastLine: []byte{},
		offsets:  []fileOffsets{},
	}
	return mr
}

// NextMessage progressively provides the next email (as an io.Reader)
// in an mbox until io.EOF. Note that the io.Reader may have valid
// contents if the error is io.EOF.
func (mr *MboxFileReader) NextMessage() (io.Reader, error) {
	_ = mr.scan()
	pos := mr.msgEnd - mr.msgStart
	var err error
	if mr.atEOF {
		err = io.EOF
	}
	mr.total++
	return io.NewSectionReader(mr.file, mr.msgStart, pos), err
}

// scan scans an mbox mailbox to retrieve the stand and end file offsets
// of emails in the mailbox. The scan function keeps track of the line
// preceeding a postmark line and check the line after it is a valid
// header, to help differentiate from postmark lines in the body of
// emails.
func (mr *MboxFileReader) scan() bool {

	// setFilePositions sets the msgStart and msgEnd to the last message
	// offsets.
	setFilePositions := func() {
		lastOffset := mr.offsets[len(mr.offsets)-1]
		mr.msgStart, mr.msgEnd = lastOffset.start, lastOffset.end
	}

	for mr.scanner.Scan() {
		by := mr.scanner.Bytes()
		previousCounter := mr.counter
		mr.counter += int64(len(by))

		// If an offsets entry has been made for the previous line,
		// ensure that this line is an email header line (key: value) or
		// a ">From" line (for older mbox formats), else undo the
		// last insert into allOffsets.
		if mr.justInserted {
			mr.justInserted = false
			if !emailHeaderLineRegexp.Match(by) {
				mr.offsets = mr.offsets[:len(mr.offsets)-1]
			} else {
				setFilePositions()
				return true
			}
		}

		// if the line starts with from and the last line is null and it
		// meets the (rough) postmarkregex.
		if postMarkRegexp.Match(by) && lineIsNull(mr.lastLine) {
			if previousCounter > 0 {
				mr.offsets = append(mr.offsets, fileOffsets{mr.start, previousCounter})
				mr.justInserted = true
			}
			mr.start = previousCounter
		}
		copy(mr.lastLine, by)
	}
	mr.offsets = append(mr.offsets, fileOffsets{mr.start, mr.counter})
	setFilePositions()

	mr.atEOF = true
	return false
}

// MboxIOReader reads reads emails from an mbox io.reader. This mode of
// reading supports reading from streams, but has to buffer the results
// in order to provide complete emails.
type MboxIOReader struct {
	reader       io.Reader
	buf          *bytes.Buffer
	tmp          *bytes.Buffer // temporary line buffer
	scanner      *bufio.Scanner
	lastLine     []byte
	justInserted bool
	atEOF        bool
	total        int
}

// NewMboxIOReader creates an MboxIOReader from an io.Reader.
func NewMboxIOReader(r io.Reader) *MboxIOReader {
	scanner := bufio.NewScanner(r)
	scanner.Split(lineSplitter)
	mr := &MboxIOReader{
		reader:   r,
		buf:      bytes.NewBuffer(nil),
		tmp:      bytes.NewBuffer(nil),
		scanner:  scanner,
		lastLine: []byte{},
	}
	return mr
}

// NextMessage progressively provides the next email (as an io.Reader)
// in an mbox until io.EOF. Note that the io.Reader may have valid
// contents if the error is io.EOF.
func (mr *MboxIOReader) NextMessage() (io.Reader, error) {
	_ = mr.scan()
	var err error
	if mr.atEOF {
		err = io.EOF
	}
	mr.total++
	return mr.buf, err
}

// scan scans an mbox mailbox to retrieve each email in the mailbox in
// bytes. The scan function keeps track of the line preceeding a
// postmark line and check the line after it is a valid header, to help
// differentiate from postmark lines in the body of emails.
func (mr *MboxIOReader) scan() bool {

	mr.buf = bytes.NewBuffer(nil)

	loadBufFromTmp := func() {
		_, _ = mr.buf.Write(mr.tmp.Bytes())
		mr.tmp = bytes.NewBuffer(nil)
	}

	// If mr.tmp holds data, add the data to mr.buf and empty mr.tmp.
	if mr.tmp.Len() > 0 {
		loadBufFromTmp()
	}

	for mr.scanner.Scan() {
		by := mr.scanner.Bytes()

		// If an offsets entry has been made for the previous line,
		// ensure that this line is an email header line (key: value) or
		// a ">From" line (for older mbox formats), else do not start a
		// new email.
		if mr.justInserted {
			mr.justInserted = false
			_, _ = mr.tmp.Write(by)

			// If the last line was not a valid postmark line, put the
			// tmp buffer onto the main buf and continue
			if !emailHeaderLineRegexp.Match(by) {
				loadBufFromTmp()
				continue
			} else {
				return true
			}
		}

		// If the line starts with from and the last line is null and it
		// meets the (rough) postmarkregex and there is already data
		// written in mr.buf, start writing to the tmp line holder to
		// check for a valid header line on the next loop.
		if postMarkRegexp.Match(by) && lineIsNull(mr.lastLine) && mr.buf.Len() > 0 {
			mr.justInserted = true
			_, _ = mr.tmp.Write(by)
			continue
		}

		// add the line to the buffer
		_, _ = mr.buf.Write(by)
		copy(mr.lastLine, by)
	}

	loadBufFromTmp()
	mr.atEOF = true
	return false
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

// lineIsNull checks if a line can be considered a "null line" for mbox
// parsing purposes.
func lineIsNull(b []byte) bool {
	b = bytes.Trim(b, "\r\n ")
	return len(b) == 0
}
