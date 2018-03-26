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

	"github.com/alphazero/gart/syslib/bitmap"
	"github.com/alphazero/gart/syslib/digest"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/syslib/fs"
	"github.com/alphazero/gart/system"
)

/// tagmap file header /////////////////////////////////////////////////////////

const tagmapHeaderSize = 48
const mmap_tagmap_ftype = 0x5807263e43839459

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

	h.crc64 = digest.Checksum64(buf[16:])
	*(*uint64)(unsafe.Pointer(&buf[8])) = h.crc64
	return nil
}

func (h *tagmapHeader) decode(buf []byte) error {
	if len(buf) < tagmapHeaderSize {
		return errors.Error("tagmapHeader.decode: insufficient buffer length: %d", len(buf))
	}
	*h = *(*tagmapHeader)(unsafe.Pointer(&buf[0]))

	// verify
	// TODO created, updated can be also checked.

	if h.ftype != mmap_tagmap_ftype {
		errors.Bug("tagmapHeader.decode: invalid ftype: %x - expect: %x",
			h.ftype, mmap_tagmap_ftype)
	}
	crc64 := digest.Checksum64(buf[16:])
	if crc64 != h.crc64 {
		errors.Bug("tagmapHeader.decode: invalid checksum: %d - expect: %d",
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
func CreateTagmapFile(tag string) error {

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
	var h = &tagmapHeader{
		ftype:   mmap_tagmap_ftype,
		created: now,
		updated: now,
	}

	var buf [tagmapHeaderSize]byte
	if e := h.encode(buf[:]); e != nil {
		return e
	}

	_, e = file.Write(buf[:])
	if e != nil {
		return errors.ErrorWithCause(e, "createTagmap: tag: %s", tag)
	}
	return nil
}

type Tagmap struct {
	Header *tagmapHeader
	Bitmap *bitmap.Wahl
}

// Loads the tagmap (in form of bitmap.Wahl) from file and closes the file.
// File is openned in private, read-only mode.
func LoadTagmap(tag string, headerOnly bool) (*Tagmap, error) {

	/// open file ///////////////////////////////////////////////////

	filename := TagmapFilename(tag)
	file, e := os.OpenFile(filename, os.O_RDONLY, system.FilePerm)
	if e != nil {
		return nil, e
	}
	defer file.Close()

	finfo, e := file.Stat()
	if e != nil {
		return nil, e
	}

	/// mmap it /////////////////////////////////////////////////////

	var fd = int(file.Fd())
	var prot = syscall.PROT_READ
	var flags = syscall.MAP_PRIVATE
	buf, e := syscall.Mmap(fd, 0, int(finfo.Size()), prot, flags)
	if e != nil {
		return nil, e
	}
	defer syscall.Munmap(buf)

	/// decode content //////////////////////////////////////////////

	// decode verifies header
	var hdr tagmapHeader
	if e := hdr.decode(buf); e != nil {
		return nil, e
	}
	if headerOnly {
		return &Tagmap{&hdr, nil}, nil
	}

	var wahl bitmap.Wahl
	if e := wahl.Decode(buf[tagmapHeaderSize:]); e != nil {
		return nil, e
	}
	return &Tagmap{&hdr, &wahl}, nil
}

// TODO create swap file, write to it, close it, done.
func SaveTagmap(tag string, wahl *bitmap.Wahl) error {
	return errors.NotImplemented("index.SaveTagmap")
}
