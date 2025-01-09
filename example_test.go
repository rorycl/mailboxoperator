package mailboxoperator_test

import (
	"fmt"
	"io"
	"log"
	"net/mail"
	"sync"
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

	// init operator with mailboxes and counter
	mo, err := NewMailboxOperator(mboxes, maildirs, &c)
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
