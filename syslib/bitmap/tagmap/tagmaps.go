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

	"github.com/alphazero/gart/index" // XXX temp - content to move to index
	"github.com/alphazero/gart/syslib/bitmap"
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

/// dev mode stub for Wahl ////////////////////////////////////////////////////

// REVU how is this useful per original notion of using the mappedFile's []byte
// buffer as the Wahl's block array. (Copying it via Wahl.decode([]byte) is ok
// and we already have that!)
//
// But if aliasing via unsafe the mmf.buf:
//
// It works for Read but then wahl must be modified so that it can not change.
//
// If used for update, then bitmap size will change and it can not be pointing
// to the mappedFile's []byte array.
//
// It is possible to use mmaps to efficiently page through a very large bitmap
// index (e.g. map large multiple of 4KB pages) for Query ops, but then Wahl
// will need to be substantially changed.
//
// Let's be Reasonable and remember "Keep it simple!" dictum:
// - this prototype as of commit 9eb07df in branch gart-2.0-tagmap-proto has
//   some useful bits so switching to just reading the full file is OK.
//
// - if we want to fully use what we have as of 9eb07df, then -simply- call
//	 wahl.Deocde(mmf.buf[:headerSize]) and we're done with READ ops.
//
func MapWahl(buf []byte) (*bitmap.Wahl, error) {
	panic(errors.Fault("over thinking this :) see comment above ^^"))
}

/// tagmap file header /////////////////////////////////////////////////////////

const headerSize = 48
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

func (h *header) Print() {
	fmt.Printf("file type:   %016x\n", h.ftype)
	fmt.Printf("crc64:       %016x\n", h.crc64)
	fmt.Printf("created:     %016x (%s)\n", h.created, time.Unix(0, h.created))
	fmt.Printf("updated:     %016x (%s)\n", h.updated, time.Unix(0, h.updated))
	fmt.Printf("bitmap-size: %d\n", h.mapSize)
	fmt.Printf("bitmap-max:  %d\n", h.mapMax)
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
	// TODO created, updated can be also checked.

	if h.ftype != mmap_tagmap_ftype {
		errors.Bug("header.decode: invalid ftype: %x - expect: %x",
			h.ftype, mmap_tagmap_ftype)
	}
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
func LoadTagmap(tag string, opMode index.OpMode) (*bitmap.Wahl, error) {

	mmf, e := OpenMappedFile(TagmapFilename(tag), opMode)
	if e != nil {
		return nil, e
	}
	defer mmf.UnmapAndClose()

	fmt.Printf("debug - OpenTagmap: %q\n", tag)
	mmf.header.Print()
	fmt.Printf("% 02x\n", mmf)
	fmt.Printf("----------------------------------------\n")

	var wahl bitmap.Wahl
	if e := wahl.Decode(mmf.buf[headerSize:]); e != nil {
		return nil, e
	}

	return &wahl, nil
}

/// memory mapped file /////////////////////////////////////////////////////////

// REVU good idea to TODO move this to fs package and use same in object-index.
type mappedFile struct {
	*header
	filename string
	opMode   index.OpMode
	file     *os.File
	finfo    os.FileInfo // size is int64
	flags    int
	prot     int
	buf      []byte
	offset   int64
	modified bool
}

func OpenMappedFile(filename string, opMode index.OpMode) (*mappedFile, error) {
	if e := opMode.Verify(); e != nil {
		return nil, errors.ErrorWithCause(e, "OpenMappedFile")
	}

	var flags int
	var oflags int
	var prot int
	switch opMode {
	case index.Read:
		prot = syscall.MAP_PRIVATE
		prot = syscall.PROT_READ
		oflags = os.O_RDONLY
	case index.Write:
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
func (mmf *mappedFile) UnmapAndClose() error {
	if mmf.modified {
		panic(errors.Bug("REVU - UnmapAndClose called with modified set"))
	}
	if e := mmf.unmap(); e != nil {
		return e
	}
	if e := mmf.file.Close(); e != nil {
		return e
	}
	mmf.file = nil // TODO object-index mmap also needs to clear all this
	mmf.opMode = 0 // this should be enough if opMode.verify is used everywhere
	mmf.flags = 0
	mmf.prot = 0
	mmf.offset = 0
	mmf.modified = false
	return nil
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
	wahl, e := LoadTagmap(tag, index.Read)
	if e != nil {
		return e
	}

	wahl.Print(os.Stdout)

	return nil
}

func writeTagmap(tag string) error {
	return errors.NotImplemented("writeTagmap")
}

func queryTagmaps(tags ...string) error {
	return errors.NotImplemented("queryTagmaps")
}
