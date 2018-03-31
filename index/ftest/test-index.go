// Doost!

package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/alphazero/gart/index"
	"github.com/alphazero/gart/syslib/digest"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system"
)

/// adhoc test /////////////////////////////////////////////////////////////////

var option = struct {
	op    string
	file  string
	tags  string
	force bool
}{}
var ops = []string{"i", "a", "u", "q"}

func init() {
	flag.StringVar(&option.op, "op", option.op, "i:init a:add u:update q:query")
	flag.StringVar(&option.file, "f", option.file, "file name")
	flag.StringVar(&option.tags, "tags", option.tags, "csv list in \" \"s ")
	flag.BoolVar(&option.force, "force", option.force, "force re-init (with op i only)")
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
	if option.op != "i" && option.op != "q" && option.file == "" {
		exitOnError(errors.Usage("file must be specified for add & update ops."))
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
	if option.op == "i" {
		goto tags_ok
	}

	for i, s := range tagnames {
		tag := strings.Trim(s, " ")
		if tag == "" {
			exitOnError(errors.Usage("option -tags has zero-len tagname: %q", tagnames))
		}
		tagnames[i] = tag
	}
	if (option.op == "a" || option.op == "u") && len(tagnames) == 0 {
		exitOnError(errors.Usage("option -tags must be non-empty"))
	}

tags_ok:

	var e error
	switch option.op {
	case "i":
		e = initializeIndex(option.force)
	case "a":
		e = addObject(option.file, tagnames...)
	case "u":
		var oid = fileOid(option.file)
		e = updateObjectTags(oid, tagnames...)
	case "q":
		e = queryByTag(tagnames...)
	default:
		exitOnError(errors.Bug("verified op is not known: %q", option.op))
	}

	if e != nil {
		exitOnError(errors.Bug("op: %q - %v", option.op, e))
	}

	os.Exit(0)
}

func initializeIndex(force bool) error {
	if force == true {
		log("initializing index - reinit:%t", force)
	}
	return index.Initialize(force)
}
func addObject(filename string, tags ...string) error {
	var oid = fileOid(filename)

	idx, e := index.OpenIndexManager(index.Write)
	if e != nil {
		return e
	}
	defer func() {
		if e := idx.Close(); e != nil {
			panic(errors.BugWithCause(e, "on deferred close of indexManager"))
		}
		log("debug - closed indexManager")
	}()

	if e := idx.UsingTags(tags...); e != nil {
		return e
	}

	key, added, e := idx.IndexObject(oid, tags...)
	if e != nil {
		return e
	}
	if !added {
		log("debug - object (oid:%s, key:%d) already indexed",
			oid.Fingerprint(), key)
	}
	log("debug - indexed object (oid:%s, key:%d)", oid.Fingerprint(), key)

	return nil
}
func updateObjectTags(oid *system.Oid, tags ...string) error {
	return errors.NotImplemented("adhoc-test.updateObjectTags")
}
func queryByTag(tags ...string) error {
	return errors.NotImplemented("adhoc-test.queryByTag")
}

/// little helpers /////////////////////////////////////////////////////////////

// little logger writes to os.Stderr
func log(fmtstr string, a ...interface{}) (int, error) {
	return fmt.Fprintf(os.Stderr, "log - "+fmtstr+"\n", a...)
}

// exits on error
func fileOid(filename string) *system.Oid {
	md, e := digest.SumFile(filename)
	if e != nil {
		exitOnError(errors.ErrorWithCause(e, "adhoc-test:fileOid"))
	}
	oid, e := system.NewOid(md)
	if e != nil {
		exitOnError(errors.ErrorWithCause(e, "adhoc-test:fileOid"))
	}
	return oid
}

var random = rand.New(rand.NewSource(0))

func randomKeys(cnt, from, to int) []uint {
	keyset := make(map[uint]uint)
	dn := to - from
	for len(keyset) < cnt {
		key := uint(random.Intn(dn) + to)
		keyset[key] = key
	}
	keys := make([]uint, len(keyset))
	var i int
	for _, k := range keyset {
		keys[i] = k
		i++
	}
	return keys
}
