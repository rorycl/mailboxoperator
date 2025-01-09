# mailboxoperator

version 0.0.2 : 09 January 2025

MailboxOperator is a golang package for reading the emails in mbox and
maildir format mailboxes and passing each email to a func meeting the
interface `Operator` with the signature `(r io.Reader) error`.

The package reads the provided mailboxes concurrently, and provides
WorkersNum worker goroutines to run the `Operator` function. Shared
resources used by the Operator should be safe for concurrent use. An
error from any one call to the Operator will shut down both the workers
and mailbox producer goroutines and return the first error.

Reading xz, gz and bz2 compressed mbox files is supported transparently.

The package is developed and tested on Linux. Feel free to submit
patches or suggestions.

## Example

```golang
package mailboxoperator

import (
	"fmt"
	"io"
	"log"
	"net/mail"
	"sync"
)

type Counter struct {
	num int
	sync.Mutex
}

// Operate must be concurrent safe
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

	mboxes := []string{"mbox/testdata/golang.mbox", "mbox/testdata/gonuts.mbox"}
	maildirs := []string{"maildir/testdata/example/"}

	mo, err := NewMailboxOperator(mboxes, maildirs, &c)
	if err != nil {
		log.Fatal(err)
	}
	err = mo.Operate()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(c.num)
	// Output: 9
}
```
