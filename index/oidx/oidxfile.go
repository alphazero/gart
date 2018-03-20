// Doost!

package oidx

import (
	"fmt"
	"os"
	//	"path/filepath"
	//	"time"
	//	"unsafe"
	//	"github.com/alphazero/gart/digest"
	//	"github.com/alphazero/gart/fs"
	//	"github.com/alphazero/gart/lang/sort"
)

/// consts and vars ///////////////////////////////////////////////////////////

// header related consts
const (
	idx_file_code = 0x763f079cf73c668e // sha256("index-file")[:8]
	idxFilename   = "object.idx"       // REVU belongs to toplevle gart package
)

/* XXX deprecated
// 4K header, 32KB blocks
const (
	//	headerSize      = 4096 // file header is fs page sized
	blockSize       = 32768
	blockHeaderSize = 32    // TODO assert this
	blockDataSize   = 32736 // 1023 32B objects per block
	blockRecordSize = 32
	recordsPerBlock = 1023
)

// file consists of header and 0 or more blocks. blocks are multiples of fs
// pagesize. Each block is headed by 32B of meta-data, followed by sequence
// of object ids.
type block struct {
	crc64    uint64
	created  int64  // std unix nano
	updated  int64  // std unix nano
	rcnt     uint16 // [0, 1023]
	reserved [6]byte
	dat      [blockDataSize]byte
}

//type header struct {
//	ftype    uint64
//	crc64    uint64 // header crc
//	created  int64
//	updated  int64
//	bcnt     uint64 // block count
//	rcnt     uint64 // record count
//	reserved [4048]byte
//}

type pendingBlock struct {
	blk *block
	off int64
}

//const recordSize = digest.HashBytes // assert this on init

type idxfile struct {
	header
	file     *os.File
	filename string
	size     int64
	opMode   OpMode
	modified bool
	nextKey  uint64
	pending  *pendingBlock
}

// panics on zerolen input REVU index pkg should give it the full name
func Filename(home string) string {
	if home == "" {
		panic("bug - oidx.idxfilename: garthome is zerolen")
	}
	return filepath.Join(home, "index", "objects.idx")
}
*/
/// op mode ///////////////////////////////////////////////////////////////////

type OpMode byte

const (
	Read OpMode = 1 << iota
	Write
	Verify
	Compact
)

// panics on unimplemented op mode
func (m OpMode) fopenFlag() int {
	switch m {
	case Read:
		return os.O_RDONLY
	case Write:
		return os.O_RDWR
	case Verify:
	case Compact:
	default:
	}
	panic(fmt.Errorf("bug - oidx.OpMode: not implemented - mode  %d", m))
}

// panics on invalid opMode
func (m OpMode) verify() error {
	switch m {
	case Read:
	case Write:
	case Verify:
	case Compact:
	default:
		return fmt.Errorf("bug - oidx.OpMode: unknown mode - %d", m)
	}
	return nil
}

// Returns string rep. of opMode
func (m OpMode) String() string {
	switch m {
	case Read:
		return "Read"
	case Write:
		return "Write"
	case Verify:
		return "Verify"
	case Compact:
		return "Compact"
	}
	panic(fmt.Errorf("bug - oidx.OpMode: unknown mode - %d", m))
}

/* XXX deprecated
// Creates file, writes initial header and closes file.
func CreateIndex(home string) error {
	// TODO just modify below
	panic("oidx.CreateIndex: not implemented")
	var filename = Filename(home)

	file, e := fs.OpenNewFile(filename, os.O_WRONLY|os.O_APPEND)
	if e != nil {
		return fmt.Errorf("oidx.CreateIndex: %s", e)
	}
	defer file.Close()

	// just write the header
	var now = time.Now().UnixNano()
	var hdr = header{
		ftype:   idx_file_code,
		created: now,
		updated: now,
		bcnt:    0,
		rcnt:    0,
	}

	var buf [headerSize]byte
	hdr.encode(buf[:])

	_, e = file.Write(buf[:])
	if e != nil {
		return fmt.Errorf("oidx.CreateIndex: %s", e)
	}

	fmt.Printf("debug - oidx.CreateIndex:\n%v\n", hdr)
	return nil
}
*/

/* XXX deprecated
func (h *header) encode(buf []byte) {
	*(*uint64)(unsafe.Pointer(&buf[0])) = h.ftype
	*(*int64)(unsafe.Pointer(&buf[16])) = h.created
	*(*int64)(unsafe.Pointer(&buf[24])) = h.updated
	*(*uint64)(unsafe.Pointer(&buf[32])) = h.bcnt
	*(*uint64)(unsafe.Pointer(&buf[40])) = h.rcnt

	h.crc64 = digest.Checksum64(buf[16:])
	*(*uint64)(unsafe.Pointer(&buf[8])) = h.crc64

	return
}

func (idx *idxfile) readAndVerifyHeader() error {

	_, e := idx.file.Seek(0, os.SEEK_SET)
	if e != nil {
		return e
	}

	var buf = make([]byte, headerSize)
	var n int
	for n < len(buf) {
		n0, e := idx.file.Read(buf[n:])
		if e != nil {
			return fmt.Errorf("idxfile.readAndVerifyHeader: Read - n: %d - %s", n, e)
		}
		n += n0
	}

	(*idx).header = *(*header)(unsafe.Pointer(&buf[0]))

	crc64 := digest.Checksum64(buf[16:])
	if idx.crc64 != crc64 {
		return fmt.Errorf("idxfile.readAndVerifyHeader: crc:%d computed:%d", idx.crc64, crc64)
	}
	return nil
}
*/

