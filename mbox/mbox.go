package mbox

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/rorycl/mailboxoperator/mailfile"
	mbox "github.com/rorycl/mailboxoperator/mbox/parser"
	"github.com/rorycl/mailboxoperator/uncompress"
)

// Mbox represents an mbox file on disk with related go-mbox reader and
// email position in the mbox file.
type Mbox struct {
	Path    string
	current int // current message being read
	file    *os.File
	reader  *mbox.MboxIOReader
	atEOF   bool
}

// NewMbox sets up a new mbox for reading
func NewMbox(path string) (*Mbox, error) {
	m := Mbox{}
	var err error
	m.file, err = os.Open(path)
	if err != nil {
		return &m, err
	}
	m.Path = path
	m.current = -1

	// transparent decompression of bzip2, xz and gzip files
	u, err := uncompress.NewReader(m.file)
	if err != nil && errors.Is(err, io.EOF) {
		return &m, fmt.Errorf("%s is an empty mailbox: %w", path, err)
	}
	if err != nil {
		return &m, fmt.Errorf("uncompress error: %w", err)
	}

	m.reader = mbox.NewMboxIOReader(u)
	return &m, err
}

// NextReader returns the next Mail from the reader until exhausted. An
// io.EOF encountered is deferred until the the next call to NextReader.
func (m *Mbox) NextReader() (*mailfile.MailFile, io.Reader, error) {
	if m.atEOF {
		return nil, nil, io.EOF
	}
	m.current++
	thisMail := mailfile.MailFile{
		Kind: "mbox",
		Path: m.Path,
		No:   m.current,
	}
	reader, err := m.reader.NextMessage()
	if err != nil && err == io.EOF {
		m.atEOF = true
		_ = m.file.Close()
		return &thisMail, reader, nil
	}
	return &thisMail, reader, err
}
