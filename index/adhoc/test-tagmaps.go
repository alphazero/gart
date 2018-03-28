// Doost!

package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/alphazero/gart/index"
	"github.com/alphazero/gart/syslib/bitmap"
	"github.com/alphazero/gart/syslib/errors"
)

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
		e = readTagmap(tagnames...)
	case "w":
		e = writeTagmap(tagnames[0])
	case "q":
		e = queryTagmaps(tagnames...)
	default:
		exitOnError(errors.Bug("verified op is not known: %q", option.op))
	}

	if e != nil {
		exitOnError(errors.Bug("op: %q - %v", option.op, e))
	}

	os.Exit(0)
}

func createTagmaps(tags ...string) error {
	for _, tag := range tags {
		tagmap, e := index.CreateTagmap(tag)
		if e != nil {
			// just print the errors
			fmt.Fprintf(os.Stderr, "on CreateTagmap %q - %v\n", tag, e)
			continue
		}
		fmt.Printf("created tagmap for %q\n", tag)
		tagmap.Print(os.Stdout)
		fmt.Println()
	}
	return nil
}

func readTagmap(tags ...string) error {
	var create bool = false
	for _, tag := range tags {
		tagmap, e := index.LoadTagmap(tag, create)
		if e != nil {
			// just print the errors
			fmt.Fprintf(os.Stderr, "on LoadTagmap(%q, false) - %v\n", tag, e)
			continue
		}
		tagmap.Print(os.Stdout)
	}

	return nil
}

func writeTagmap(tag string) error {
	var wahl = bitmap.NewWahl()
	keys := randomKeys(999, 1, 1132)
	wahl.Set(keys...)
	wahl.Compress()
	wahl.Print(os.Stdout)
	fmt.Printf("/// ^^ test data ^^ ///\n")

	// REVU zerolen invalid arg in wahl.decode should be fixed in tagmaps.
	//      CreateTagmap -will- create tagmaps with len 0.
	tagmap, e := index.LoadTagmap(tag, true)
	if e != nil {
		return errors.ErrorWithCause(e, "writeTagmap: LoadTagmap")
	}
	fmt.Printf("debug - writeTagmap: loaded tagamp for %q\n", tag)
	tagmap.Print(os.Stdout)
	fmt.Println()

	tagmap.Update(keys...) // REVU wahl.Set (and thus Tagmap.Update need to return an updated..

	fmt.Printf("debug - writeTagmap: updatd tagamp for %q\n", tag)
	tagmap.Print(os.Stdout)
	fmt.Println()

	ok, e := tagmap.Save()
	if e != nil {
		exitOnError(e)
	}
	if ok {
		fmt.Printf("debug - writeTagmap: wrote tagamp for %q\n", tag)
	}

	return nil
}

func queryTagmaps(tags ...string) error {
	return errors.NotImplemented("queryTagmaps")
}

var random = rand.New(rand.NewSource(0))

/// little helpers /////////////////////////////////////////////////////////////
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
