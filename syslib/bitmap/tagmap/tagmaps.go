// Doost!

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/alphazero/gart/syslib/digest"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/syslib/fs"
	"github.com/alphazero/gart/system"
)

// to try:
// using system fixtures
// - create a Wahl tagmap
// - open Wahl tagmap in OpUpdate mode
//	- save using mmap
//		- multiple Sets then Compress then Sync
//
// - load Wahl in OpQuery mode
//		- query 'keys' that are set
//
// - tagmap manager
//		- AND 2 or more tagmaps and get 'keyset'

/// op mode ///////////////////////////////////////////////////////////////////

// REVU this is exactly the same thing as op mode for index.
// can't be in gart/index as index package will include the object and
// tagmap sub-packages.  TODO think about this ..
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
	panic(errors.NotImplemented("tagmap.OpMode: not implemented - mode  %d", m))
}

// panics on invalid opMode
func (m OpMode) verify() error {
	switch m {
	case Read:
	case Write:
	case Verify:
	case Compact:
	default:
		return errors.Bug("tagmap.OpMode: unknown mode - %d", m)
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
	panic(errors.Bug("tagmap.OpMode: unknown mode - %d", m))
}

/// tagmap file header /////////////////////////////////////////////////////////

const headerSize = 32
const mmap_tagmap_ftype = 0x5807263e43839459

// tagmap header is the minimal content of a valid gart tagmap file.
type header struct {
	ftype   uint64
	crc64   uint64
	created int64  // unix nanos
	updated int64  // unix nanos
	mapSize uint64 // bytes - number of blocks * 4
	mapMax  uint64 // max bitnum in bitmap
}

// encode writes the header data to the given buffer.
// Returns error if buf length < tagmap.headerSize.
func (h *header) encode(buf []byte) error {
	if len(buf) < headerSize {
		return errors.Error("header.encode: insufficient buffer length: %d", len(buf))
	}
	*(*uint64)(unsafe.Pointer(&buf[0])) = h.ftype
	*(*int64)(unsafe.Pointer(&buf[16])) = h.created
	*(*int64)(unsafe.Pointer(&buf[24])) = h.updated

	h.crc64 = digest.Checksum64(buf[16:])
	*(*uint64)(unsafe.Pointer(&buf[8])) = h.crc64
	return nil
}
func (h *header) decode(buf []byte) error {
	if len(buf) < headerSize {
		return errors.Error("header.decode: insufficient buffer length: %d", len(buf))
	}
	*h = *(*header)(unsafe.Pointer(&buf[0]))

	// verify
	// ftype
	if h.ftype != mmap_tagmap_ftype {
		errors.Bug("header.decode: invalid ftype: %x - expect: %x",
			h.ftype, mmap_tagmap_ftype)
	}
	// TODO created, updated can be also checked.
	// crc
	crc64 := digest.Checksum64(buf[16:])
	if crc64 != h.crc64 {
		errors.Bug("header.decode: invalid checksum: %d - expect: %d",
			h.crc64, crc64)
	}

	return nil
}

/// tagmap file ////////////////////////////////////////////////////////////////

// Returns the absolute path filename for the given tag. The full path is a
// variation on git's approach to blob file paths: tag name is converted to
// lower-case form; (b) the blake2b uint64 hash of that is used to construct
// the path fragment /xx/xxxxxxxxxxxx. We only use 8 bytes since that is more
// than sufficient given that the total number of tags in gart will be far less
// than 2^32.
func TagmapFilename(tag string) string {
	tag = strings.ToLower(tag)
	hash := fmt.Sprintf("%x.bitmap", digest.SumUint64([]byte(tag)))
	path := filepath.Join(system.IndexTagmapsPath, hash[:2])
	return filepath.Join(path, hash[2:])
}

// Creates the initial tagmap file for the given tag in the canonical
// repo location. Tag names in gart are case-insensitive and the tag
// (name) will always be converted to lower-case form.
func CreateTagmap(tag string) error {

	filename := TagmapFilename(tag)

	// if dir structure does not exist, create it.
	dir := filepath.Dir(filename)
	if e := os.MkdirAll(dir, system.DirPerm); e != nil {
		return errors.ErrorWithCause(e, "CreateTagmap: dir: %q", dir)
	}

	file, e := fs.OpenNewFile(filename, os.O_WRONLY|os.O_APPEND)
	if e != nil {
		return errors.ErrorWithCause(e, "CreateTagmap: tag: %q", tag)
	}
	defer file.Close()

	var now = time.Now().UnixNano()
	var h = &header{
		ftype:   mmap_tagmap_ftype,
		created: now,
		updated: now,
	}

	var buf [headerSize]byte
	if e := h.encode(buf[:]); e != nil {
		return e
	}

	_, e = file.Write(buf[:])
	if e != nil {
		return errors.ErrorWithCause(e, "createTagmap: tag: %s", tag)
	}
	return nil
}

// tagmap use-cases:
//
//	- Query: given one or more tags, get all object keys. N Reads.
//	- Read: [dev] simply get the wahl bitmap for a given tag. 1 Read.
//  - Update: set/clear bit(s) for a given tag. N updates.
//	  this is a swap file mode. REVU the entire point of this adhoc exercise
//	  was to find out what is the best way to update the tagmap files!
//  - Compact: See update.
//
// REVU both tagmap and object-index (mmap) share the same OpMode and open
// 		sequence. TODO think about consolidation (later).
//
func OpenTagmap(tag string, opMode OpMode) (*mappedFile, error) {

	return OpenMappedFile(TagmapFilename(tag), opMode)
}

/// memory mapped file /////////////////////////////////////////////////////////

// REVU good idea to TODO move this to fs package and use same in object-index.
type mappedFile struct {
	*header
	filename string
	opMode   OpMode
	file     *os.File
	finfo    os.FileInfo // size is int64
	flags    int
	prot     int
	buf      []byte
	offset   int64
	modified bool
}

func OpenMappedFile(filename string, opMode OpMode) (*mappedFile, error) {
	if e := opMode.verify(); e != nil {
		return nil, errors.ErrorWithCause(e, "OpenMappedFile")
	}

	var flags int
	var oflags int
	var prot int
	switch opMode {
	case Read:
		prot = syscall.MAP_PRIVATE
		prot = syscall.PROT_READ
		oflags = os.O_RDONLY
	case Write:
		flags = syscall.MAP_SHARED
		prot = syscall.PROT_WRITE
		oflags = os.O_RDWR
	default:
		panic(errors.Bug("OpenMappedFile: unsupported opMode: %s", opMode))
	}

	file, e := os.OpenFile(filename, oflags, system.FilePerm)
	if e != nil {
		return nil, e
	}
	// Note:close file on all errors after this point

	finfo, e := file.Stat()
	if e != nil {
		file.Close()
		return nil, e
	}

	var h header
	mmf := &mappedFile{
		header:   &h,
		filename: filename,
		opMode:   opMode,
		file:     file,
		finfo:    finfo,
		prot:     prot,
		flags:    flags,
	}

	var offset int64 = 0
	if e := mmf.mmap(offset, int(finfo.Size()), false); e != nil {
		file.Close()
		return nil, e
	}

	return mmf, nil
}

// unmapAndClose first updates header timestamp and (re)writes
// the header bytes (if opMode is Read & modified). If so, the
// bool retval will reflect that.
func (mmf *mappedFile) UnmapAndClose() (bool, error) {
	// NOTE the write of header is here is very important
	// REVU this should be done in AddObject (which is the only
	// place that actually modifies the header! It does not belong here.
	// TODO first try it in ref/mmap.go
	fmt.Printf("debug - mappedFile.unamp -- IN\n")
	var updated bool
	if mmf.modified {
		mmf.header.updated = time.Now().UnixNano()
		mmf.header.encode(mmf.buf)
		updated = true
	}
	if e := mmf.unmap(); e != nil {
		return updated, e
	}
	if e := mmf.file.Close(); e != nil {
		return updated, e
	}
	mmf.file = nil
	return updated, nil
}

// if mmf.buf != nil, function first unmaps and then maps again at given
// offset and for buf length specfieid -- effectively remap. Otherwise it
// just a mapping.
func (mmf *mappedFile) mmap(offset int64, length int, remap bool) error {
	if mmf.buf != nil {
		if !remap {
			return errors.Bug("mappedFile.mmap: []buf not nil - remap: %t", remap)
		}
		if e := mmf.unmap(); e != nil {
			return e
		}
	}
	var fd = int(mmf.file.Fd())
	buf, e := syscall.Mmap(fd, offset, length, mmf.prot, mmf.flags)
	if e != nil {
		return e
	}
	mmf.buf = buf
	mmf.offset = offset
	if !remap {
		if e := mmf.header.decode(mmf.buf); e != nil {
			if e0 := mmf.unmap(); e0 != nil {
				return errors.Error("mappedFile.mmap: deocde-err:%s - unmap-err:%s", e, e0)
			}
			return e
		}
	}

	return nil
}

func (mmf *mappedFile) unmap() error {
	fmt.Printf("debug - mappedFile.unamp -- IN")
	if mmf.buf == nil {
		return errors.Bug("mappedFile.unmap: buf is nil")
	}
	if e := syscall.Munmap(mmf.buf); e != nil {
		return errors.FaultWithCause(e, "mappedFile.unmap: %q", mmf.filename)
	}
	mmf.buf = nil
	mmf.offset = 0
	return nil
}

/// adhoc test /////////////////////////////////////////////////////////////////

var option = struct {
	op   string
	tags string
}{}
var ops = []string{"c", "r", "w", "q"}

func init() {
	flag.StringVar(&option.op, "op", option.op, "c:create, r:read, w:write, q:query")
	flag.StringVar(&option.tags, "tags", option.tags, "csv list in \" \"s ")
}

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "exit on error: %v\n", e)
	os.Exit(1)
}

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")

	// parse flags and verify option
	flag.Parse()

	if option.op == "" {
		exitOnError(errors.Usage("flag -op must be specified"))
	}
	option.op = strings.ToLower(option.op)
	for _, op := range ops {
		if op == option.op {
			goto op_verified
		}
	}
	exitOnError(errors.Usage("invalid op:%q", option.op))

