package mailboxoperator

import (
	"errors"
	"testing"
)

func TestOpErrPrintHandler(t *testing.T) {
	var err error
	err = &OperationError{"a", "b", 3, errors.New("test")}
	err = OpErrPrintHandler(err)
	if err != nil {
		t.Fatalf("got non-nil err %s", err)
	}
	err = errors.New("non typed error")
	err = OpErrPrintHandler(err)
	if err == nil {
		t.Fatal("got unexpected nil err")
	}
}

func TestOpErrFatalHandler(t *testing.T) {
	var err error
	err = &OperationError{"a", "b", 3, errors.New("test")}
	err = OpErrFatalHandler(err)
	if err == nil {
		t.Fatal("got unexpected nil err")
	}
}
