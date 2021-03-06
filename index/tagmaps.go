// Doost!

package index

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/alphazero/gart/repo"
	"github.com/alphazero/gart/syslib/bitmap"
	"github.com/alphazero/gart/syslib/debug"
	"github.com/alphazero/gart/syslib/digest"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/syslib/fs"
)

var ErrTagNotExist = errors.Error("Tag does not exit")

/// tagmap file header /////////////////////////////////////////////////////////

const tagmapHeaderSize = 48
const mmap_tagmap_ftype uint64 = 0x5807263e43839459

// tagmap header is the minimal content of a valid gart tagmap file.
type tagmapHeader struct {
	ftype   uint64
	crc64   uint64
	created int64  // unix nanos
	updated int64  // unix nanos
	mapSize uint64 // bytes - number of blocks * 4
	mapMax  uint64 // max bitnum in bitmap
}

func (h *tagmapHeader) Print(w io.Writer) {
	fmt.Fprintf(w, "file type:   %016x\n", h.ftype)
	fmt.Fprintf(w, "crc64:       %016x\n", h.crc64)
	fmt.Fprintf(w, "created:     %016x (%s)\n", h.created, time.Unix(0, h.created))
	fmt.Fprintf(w, "updated:     %016x (%s)\n", h.updated, time.Unix(0, h.updated))
	fmt.Fprintf(w, "bitmap-size: %d\n", h.mapSize)
	fmt.Fprintf(w, "bitmap-max:  %d\n", h.mapMax)
}

// encode writes the header data to the given buffer.
// Returns error if buf length < tagmap.tagmapHeaderSize.
func (h *tagmapHeader) encode(buf []byte) error {
	if len(buf) < tagmapHeaderSize {
		return errors.Error("tagmapHeader.encode: insufficient buffer length: %d", len(buf))
	}
	*(*uint64)(unsafe.Pointer(&buf[0])) = h.ftype
	*(*int64)(unsafe.Pointer(&buf[16])) = h.created
	*(*int64)(unsafe.Pointer(&buf[24])) = h.updated
	// skip crc64
	*(*uint64)(unsafe.Pointer(&buf[32])) = h.mapSize
	*(*uint64)(unsafe.Pointer(&buf[40])) = h.mapMax

	h.crc64 = digest.Checksum64(buf[16:tagmapHeaderSize])
	*(*uint64)(unsafe.Pointer(&buf[8])) = h.crc64
	return nil
}

// encode writes the header structure to the buffer provided. Checksum is for header
// data only.
func (h *tagmapHeader) decode(buf []byte) error {
	var err = errors.For("tagmapHeader.decode")
	if len(buf) < tagmapHeaderSize {
		return err.InvalidArg("len(buf):%d < %d", len(buf), tagmapHeaderSize)
	}
	*h = *(*tagmapHeader)(unsafe.Pointer(&buf[0]))

	/// verify //////////////////////////////////////////////////////

	if h.ftype != mmap_tagmap_ftype {
		return err.Bug("ftype:%x - expect: %x", h.ftype, mmap_tagmap_ftype)
	}
	crc64 := digest.Checksum64(buf[16:tagmapHeaderSize])
	if crc64 != h.crc64 {
		return err.Bug("checksum:%d - expect: %d", h.crc64, crc64)
	}
	if h.created == 0 {
		return err.Bug("created:%d", h.created)
	}
	if h.updated < h.created {
		return err.Bug("updated: %d < created:%d", h.updated, h.created)
	}

	return nil
}

/// tagmap file ////////////////////////////////////////////////////////////////

// Tagmap encapsulates tagmap file metadata (in header) and the in-mem bitmap
// of the tagmap. It also provides functions to update, save, and print the
// tagmap.
type Tagmap struct {
	header   *tagmapHeader
	tag      string
	bitmap   *bitmap.Wahl
	source   string
	modified bool
}

// multi-line print function suitable for debugging, prints both the header
// and bitmap details.
func (t *Tagmap) Print(w io.Writer) {
	fmt.Fprintf(w, "-- Tagmap (%q)\n", t.tag)
	t.header.Print(w)
	fmt.Fprintf(w, "source:     %q\n", t.source)
	fmt.Fprintf(w, "modified:   %t\n", t.modified)
	fmt.Fprintf(w, "-- bitmap --\n")
	t.bitmap.Print(w)
}

