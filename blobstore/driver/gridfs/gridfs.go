// Package gridfs implements a GridFS driver for the blobstore.
//
// GridFS is a file storage system built on top of mongodb. For more
// information about mongodb visit http://www.mongodb.com.
//
// The URL for this driver must take the form gridfs://host/database[#prefix={prefix}].
// If prefix is not provided, "fs" is used.
package gridfs

import (
	"fmt"
	"gnd.la/blobstore/driver"
	"gnd.la/config"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"sync"
)

const (
	metadataKey = "meta"
)

// mgo recommends Copy'ing the initial
// session instead of calling Dial
// everytime
var connections struct {
	sync.RWMutex
	sessions map[string]*mgo.Session
}

type rfile mgo.GridFile

func (r *rfile) Seek(offset int64, whence int) (int64, error) {
	return (*mgo.GridFile)(r).Seek(offset, whence)
}

func (r *rfile) Read(p []byte) (int, error) {
	return (*mgo.GridFile)(r).Read(p)
}

func (r *rfile) Close() error {
	return (*mgo.GridFile)(r).Close()
}

func (r *rfile) Metadata() ([]byte, error) {
	out := make(map[string][]byte)
	if err := (*mgo.GridFile)(r).GetMeta(&out); err != nil {
		return nil, err
	}
	return out[metadataKey], nil
}

// gridfs Seek is broken for writing files, so
// hide that method by wrapping it into a struct
type wfile mgo.GridFile

func (w *wfile) SetMetadata(b []byte) error {
	(*mgo.GridFile)(w).SetMeta(bson.M{metadataKey: b})
	return nil
}

func (w *wfile) Write(p []byte) (int, error) {
	return (*mgo.GridFile)(w).Write(p)
}

func (w *wfile) Close() error {
	return (*mgo.GridFile)(w).Close()
}

type gridfsDriver struct {
	fs      *mgo.GridFS
	session *mgo.Session
}

func (d *gridfsDriver) Create(id string) (driver.WFile, error) {
	f, err := d.fs.Create("")
	if err != nil {
		return nil, err
	}
	f.SetId(bson.ObjectIdHex(id))
	return (*wfile)(f), nil
}

func (d *gridfsDriver) Open(id string) (driver.RFile, error) {
	r, err := d.fs.OpenId(bson.ObjectIdHex(id))
	return (*rfile)(r), err
}

func (d *gridfsDriver) Remove(id string) error {
	return d.fs.RemoveId(bson.ObjectIdHex(id))
}

func (d *gridfsDriver) Close() error {
	d.session.Close()
	return nil
}

func gridfsOpener(url *config.URL) (driver.Driver, error) {
	value := url.Value
	connections.RLock()
	session := connections.sessions[value]
	connections.RUnlock()
	if session == nil {
		var err error
		session, err = mgo.Dial(value)
		if err != nil {
			return nil, fmt.Errorf("error connecting to mongodb: %s", err)
		}
		// Check if a database was provided
		db := session.DB("")
		if db.Name == "" || db.Name == "test" {
			session.Close()
			return nil, fmt.Errorf("invalid gridfs url, it does not specify a database. Please, add a path to the url e.g. gridfs://localhost/database")
		}
		connections.Lock()
		if connections.sessions == nil {
			connections.sessions = make(map[string]*mgo.Session)
		}
		connections.sessions[value] = session
		connections.Unlock()
	}
	scopy := session.Copy()
	prefix := url.Fragment.Get("prefix")
	if prefix == "" {
		prefix = "fs"
	}
	return &gridfsDriver{
		session: scopy,
		fs:      scopy.DB("").GridFS(prefix),
	}, nil
}

func init() {
	driver.Register("gridfs", gridfsOpener)
}
