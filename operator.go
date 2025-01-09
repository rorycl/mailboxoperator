package mailboxoperator

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/rorycl/mailboxoperator/maildir"
	"github.com/rorycl/mailboxoperator/mailfile"
	"github.com/rorycl/mailboxoperator/mbox"
	"golang.org/x/sync/errgroup"
)

// Operator is an interface for reading emails from mailboxes. Shared
// resources used by the Operator should be concurrent safe. For
// example, saving information to a shared struct should be mutex
// protected.
type Operator interface {
	Operate(io.Reader) error
}

var (
	// WorkersNum is the number of concurrent workers used to process
	// `Operator`.
	WorkersNum int = 8
)

// MailboxOperator is a struct setting out the mailboxes to be processed
// with `Operator`.
type MailboxOperator struct {
	mboxes   []string
	maildirs []string
	operator Operator
}

// NewMailboxOperator creates a new MailboxOperator with the provided
// one or more mbox format files or maildir directories.
func NewMailboxOperator(mboxes []string, maildirs []string, operator Operator) (*MailboxOperator, error) {
	if len(mboxes)+len(maildirs) < 1 {
		return nil, errors.New("no mailboxes or maildirs provided")
	}
	if operator == nil {
		return nil, errors.New("nil operator provided")
	}
	return &MailboxOperator{
		mboxes:   mboxes,
		maildirs: maildirs,
		operator: operator,
	}, nil
}

// Operate performs operations on the emails in each mailbox, exiting
// with an error after the first error, if any. Each mailbox is
// processed concurrently and the `Operator` function run by
// `WorkersNum` goroutines.
func (m *MailboxOperator) Operate() error {
	err := m.process()
	if err != nil {
		return err
	}
	return nil
}

// OperationError is a decorated error describing the mbox/maildir file
// path, file offset and error occurring from a call to Operate.
type OperationError struct {
	Kind   string // mbox or maildir
	Path   string // path to mbox or maildir
	Offset int    // email offset in mbox or maildir
	Err    error
}

func (o OperationError) Error() string {
	return fmt.Sprintf("%s path %s offset %d error: %s", o.Kind, o.Path, o.Offset, o.Err.Error())
}

// mailBytesId passes mail data from the reader to the worker
type mailBytesId struct {
	m   *mailfile.MailFile
	buf *bytes.Buffer
	i   int // this email offset
}

// workers process mail on the reader chan with the Operator.
func (m *MailboxOperator) workers(reader <-chan mailBytesId) <-chan error {

	workerErrChan := make(chan error)
	g := new(errgroup.Group)
	for w := 0; w < WorkersNum; w++ {
		g.Go(func() error {
			for mbi := range reader {

				// run the operator
				err := m.operator.Operate(mbi.buf)
				if err != nil {
					thisErr := OperationError{mbi.m.Kind, mbi.m.Path, mbi.m.No, err}
					workerErrChan <- thisErr
					close(workerErrChan)
					return thisErr
				}
			}
			return nil
		})
	}
	go func() {
		workerErrChan <- g.Wait()
	}()
	return workerErrChan
}

// process processes all mailboxes and maildirs in separate goroutines
// for each feeding the emails to the workers func over the reader chan.
func (m *MailboxOperator) process() error {

	// readNextMail is a common interface for mbox, maildir reading
	type readNextMail interface {
		NextReader() (*mailfile.MailFile, io.Reader, error)
	}

	allMboxesAndMailDirs := []readNextMail{}
	for _, m := range m.mboxes {
		b, err := mbox.NewMbox(m)
		if err != nil {
			return fmt.Errorf("register mbox error: %w", err)
		}
		allMboxesAndMailDirs = append(allMboxesAndMailDirs, b)
	}
	for _, m := range m.maildirs {
		b, err := maildir.NewMailDir(m)
		if err != nil {
			return fmt.Errorf("register maildir error: %w", err)
		}
		allMboxesAndMailDirs = append(allMboxesAndMailDirs, b)
	}

	// reader is a chan for sending emails to workers
	reader := make(chan mailBytesId)

	// initiate email operator workers
	workerErrChan := m.workers(reader)

	// Read each mbox/maildir in a separate goroutine, exiting on first
	// error. Errors from workers are signalled on the workerErrChan,
	// with the first worker error being reported after which the
	// workerErrChan is closed, causing other produer goroutines to
	// exit.
	g := new(errgroup.Group)
	for ii, mm := range allMboxesAndMailDirs {
		g.Go(func() error {
			i := ii
			m := mm
			for {
				// check for error or closed worker chan, exiting in
				// either case
				select {
				case err, ok := <-workerErrChan:
					if err != nil {
						return err
					}
					if !ok {
						return nil
					}
				default:
				}

				n, r, err := m.NextReader()
				if err != nil && err == io.EOF {
					break
				}
				if err != nil {
					return fmt.Errorf("read next mail error: %w", err)
				}
				b := bytes.Buffer{}
				_, err = b.ReadFrom(r)
				if err != nil {
					return fmt.Errorf("buffer error: %w", err)
				}
				reader <- mailBytesId{n, &b, i}
			}
			return nil
		})
	}
	err := g.Wait()
	if err != nil {
		return err
	}
	close(reader) // signal completion to workers

	// wait for workers to complete, possibly with error, if not
	// completed already
	err = <-workerErrChan
	return err
}
