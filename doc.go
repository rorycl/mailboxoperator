// MailboxOperator is a golang package for reading the emails in mbox
// and maildir format mailboxes and passing each email to a func meeting
// the interface `Operator` with the signature `(r io.Reader) error`.
//
// The package reads the provided mailboxes concurrently, and provides
// WorkersNum worker goroutines to run the `Operator` function. Shared
// resources used by the Operator should be safe for concurrent use. An
// error from any one call to the Operator will shut down both the
// workers and mailbox producer goroutines and return the first error.
//
// Reading xz, gz and bz2 compressed mbox files is supported
// transparently.
package mailboxoperator