// Returns the absolute path filename for the given tag. The full path is a
// variation on git's approach to blob file paths: tag name is converted to
// lower-case form; (b) the blake2b uint64 hash of that is used to construct
// the path fragment /xx/xxxxxxxxxxxx. We only use 8 bytes since that is more
// than sufficient given that the total number of tags in gart will be far less
// than 2^32.
func TagmapFilename(tag string) string {
	tag = strings.ToLower(tag)
	hash := fmt.Sprintf("%x.bitmap", digest.SumUint64([]byte(tag)))
	path := filepath.Join(repo.IndexTagmapsPath, hash[:2])
	return filepath.Join(path, hash[2:])
}

// Creates the initial tagmap file for the given tag in the canonical
// repo location. Tag names in gart are case-insensitive and the tag
// (name) will always be converted to lower-case form.
func createTagmap(tag string) (*Tagmap, error) {
	var err = errors.For(fmt.Sprintf("index.createTagmap(%q)", tag))
	var tagmap = &Tagmap{}

	filename := TagmapFilename(tag)

	// if dir structure does not exist, create it.
	dir := filepath.Dir(filename)
	if e := os.MkdirAll(dir, repo.DirPerm); e != nil {
		return nil, err.ErrorWithCause(e, "dir:%q", dir)
	}

	file, e := fs.OpenNewFile(filename, os.O_WRONLY|os.O_APPEND)
	if e != nil {
		return nil, e
	}
	defer file.Close()

	var now = time.Now().UnixNano()
	var h = &tagmapHeader{
		ftype:   mmap_tagmap_ftype,
		created: now,
		updated: now,
		mapSize: 0,
		mapMax:  0,
	}

	var buf [tagmapHeaderSize]byte
	if e := h.encode(buf[:]); e != nil {
		return nil, e
	}

	_, e = file.Write(buf[:])
	if e != nil {
		return nil, err.ErrorWithCause(e, "on file.Write")
	}

	tagmap = &Tagmap{
		header: h,
		tag:    tag,
		bitmap: bitmap.NewWahl(),
		source: filename,
	}

	return tagmap, nil
}

// Loads the tagmap (in form of bitmap.Wahl) from file and closes the file.
// File is openned in private, read-only mode.
func loadTagmap(tag string, create bool) (*Tagmap, error) {

	var err = errors.For(fmt.Sprintf("index.loadTagmap(%q)", tag))
	var debug = debug.For("index.loadTagmap")
	debug.Printf("tag: %q create: %t", tag, create)

	/// open file ///////////////////////////////////////////////////

	filename := TagmapFilename(tag)
	file, e := os.OpenFile(filename, os.O_RDONLY, repo.FilePerm)
	if e != nil {
		if os.IsNotExist(e) {
			if !create {
				return nil, ErrTagNotExist
			}
			tagmap, e := createTagmap(tag)
			if e != nil {
				return nil, err.ErrorWithCause(e, "on createTagmap")
			}
			return tagmap, nil
		}
		return nil, e
	}
	defer file.Close()

	finfo, e := file.Stat()
	if e != nil {
		return nil, e
	}

	/// mmap it /////////////////////////////////////////////////////

	var fd = int(file.Fd())
	var fsize = finfo.Size()
	var prot = syscall.PROT_READ
	var flags = syscall.MAP_PRIVATE
	buf, e := syscall.Mmap(fd, 0, int(fsize), prot, flags)
	if e != nil {
		return nil, e
	}
	defer syscall.Munmap(buf)

	/// decode content //////////////////////////////////////////////

	// decode verifies header
	var header tagmapHeader
	if e := header.decode(buf); e != nil {
		return nil, err.ErrorWithCause(e, "hdr.decode")
	}

	var wahl bitmap.Wahl
	if e := wahl.Decode(buf[tagmapHeaderSize:]); e != nil {
		return nil, err.ErrorWithCause(e, "Wahl.Decode")
	}

	// verify: compare header & actual bitmap
	bug := func(what string, have, expect uint64) error {
		return err.Bug("%s verify - wahl:%d header:%d", what, have, expect)
	}
	wahlSize := uint64(wahl.Size())
	if header.mapSize != wahlSize {
		return nil, bug("mapSize", wahlSize, header.mapSize)
	}
	wahlMax := uint64(wahl.Max())
	if header.mapMax != wahlMax {
		return nil, bug("mapMax", wahlMax, header.mapMax)
	}

	// Good to go.
	var tagmap = &Tagmap{
		header: &header,
		tag:    tag,
		bitmap: &wahl,
		source: filename,
	}

	return tagmap, nil
}

