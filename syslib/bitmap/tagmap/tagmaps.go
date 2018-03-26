// Doost!

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	//	"syscall"
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

/// tagmaps prototype //////////////////////////////////////////////////////////

const headerSize = 512

type header struct {
	ftype    uint64
	crc64    uint64
	created  int64
	updated  int64
	reserved [480]byte
}

func (h header) encode(buf []byte) error {
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

// Returns the absolute path filename for the given tag.
func TagFilename(tag string) string {
	hash := fmt.Sprintf("%x.bitmap", digest.SumUint64([]byte(tag)))
	path := filepath.Join(system.IndexTagmapsPath, hash[:2])
	return filepath.Join(path, hash[2:])
}

// Creates the initial tagmap file for the given tag in the canonical
// repo location. Tag names in gart are case-insensitive and the tag
// (name) will always be converted to lower-case form.
func CreateTagmap(tag string) error {

	tag = strings.ToLower(tag)
	filename := TagFilename(tag)
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
		ftype:   0x5807263e43839459,
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

	return errors.NotImplemented("createTagmap - filename: %s", filename)
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
	return errors.NotImplemented("readTagmap")
}

func writeTagmap(tag string) error {
	return errors.NotImplemented("writeTagmap")
}

func queryTagmaps(tags ...string) error {
	return errors.NotImplemented("queryTagmaps")
}
