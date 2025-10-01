# mailboxoperator

version 0.0.7 : 30 September 2025

MailboxOperator is a golang package for reading the emails in mbox and
maildir format mailboxes and passing each email to a func meeting the
interface `Operator` with the signature `(r io.Reader) error`.

An example cli client that uses MailboxOperator is [mailfinder](https://github.com/rorycl/mailfinder).

The package reads the provided mailboxes concurrently, and provides
`WorkersNum` worker goroutines to run the `Operator` function. Shared
resources used by the Operator should be safe for concurrent use.

Error management from errors arising from normal operation (for example,
an email header that cannot be parsed) is provided by a simple error
wrapper.

Reading xz, gz and bz2 compressed mbox files is supported transparently.

The package includes custom mbox and maildir readers. The maildir reader
is particularly simple-minded. The mbox reader scans for postmark
beginning of record lines using a technique adapted from
[grepmail](https://github.com/coppit/grepmail), but also checks for
preceding null and header lines, to differentiate from postmark lines
quoted in email bodies.

The package is developed and tested on Linux. Feel free to submit
patches or suggestions.

## Example

```golang
import (
	"fmt"
	"io"
	"log"
	"net/mail"
	"sync"

	mbo "github.com/rorycl/mailboxoperator"
)

// counter is a simple struct with mutex protected int
type Counter struct {
	num int
	sync.Mutex
}

// Operate fulfils the mailboxoperator.Operator interface requirement to
// operate on an email. In this case it is using net/mail.ReadMessage,
// but another useful module is github.com/mnako/letters. Mailboxes are
// processed concurrently using NumWorkers worker goroutines.
func (c *Counter) Operate(r io.Reader) error {
	_, err := mail.ReadMessage(r)
	if err != nil {
		return err
	}
	c.Lock()
	defer c.Unlock()
	c.num++
	return nil
}

func Example() {
	c := Counter{}

	// use mailboxes in submodule testdata
	mboxes := []string{"mbox/testdata/golang.mbox", "mbox/testdata/gonuts.mbox"}
	maildirs := []string{"maildir/testdata/example/"}

	// choose an error handler (this one fails on first error)
	eof := mbo.OpErrFatalHandler

	// init operator with mailboxes and counter
	mo, err := mbo.NewMailboxOperator(mboxes, maildirs, &c, eof)
	if err != nil {
		log.Fatal(err)
	}

	// perform the operation
	err = mo.Operate()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(c.num)
	// Output: 9
}
```
