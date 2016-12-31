package form

import (
	"io"
	"mime/multipart"
	"reflect"
)

var (
	fileType = reflect.TypeOf((*File)(nil)).Elem()
)

// File represents a file uploaded using a form. To use files
// in a form struct, just include a field of File type and check
// if it was provided using its IsEmpty() method. Its filename
// can be retrieved using the Filename() method.
type File interface {
	IsEmpty() bool
	Filename() string

	io.Reader
	io.ReaderAt
	io.Seeker
	io.Closer
}

type formFile struct {
	file   multipart.File
	header *multipart.FileHeader
}

func (f *formFile) IsEmpty() bool {
	return f == nil || f.file == nil || f.header == nil
}

func (f *formFile) Filename() string {
	if f.IsEmpty() {
		return ""
	}
	return f.header.Filename
}

func (f *formFile) Read(p []byte) (n int, err error) {
	if f.IsEmpty() {
		return 0, io.EOF
	}
	return f.file.Read(p)
}

// ReadAt implements io.ReaderAt
func (f *formFile) ReadAt(p []byte, off int64) (n int, err error) {
	if f.IsEmpty() {
		return 0, io.EOF
	}
	return f.file.ReadAt(p, off)
}

// Seek implements io.Seeker
func (f *formFile) Seek(offset int64, whence int) (int64, error) {
	if f.IsEmpty() {
		return 0, io.EOF
	}
	return f.file.Seek(offset, whence)
}

func (f *formFile) Close() error {
	if f.IsEmpty() {
		return nil
	}
	return f.file.Close()
}

func (f *formFile) compileTimeCheck() File {
	return f
}
