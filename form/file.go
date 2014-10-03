package form

import (
	"io/ioutil"
	"mime/multipart"
	"reflect"
)

var (
	fileType = reflect.TypeOf(File{})
)

// File represents an uploaded file. The slice fields should
// not be accessed directly, the helper methods should be used
// instead.
type File []interface{}

func (f File) IsEmpty() bool {
	return len(f) == 0
}

func (f File) Header() *multipart.FileHeader {
	if f.IsEmpty() {
		return nil
	}
	return f[1].(*multipart.FileHeader)
}

func (f File) Filename() string {
	if f.IsEmpty() {
		return ""
	}
	return f.Header().Filename
}

func (f File) multipartFile() multipart.File {
	return f[0].(multipart.File)
}

func (f File) Read(p []byte) (n int, err error) {
	return f.multipartFile().Read(p)
}

func (f File) ReadAt(p []byte, off int64) (n int, err error) {
	return f.multipartFile().ReadAt(p, off)
}

func (f File) Seek(offset int64, whence int) (int64, error) {
	return f.multipartFile().Seek(offset, whence)
}

func (f File) Close() error {
	return f.multipartFile().Close()
}

func (f File) ReadAll() ([]byte, error) {
	return ioutil.ReadAll(f)
}
