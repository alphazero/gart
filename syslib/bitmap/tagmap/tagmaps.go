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
		e = createTagmaps(system.RepoPath, tagnames...)
	case "r":
		e = readTagmap(system.RepoPath, tagnames[0])
	case "w":
		e = writeTagmap(system.RepoPath, tagnames[0])
	case "q":
		e = queryTagmaps(system.RepoPath, tagnames...)
	default:
		exitOnError(errors.Bug("verified op is not known: %q", option.op))
	}

	if e != nil {
		exitOnError(errors.Bug("op: %q -  %v", option.op, e))
	}

	os.Exit(0)
}

/// op prototypes //////////////////////////////////////////////////////////////

func createTagmaps(repoDir string, tags ...string) error {
	for _, tag := range tags {
		if e := createTagmap(repoDir, tag); e != nil {
			return e
		}
	}
	return nil
}

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

func tagFilename(path, tag string) string {
	name := fmt.Sprintf("%x.bitmap", digest.SumUint64([]byte(tag)))
	filename := filepath.Join(path, name)
	return filename
}

func createTagmap(repoDir string, tag string) error {

	filename := tagFilename(repoDir, tag)

	file, e := fs.OpenNewFile(filename, os.O_WRONLY|os.O_APPEND)
	if e != nil {
		return fmt.Errorf("oidx.CreateIndex: %s", e)
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

func readTagmap(repoDir string, tag string) error {
	return errors.NotImplemented("readTagmap")
}

func writeTagmap(repoDir string, tag string) error {
	return errors.NotImplemented("writeTagmap")
}

func queryTagmaps(repoDir string, tags ...string) error {
	return errors.NotImplemented("queryTagmaps")
}
