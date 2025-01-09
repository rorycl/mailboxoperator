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