// TODO redo in mmap.go
/* XXX deprecated
// Opens the object index file. REVU mode?
func OpenIndex(home string, opMode OpMode) (*idxfile, error) {

	panic("oidx.OpenIndex: not implemented")
	/// open ////////////////////////////////////////////////////////

	if e := opMode.verify(); e != nil {
		return nil, e
	}

	// open file and get stat
	var filename = Filename(home)
	var flag = opMode.fopenFlag()
	file, e := os.OpenFile(filename, flag, fs.FilePerm)
	if e != nil {
		return nil, fmt.Errorf("oidx.OpenIndex: %s", e)
	}
	finfo, e := file.Stat()
	if e != nil {
		return nil, fmt.Errorf("oidx.OpenIndex: unexpected: %s", e)
	}
	fmt.Printf("debug - idxfile.OpenIndex - open flag: %d - filename: %q\n", flag, filename)

	/// initialize idxfile //////////////////////////////////////////

	idx := &idxfile{
		file:     file,
		filename: filename,
		size:     finfo.Size(),
		opMode:   opMode,
		modified: false,
		nextKey:  0,
		pending:  nil,
	}

	if e := idx.readAndVerifyHeader(); e != nil {
		idx.file.Close()
		return nil, e
	}
	idx.nextKey = idx.rcnt + 1

	var oe error
	switch opMode {
	case Read:
		oe = idx.initModeRead()
	case Write:
		oe = idx.initModeWrite()
	default:
		return nil, fmt.Errorf("")
	}
	if oe != nil {
		idx.file.Close()
		return nil, fmt.Errorf("oidx.OpenIndex: open for opMode:%s - %s", opMode, e)
	}

	// REVU TODO determine if we last block is partial or not
	if idx.rcnt%recordsPerBlock != 0 {
		fmt.Printf("debug - has partial block\n")
	}

	return idx, nil
}
*/

/* XXX deprecated
func (idx *idxfile) initModeRead() error {
	fmt.Printf("debug - idxfile\n%v\n", idx)
	// NOTE mmap would be appropriate here.
	//      Read mode is to support idxfile.Lookup(key...uint64)
	//		which is a scan function.
	panic("idxfile.initModeRead: not implemented")
}

func (idx *idxfile) initModeWrite() error {
	fmt.Printf("debug - idxfile\n%v\n", idx)
	panic("idxfile.initModeWrite: not implemented")
}

// Register adds an entry for the object content hash 'oid'.
// REVU Note that this function (nor this index type) checks for
// duplicates.
func (idx *idxfile) Register(oid []byte) (uint64, error) {
	if oid == nil || len(oid) != recordSize {
		return 0, ErrInvalidOid
	}
	if idx.opMode != Write {
		return 0, ErrInvalidOp
	}

	panic("oidx.Register: not implemented")
}

// Returns the oid (bytes) for the key set. Note that keys are
// sorted ascending to optimize reads and the returned []byte array
// order corresponds to the sorted order of keys. As 'key' is purely
// an internal concern this is of no concern to top-level funcs using
// Lookup(). There the [][]byte will be converted to []OID and identified
// index.Cards will have full information, including 'key' should be required.
func (idx *idxfile) Lookup(key ...uint64) ([][]byte, error) {
	// REVU will never query in Write mode?
	if idx.opMode != Read {
		return nil, ErrInvalidOp
	}

	if key == nil {
		return nil, ErrNilArg
	}

	var klen = len(key)
	if klen == 0 {
		return [][]byte{}, nil // nop
	}

	sort.Uint64(key)
	if key[klen-1] >= idx.nextKey {
		return nil, ErrInvalidKeyRange
	}

	// REVU certainly Read ops can (should) be mmap
	//	_, e := idx.file.Seek(0, os.SEEK_SET)
	//	if e != nil {
	//		return nil, e
	//	}

	panic("oidx.Lookup: not implemented")
}

func (idx *idxfile) Sync() (bool, error) {
	if idx.opMode != Write {
		return false, ErrInvalidOp
	}
	panic("oidx.Sync: not implemented")
}

func (idx *idxfile) Close() error {
	if idx.file == nil {
		return ErrIndexIsClosed
	}
	if idx.modified {
		return ErrPendingChanges
	}

	// if we error out here, then we have a either an os fault or
	// or a bug. Either way, the idxfile object is out of commission.
	e := idx.file.Close()
	idx.file = nil

	return e
}
*/
