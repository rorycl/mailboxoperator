package mailboxoperator

import (
	"errors"
	"io"
	"net/mail"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type counter struct {
	num int
	sync.Mutex
}

func (c *counter) Operate(r io.Reader) error {
	_, err := mail.ReadMessage(r)
	if err != nil {
		return err
	}
	c.Lock()
	defer c.Unlock()
	c.num++
	return nil
}

func TestProcessSimple(t *testing.T) {
	c := counter{}

	maildirs := []string{"maildir/testdata/example/"}
	mboxes := []string{"mbox/testdata/golang.mbox", "mbox/testdata/gonuts.mbox"}

	mo, err := NewMailboxOperator(mboxes, maildirs, &c)
	if err != nil {
		t.Fatal(err)
	}

	err = mo.Operate()
	if err != nil {
		t.Fatal(err)
	}

	if got, want := c.num, 9; got != want {
		t.Errorf("got %d want %d", got, want)
	}
}

type simple int

func (s *simple) Operate(r io.Reader) error {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return err
	}
	_, err = msg.Header.AddressList("From")
	if err != nil {
		return err
	}
	return nil
}

func TestProcessSimpleErr(t *testing.T) {
	var s simple
	maildirs := []string{"maildir/testdata/example/"}
	mboxes := []string{"mbox/testdata/golang.mbox", "mbox/testdata/gonuts.mbox"}

	mo, err := NewMailboxOperator(mboxes, maildirs, &s)
	if err != nil {
		t.Fatal(err)
	}
	err = mo.Operate()
	if err == nil {
		t.Fatal("expected charset error")
	}
	var oe OperationError
	if !errors.As(err, &oe) {
		t.Fatalf("expected OperationError, got %T for %s", err, err)
	}
	errReturned := err.(OperationError)
	if got, want := errReturned.Kind, "maildir"; got != want {
		t.Errorf("OperationError kind got %s want %s", got, want)
	}
	if got, want := errReturned.Offset, 3; got != want {
		t.Errorf("OperationError offset got %d want %d", got, want)
	}
	if got, want := errReturned.Err.Error(), `charset not supported: "koi8-r"`; got != want {
		t.Errorf("OperationError err descriptor got %s want %s", got, want)
	}

	if got, want := err.Error(), `maildir path maildir/testdata/example/cur/1735238277.2023287_9.example_2_s offset 3 error: charset not supported: "koi8-r"`; got != want {
		t.Errorf("Error string\ngot  %s\nwant %s", got, want)
	}
	// fmt.Printf("%#v\n", errReturned)
}

type findFrom struct {
	m map[string]int
	sync.Mutex
}

func (f *findFrom) Operate(r io.Reader) error {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return err
	}
	froms, err := msg.Header.AddressList("From")
	// custom error handling
	if err != nil && strings.Contains(err.Error(), "charset not supported") {
		return nil
	}
	if err != nil {
		return err
	}
	for _, from := range froms {
		f.Lock()
		defer f.Unlock()
		if _, ok := f.m[from.Address]; !ok {
			f.m[from.Address] = 1
		} else {
			f.m[from.Address]++
		}
	}
	return nil
}

func TestProcessNames(t *testing.T) {
	f := findFrom{m: map[string]int{}}
	maildirs := []string{"maildir/testdata/example/"}
	mboxes := []string{"mbox/testdata/golang.mbox", "mbox/testdata/gonuts.mbox"}

	mo, err := NewMailboxOperator(mboxes, maildirs, &f)
	if err != nil {
		t.Fatal(err)
	}
	err = mo.Operate()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{
		"abc@golang.org":      1,
		"announce@golang.org": 2,
		"example@clark.net":   4,
		"example@mindrot.org": 1,
		// one of the nine mails is missing due to charset error skipping
	}
	if got, want := f.m, want; !cmp.Equal(got, want) {
		t.Errorf("froms not as expected %s", cmp.Diff(got, want))
	}
}

type ss struct{}

func (s *ss) Operate(r io.Reader) error {
	return nil
}

func TestNewMailboxOperator(t *testing.T) {
	s := ss{}
	_, err := NewMailboxOperator(nil, nil, &s)
	if err == nil {
		t.Fatal("expected empty mailboxes error", err)
	}
	_, err = NewMailboxOperator([]string{"abc"}, []string{"def"}, nil)
	if err == nil {
		t.Fatal("expected nil operator error", err)
	}
}
