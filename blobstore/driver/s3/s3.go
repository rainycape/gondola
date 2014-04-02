// Package s3 implements an s3 driver for the blobstore.
//
// The URL format for this driver is:
//
//  s3://bucket_name?access_key={key}&secret_key={secret}[&region={region}]
//
// Region is option, but if it's provided, it must be a valid one.
package s3

import (
	"bytes"
	"fmt"
	"gnd.la/blobstore/driver"
	"gnd.la/config"
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
	"strings"
	"sync"
)

// Avoid extra roundtrips to the server to make sure
// the credentials are correct and the bucket exists.
// Also, since the buckets are thread-safe we can share
// them among all connections.
var buckets struct {
	buckets map[string]*s3.Bucket
	sync.RWMutex
}

type rfile bytes.Reader

func (r *rfile) Read(p []byte) (int, error) {
	return (*bytes.Reader)(r).Read(p)
}

func (r *rfile) Seek(offset int64, whence int) (int64, error) {
	return (*bytes.Reader)(r).Seek(offset, whence)
}

func (r *rfile) Close() error {
	return nil
}

type wfile struct {
	id     string
	bucket *s3.Bucket
	buf    bytes.Buffer
}

func (w *wfile) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *wfile) Close() error {
	return w.bucket.Put(w.id, w.buf.Bytes(), "", s3.Private)
}

type s3Driver struct {
	bucket *s3.Bucket
}

func (d *s3Driver) Create(id string) (driver.WFile, error) {
	return &wfile{
		id:     id,
		bucket: d.bucket,
	}, nil
}

func (d *s3Driver) Open(id string) (driver.RFile, error) {
	data, err := d.bucket.Get(id)
	if err != nil {
		return nil, err
	}
	return (*rfile)(bytes.NewReader(data)), nil
}

func (d *s3Driver) Remove(id string) error {
	return d.bucket.Del(id)
}

func (d *s3Driver) Close() error {
	return nil
}

func s3Opener(value string, o config.Options) (driver.Driver, error) {
	accessKey := o.Get("access_key")
	if accessKey == "" {
		return nil, fmt.Errorf("no S3 access key provided")
	}
	secretKey := o.Get("secret_key")
	if secretKey == "" {
		return nil, fmt.Errorf("no S3 secret key provided")
	}
	if value == "" {
		return nil, fmt.Errorf("please, provide a bucket name e.g. s3://mybucket")
	}
	region := aws.USEast
	if r := o.Get("region"); r != "" {
		reg, ok := aws.Regions[r]
		if !ok {
			var regions []string
			for k := range aws.Regions {
				regions = append(regions, fmt.Sprintf("%q", k))
			}
			return nil, fmt.Errorf("invalid S3 region %q. valid regions are %s", r, strings.Join(regions, ", "))
		}
		region = reg
	}
	key := value + accessKey + secretKey + region.Name
	buckets.RLock()
	bucket := buckets.buckets[key]
	buckets.RUnlock()
	if bucket == nil {
		auth := aws.Auth{
			AccessKey: accessKey,
			SecretKey: secretKey,
		}
		s := s3.New(auth, region)
		bucket = s.Bucket(value)
		if err := bucket.PutBucket(s3.Private); err != nil {
			return nil, err
		}
		buckets.Lock()
		if buckets.buckets == nil {
			buckets.buckets = make(map[string]*s3.Bucket)
		}
		buckets.buckets[key] = bucket
		buckets.Unlock()
	}
	return &s3Driver{
		bucket: bucket,
	}, nil
}

func init() {
	driver.Register("s3", s3Opener)
}