type bitmapOp byte

const (
	clearBits bitmapOp = 1 << iota
	setBits
)

// update sets or clears the bits for the keys per the specified bitmapOp.
// Update does not compress the bitmap.
//
// Returns true if bitmap changed.
// panics with a Bug if Tagmap bitmap is nil
func (t *Tagmap) update(op bitmapOp, keys ...uint) bool {
	if t.bitmap == nil {
		panic(errors.Bug("Tagmap.update: bitmap is nil"))
	}

	var ok bool
	switch op {
	case clearBits:
		ok = t.bitmap.Clear(keys...)
	case setBits:
		ok = t.bitmap.Set(keys...)
	}

	if ok {
		t.header.mapMax = uint64(t.bitmap.Max())
		t.header.mapSize = uint64(t.bitmap.Size())
		t.modified = true
		return true
	}

	return false // REVU don't return t.modified - it could have been set before
}

// Tagmap#save saves the tagmap if modified. If modified, the Wahl bitmap
// is compressed; the tagmap is saved to a swap file; and finally the swapfile
// is swapped with the original source file.
//
// Function returns a bool indicating if IO was performed, and, errors if
// any. If error is not nil, the bool result should be ignored as a swap file
// is used.
func (t *Tagmap) save() (bool, error) {
	var err = errors.For("Tagmap.save")

	if !t.modified {
		return false, nil
	}

	// compress bitmap - this may change bitmap Max bit and map size.
	t.bitmap.Compress()

	// update header
	t.header.updated = time.Now().UnixNano()
	t.header.mapSize = uint64(t.bitmap.Size())
	t.header.mapMax = uint64(t.bitmap.Max())

	/// swapfile ////////////////////////////////////////////////////

	var size = int64(tagmapHeaderSize + t.header.mapSize)
	var swapfile = fs.SwapfileName(t.source)
	var ops = os.O_RDWR //| os.O_APPEND
	sfile, e := fs.OpenNewFile(swapfile, ops)
	if e != nil {
		return false, err.ErrorWithCause(e, "swapfile %q open-new", swapfile)
	}
	if e := sfile.Truncate(size); e != nil {
		return false, err.ErrorWithCause(e, "swapfile trunacte")
	}
	if xoff, e := sfile.Seek(0, os.SEEK_SET); e != nil || xoff != 0 {
		return false, err.ErrorWithCause(e, "swapfile seek head - xoff:%d", xoff)
	}

	// mmap it

	var fd = int(sfile.Fd())
	buf, e := syscall.Mmap(fd, 0, int(size), syscall.PROT_WRITE, syscall.MAP_SHARED)
	if e != nil {
		return false, err.ErrorWithCause(e, "swapfile mmap")
	}

	// encode buffer and unmap and close swapfile

	if e := t.header.encode(buf[:tagmapHeaderSize]); e != nil {
		return false, err.ErrorWithCause(e, "header.encode")
	}
	if e := t.bitmap.Encode(buf[tagmapHeaderSize:]); e != nil {
		return false, err.ErrorWithCause(e, "bitmap.encode")
	}

	if e := syscall.Munmap(buf); e != nil {
		return false, err.ErrorWithCause(e, "unmap")
	}
	if e := sfile.Close(); e != nil {
		return false, err.ErrorWithCause(e, "swapfile %q close", swapfile)
	}

	// swap it
	if e := os.Rename(swapfile, t.source); e != nil {
		return false, err.ErrorWithCause(e, "os.Replace %q %q", swapfile, t.source)
	}

	return true, nil // : π U
}
