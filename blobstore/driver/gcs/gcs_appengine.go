// +build appengine

package gcs

import (
	"fmt"
	"io"
	"net/http"

	"gnd.la/blobstore/driver"
	"gnd.la/config"

	"appengine"
	"appengine/blobstore"
	"appengine/file"
)

var (
	defaultBucketName string
)

type gcsDriver struct {
	bucket string
	c      appengine.Context
}

type rfile struct {
	file.FileReader
}

func (f rfile) Metadata() ([]byte, error) {
	return nil, driver.ErrMetadataNotHandled
}

type wfile struct {
	io.WriteCloser
}

func (f wfile) SetMetadata(_ []byte) error {
	return driver.ErrMetadataNotHandled
}

func (d *gcsDriver) path(id string) string {
	return fmt.Sprintf("/gs/%s/%s", d.bucket, id)
}

func (d *gcsDriver) Create(id string) (driver.WFile, error) {
	f, _, err := file.Create(d.c, d.path(id), nil)
	if err != nil {
		return nil, err
	}
	return wfile{f}, nil
}

func (d *gcsDriver) Open(id string) (driver.RFile, error) {
	f, err := file.Open(d.c, d.path(id))
	if err != nil {
		return nil, err
	}
	return rfile{f}, nil
}

func (d *gcsDriver) Remove(id string) error {
	return file.Delete(d.c, d.path(id))
}

func (d *gcsDriver) Close() error {
	return nil
}

func (d *gcsDriver) Serve(w http.ResponseWriter, id string, rng driver.Range) (bool, error) {
	if rng.IsValid() {
		w.Header().Set("X-AppEngine-BlobRange", rng.String())
	}
	key, err := blobstore.BlobKeyForFile(d.c, d.path(id))
	if err != nil {
		return false, err
	}
	blobstore.Send(w, key)
	return true, nil
}

func (d *gcsDriver) SetContext(ctx appengine.Context) {
	d.c = ctx
	if d.bucket == "" {
		if defaultBucketName == "" {
			bucket, err := file.DefaultBucketName(ctx)
			if err != nil {
				panic(fmt.Errorf("no GCS bucket specified and a default could not be found: %s", err))
			}
			defaultBucketName = bucket
		}
		d.bucket = defaultBucketName
	}
}

func gcsOpener(url *config.URL) (driver.Driver, error) {
	value := url.Value
	return &gcsDriver{bucket: value}, nil
}

func init() {
	driver.Register("gcs", gcsOpener)
}
