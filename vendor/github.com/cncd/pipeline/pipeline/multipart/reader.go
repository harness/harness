package multipart

import (
	"bufio"
	"bytes"
	"io"
	"mime/multipart"
	"net/textproto"
)

// Reader is an iterator over parts in a multipart log stream.
type Reader interface {
	// NextPart returns the next part in the multipart or
	// an error. When there are no more parts, the error
	// io.EOF is returned.
	NextPart() (Part, error)
}

// A Part represents a single part in a multipart body.
type Part interface {
	io.Reader

	// Header returns the headers of the body with the
	// keys canonicalized.
	Header() textproto.MIMEHeader

	// FileName returns the filename parameter of the
	// Content-Disposition header.
	FileName() string

	// FormName returns the name parameter if p has a
	// Content-Disposition of type form-data.
	FormName() string
}

// New returns a new multipart Reader.
func New(r io.Reader) Reader {
	buf := bufio.NewReader(r)
	out, _ := buf.Peek(4)

	if bytes.Equal(out, []byte("MIME")) {
		return &multipartReader{
			reader: multipart.NewReader(buf, "boundary"),
		}
	}
	return &textReader{
		reader: r,
	}
}

//
//
//

type multipartReader struct {
	reader *multipart.Reader
}

func (r *multipartReader) NextPart() (Part, error) {
	next, err := r.reader.NextPart()
	if err != nil {
		return nil, err
	}
	part := new(part)
	part.Reader = next
	part.filename = next.FileName()
	part.formname = next.FormName()
	part.header = next.Header
	return part, nil
}

//
//
//

type textReader struct {
	reader io.Reader
	done   bool
}

func (r *textReader) NextPart() (Part, error) {
	if r.done {
		return nil, io.EOF
	}
	r.done = true
	p := new(part)
	p.Reader = r.reader
	p.filename = "terminal.log"
	return p, nil
}

type part struct {
	io.Reader

	filename string
	formname string
	header   textproto.MIMEHeader
}

func (p *part) Header() textproto.MIMEHeader { return p.header }
func (p *part) FileName() string             { return p.filename }
func (p *part) FormName() string             { return p.filename }
