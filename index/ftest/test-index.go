// Doost!

package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/alphazero/gart/index"
	"github.com/alphazero/gart/syslib/digest"
	"github.com/alphazero/gart/syslib/errors"
	"github.com/alphazero/gart/system"
)

/// adhoc test /////////////////////////////////////////////////////////////////

var option = struct {
	op    string
	text  string
	file  string
	tags  string
	force bool
}{}
var ops = []string{"i", "r", "a", "d", "u", "q"}

func init() {
	flag.StringVar(&option.op, "op", option.op, "i:init a:add d:delete r:read u:update q:query")
	flag.StringVar(&option.file, "file", option.file, "name of file to index")
	flag.StringVar(&option.text, "text", option.text, "text to index")
	flag.StringVar(&option.tags, "tags", option.tags, "csv list in \" \"s ")
	flag.BoolVar(&option.force, "force", option.force, "force re-init (with op i only)")
}

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "exit on error: %v\n", e)
	os.Exit(1)
}

func main() {
	fmt.Printf("Salaam Samad OMNI Sultan of LOVE!\n")

	// parse flags and verify option
	flag.Parse()

	if option.op == "" {
		exitOnError(errors.Usage("flag -op must be specified"))
	}
	if (option.op != "i" && option.op != "q") && (option.file == "" && option.text == "") {
		exitOnError(errors.Usage("either text or file must be specified for op %q", option.op))
	}
	if (option.op == "q") && (option.tags == "") {
		exitOnError(errors.Usage("-tags must be specified for op %q", option.op))
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
	var tagnames = []string{}
	if option.tags != "" {
		tagnames = strings.Split(option.tags, ",")
		if option.op == "i" {
			goto tags_ok
		}
	}

	if option.op == "r" && len(tagnames) > 0 {
		system.Debugf("test-index: ignoring -tags for option 'r'")
		goto tags_ok
	}

	fmt.Printf("tagnames %d %q\n", len(tagnames), option.tags)
	if option.op == "d" && len(tagnames) == 0 {
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
	if option.op == "d" && len(tagnames) == 0 && option.file == "" && option.text == "" {
		exitOnError(errors.Usage("test-index: -op d requires either text, file, or tags flag"))
	}

tags_ok:

	var e error
	switch option.op {
	case "i":
		e = initializeIndex(option.force)
	case "a":
		e = addObject(tagnames...)
	case "d":
		e = deleteOp(tagnames...)
	case "r":
		e = readCard()
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

func readCard() error {

	var e error
	var oid *system.Oid
	switch {
	case option.text != "":
		md := digest.Sum([]byte(option.text))
		oid, e = system.NewOid(md[:])
		if e != nil {
			exitOnError(errors.Bug("test-index: readCard: oid:  - %s", e))
		}
	case option.file != "":
		md, e := digest.SumFile(option.file)
		if e != nil {
			exitOnError(errors.Bug("test-index: readCard: oid:  - %s", e))
		}
		oid, e = system.NewOid(md[:])
		if e != nil {
			exitOnError(errors.Bug("test-index: readCard: oid:  - %s", e))
		}
	}
	card, e := index.LoadCard(oid)
	if e != nil {
		exitOnError(errors.Bug("test-index: index.LoadCard - %s", e))
	}
	card.Print(os.Stdout)
	return nil
}

func deleteOp(tags ...string) error {
	idx, e := index.OpenIndexManager(index.Write)
	if e != nil {
		return e
	}
	defer func() {
		if e := idx.Close(); e != nil {
			panic(errors.BugWithCause(e, "on deferred close of indexManager"))
		}
		log("closed indexManager")
	}()

	var oid *system.Oid
	switch {
	case option.text != "":
		md := digest.Sum([]byte(option.text))
		oid, e = system.NewOid(md[:])
		if e != nil {
			exitOnError(errors.Bug("test-index: readCard: oid:  - %s", e))
		}
	case option.file != "":
		md, e := digest.SumFile(option.file)
		if e != nil {
			exitOnError(errors.Bug("test-index: readCard: oid:  - %s", e))
		}
		oid, e = system.NewOid(md[:])
		if e != nil {
			exitOnError(errors.Bug("test-index: readCard: oid:  - %s", e))
		}
	case len(tags) != 0:
		return deleteObjectsByTag(idx, tags...)
	}

	ok, e := idx.DeleteObject(oid)
	if e != nil {
		log("error on idx.DeleteObject(%s)", oid.Fingerprint())
	}
	if !ok {
		log("DeleteObject (oid:%s) return false, nil", oid.Fingerprint())
		return nil
	}
	log("object (oid:%s) deleted", oid.Fingerprint())

	return nil
}

func deleteObjectsByTag(idx index.IndexManager, tags ...string) error {
	return errors.NotImplemented("test-index.deleteOp")
}

func addObject(tags ...string) error {
	idx, e := index.OpenIndexManager(index.Write)
	if e != nil {
		return e
	}
	defer func() {
		if e := idx.Close(); e != nil {
			panic(errors.BugWithCause(e, "on deferred close of indexManager"))
		}
		log("closed indexManager")
	}()

	var card index.Card
	var added bool
	switch {
	case option.text != "":
		card, added, e = idx.IndexText(option.text, tags...)
	case option.file != "":
		filename, _ := filepath.Abs(option.file)
		card, added, e = idx.IndexFile(filename, tags...)
	}
	if e != nil {
		return e
	}
	if !added {
		log("object (oid:%s, key:%d) already indexed", card.Oid(), card.Key())
	}
	log("indexed object (type:%s oid:%s key:%d added:%t)", card.Type(), card.Oid().Fingerprint(), card.Key(), added)

	card.Print(system.Writer)

	return nil
}

func queryByTag(tags ...string) error {
	idx, e := index.OpenIndexManager(index.Read)
	if e != nil {
		return e
	}
	defer func() {
		if e := idx.Close(); e != nil {
			panic(errors.BugWithCause(e, "on deferred close of indexManager"))
		}
		log("closed indexManager")
	}()

	system.Debugf("==== RUN QUERY =====================================")
	oids, e := idx.Select(index.All, tags...)
	if e != nil {
		exitOnError(errors.ErrorWithCause(e, "test-index: index.Select"))
	}
	for _, oid := range oids {
		fmt.Printf("Object (oid:%s) selected by tags:%v\n", oid.Fingerprint(), tags)
		card, e := index.LoadCard(oid)
		if e != nil {
			exitOnError(errors.Bug("test-index: index.LoadCard - %s", e))
		}
		card.Print(os.Stdout)
	}
	system.Debugf("==== RUN QUERY ============================= end ===")
	return nil
}

func updateObjectTags(oid *system.Oid, tags ...string) error {
	return errors.NotImplemented("adhoc-test.updateObjectTags - ? is this necessary ?")
}

/// little helpers /////////////////////////////////////////////////////////////

// little logger writes to os.Stderr
func log(fmtstr string, a ...interface{}) (int, error) {
	return fmt.Fprintf(os.Stderr, "test-index - "+fmtstr+"\n", a...)
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
