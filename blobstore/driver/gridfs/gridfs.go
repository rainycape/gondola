package gridfs

import (
	"fmt"
	"gnd.la/blobstore/driver"
	"gnd.la/config"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"sync"
)

// mgo recommends Copy'ing the initial
// session instead of calling Dial
// everytime
var connections struct {
	sync.RWMutex
	sessions map[string]*mgo.Session
}

// gridfs Seek is broken for writing files, so
// hide that method by wrapping it into a struct
type wfile struct {
	file *mgo.GridFile
}

func (w *wfile) Write(p []byte) (int, error) {
	return w.file.Write(p)
}

func (w *wfile) Close() error {
	return w.file.Close()
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
	return &wfile{
		file: f,
	}, nil
}

func (d *gridfsDriver) Open(id string) (driver.RFile, error) {
	return d.fs.OpenId(bson.ObjectIdHex(id))
}

func (d *gridfsDriver) Remove(id string) error {
	return d.fs.RemoveId(bson.ObjectIdHex(id))
}

func (d *gridfsDriver) Close() error {
	d.session.Close()
	return nil
}

func gridfsOpener(value string, o config.Options) (driver.Driver, error) {
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
	prefix := o.Get("prefix")
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
