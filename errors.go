package mailboxoperator

// errors provides "error handler" funcs for dealing with email parsing
// errors. The user can choose to catch "operator" errors, which might
// be errors such as address parsing, which perhaps shouldn't permit the
// program to exit, or to exit immediately, or other user-defined
// functions.

import (
	"errors"
	"fmt"
)

// OperatorErrorHandler is a function type for dealing with errors from
// mailboxes. Invocations returning a non-nil error will cause
// mailboxoperator to terminate. User-supplied functions may be supplied.
type OperatorErrorHandler func(error) error

// OpErrPrintHandler prints errors and continues, unless the error is
// not of the anticipated OperationError type.
var OpErrPrintHandler OperatorErrorHandler = func(err error) error {
	var mbo *OperationError
	if errors.As(err, &mbo) {
		// Kind   string // mbox or maildir
		// Path   string // path to mbox or maildir
		// Offset int    // email offset in mbox or maildir
		// Err    error
		fmt.Printf("%s: offset: %d error: %s\n", mbo.Path, mbo.Offset, mbo.Err)
		return nil
	}
	return err
}

// OpErrFatalHandler always returns the error, if any.
var OpErrFatalHandler OperatorErrorHandler = func(err error) error {
	return err
}