op_verified:
	option.tags = strings.TrimSuffix(option.tags, ",")
	var tagnames = strings.Split(option.tags, ",")
	for i, s := range tagnames {
		tag := strings.Trim(s, " ")
		if tag == "" {
			exitOnError(errors.Usage("option -tags has zero-len tagname: %q", tagnames))
		}
		tagnames[i] = tag
	}
	if len(tagnames) == 0 {
		exitOnError(errors.Usage("option -tags must be non-empty"))
	}

	var e error
	switch option.op {
	case "c":
		e = createTagmaps(tagnames...)
	case "r":
		e = readTagmap(tagnames[0])
	case "w":
		e = writeTagmap(tagnames[0])
	case "q":
		e = queryTagmaps(tagnames...)
	default:
		exitOnError(errors.Bug("verified op is not known: %q", option.op))
	}

	if e != nil {
		exitOnError(errors.Bug("op: %q -  %v", option.op, e))
	}

	os.Exit(0)
}

func createTagmaps(tags ...string) error {
	for _, tag := range tags {
		if e := CreateTagmap(tag); e != nil {
			return e
		}
	}
	return nil
}

func readTagmap(tag string) error {
	mmf, e := OpenTagmap(tag, Read)
	if e != nil {
		return e
	}
	fmt.Printf("% 02x\n", mmf)
	updated, e := mmf.UnmapAndClose()
	if e != nil {
		return e
	}
	if updated {
		return errors.Bug("UnmapAndClose() returned updated true")
	}
	return nil
}

func writeTagmap(tag string) error {
	return errors.NotImplemented("writeTagmap")
}

func queryTagmaps(tags ...string) error {
	return errors.NotImplemented("queryTagmaps")
}
