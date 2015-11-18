package leveldb

import (
	"crypto/sha1"
	"errors"
	"sync"

	"gnd.la/blobstore/chunk"
	"gnd.la/blobstore/chunk/fixed"
	"gnd.la/encoding/binary"
	"gnd.la/internal"

	"github.com/syndtr/goleveldb/leveldb"
)

const (
	chunkSize    = 256 * 1024    // 256 KiB
	maxBatchSize = 4 * (1 << 20) // 4MiB
)

var (
	littleEndian = binary.LittleEndian
	wfilesPool   sync.Pool
)

type wfile struct {
	drv       *leveldbDriver
	id        string
	chunks    [][]byte
	batch     *leveldb.Batch
	batchSize int
	metadata  []byte
	chunk.Chunker
}

func (f *wfile) WriteChunk(data []byte) error {
	h := sha1.Sum(data)
	hash := h[:]
	f.chunks = append(f.chunks, hash)
	if ch, err := f.drv.chunks.Get(hash, nil); err == nil {
		if len(ch) != len(data) {
			return errors.New("hash collision")
		}
		// Chunk already known. Ignore errors != nil here, since
		// the worst thing that could happen could be overwriting
		// an existing chunk with the same data. If there was an error
		// reading the db, we'll get an error when putting the data
		// a few lines later.
		return nil
	}
	// Not found, put it into the writing queue
	f.batch.Put(hash, data)
	f.batchSize += len(data)
	if f.batchSize >= maxBatchSize {
		return f.flushBatch()
	}
	return nil
}

func (f *wfile) flushBatch() error {
	err := f.drv.chunks.Write(f.batch, nil)
	f.batchSize = 0
	f.batch.Reset()
	return err
}

func (f *wfile) SetMetadata(b []byte) error {
	f.metadata = b
	return nil
}

func (f *wfile) Close() error {
	if rem := f.Chunker.Remaining(); len(rem) > 0 {
		if len(f.chunks) == 0 {
			// Store the file inline. Data is uint32 + len(metadata) + uint32 + rem
			total := 4 + len(f.metadata) + 4 + len(rem)
			data := make([]byte, total)
			out := putMetadata(data, f.metadata)
			// 0 chunks indicates the data is inline
			littleEndian.PutUint32(out, uint32(0))
			copy(out[4:], rem)
			id := f.id
			wfilesPool.Put(f)
			return f.drv.files.Put(internal.StringToBytes(id), data, nil)
		}
		if err := f.Chunker.Flush(); err != nil {
			return err
		}
	}
	if err := f.flushBatch(); err != nil {
		return err
	}
	// Reserve uint32 + len(metadata) + n sha1 hashes + n uint32 + 1 uint32 (for the chunk count)
	total := 4 + len(f.metadata) + (len(f.chunks) * (sha1.Size + 4)) + 4
	data := make([]byte, total)
	out := putMetadata(data, f.metadata)
	littleEndian.PutUint32(out, uint32(len(f.chunks)))
	pos := 4
	for _, chunk := range f.chunks {
		littleEndian.PutUint32(out[pos:], uint32(len(chunk)))
		pos += 4
		n := copy(out[pos:], chunk)
		pos += n
	}
	id := f.id
	wfilesPool.Put(f)
	return f.drv.files.Put(internal.StringToBytes(id), data, nil)
}

func newWFile(drv *leveldbDriver, id string) *wfile {
	if x := wfilesPool.Get(); x != nil {
		w := x.(*wfile)
		w.drv = drv
		w.id = id
		w.chunks = w.chunks[:0]
		w.metadata = nil
		w.Chunker.Reset()
		return w
	}
	w := &wfile{drv: drv, id: id, batch: new(leveldb.Batch)}
	w.Chunker = fixed.New(w, chunkSize)
	return w
}

func putMetadata(data []byte, metadata []byte) []byte {
	littleEndian.PutUint32(data, uint32(len(metadata)))
	n := copy(data[4:], metadata)
	return data[4+n:]
}
