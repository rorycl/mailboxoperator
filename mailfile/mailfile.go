// package mailfile provides shared information about a mail file.
package mailfile

// MailFile represents a Mail file on disk and location of an email
type MailFile struct {
	Kind string // mbox or maildir
	Path string // filepath
	No   int    // the item number in the maildir or mbox
}
