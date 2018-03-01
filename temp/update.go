// Doost

package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"unsafe"
)

var random = rand.New(rand.NewSource(0))

// test updating a \n delimited var-rec file in place.

/// test record ///////////////////////////////////////////////////////////////

// Record is variable length and persisted with \n record delim.
type Record struct {
	id     uint32
	refcnt uint32 // REVU use unum?
	val    string
}

func (Record) randomRecord(maxValLen int) Record {
	// rec len is 5 + value len
	valLen := random.Intn(maxValLen)
	if valLen < 3 {
		valLen = 3
	}
	// for now, just write valLen 'v'
	val := make([]byte, valLen)
	for i := range val {
		val[i] = 'v'
	}
	return Record{0, 0, string(val)}
}

/// test //////////////////////////////////////////////////////////////////////

func main() {

	var items = 80
	const fname = "tempfile"

	// create file, write rand records, and close
	if n, e := writeRandomFile(fname, items); e != nil {
		exitOnError(fmt.Errorf("err - wrote %d records - error %v", n, e))
	} else {
		fmt.Printf("wrote %d records\n", n)
	}
	if n, e := verifyFileWrite(fname, items); e != nil {
		exitOnError(fmt.Errorf("err - wrote %d records - error %v", n, e))
	} else {
		fmt.Printf("verified %d records\n", n)
	}
	os.Exit(0)
	// open file in O_RDWR and emit records.
}

var E_IllegalArgument = fmt.Errorf("err - illegal argument")
var recterm = []byte("\n")

func verifyFileWrite(fname string, items int) (int, error) {
	return items, nil
}

func writeRandomFile(fname string, items int) (int, error) {
	//	const flags = os.O_CREATE | os.O_EXCL | os.O_APPEND | os.O_SYNC // REVU is O_SYNC necessary?
	const flags = os.O_CREATE | os.O_EXCL | os.O_WRONLY | os.O_SYNC | os.O_APPEND // | os.O_SYNC // REVU is O_SYNC necessary?
	file, e := os.OpenFile(fname, flags, 0644)
	if e != nil {
		return 0, e
	}
	defer file.Close()
	defer file.Sync()

	var w = bufio.NewWriter(file)
	defer w.Flush()

	for i := 0; i < items; i++ {
		var record Record
		record = record.randomRecord(64)
		record.id = uint32(i)

		pid := (*[4]byte)(unsafe.Pointer(&record.id))
		refcnt := (*[4]byte)(unsafe.Pointer(&record.refcnt))
		sbuf := []byte(record.val)
		var e error
		e = verifyWrite(e, w, (*pid)[:], 4)
		e = verifyWrite(e, w, (*refcnt)[:], 4)
		e = verifyWrite(e, w, sbuf, len(sbuf))
		e = verifyWrite(e, w, recterm, 1)
		if e != nil {
			return i, e
		}
	}
	return items, nil
}

func verifyWrite(e error, w io.Writer, b []byte, n int) error {
	if e != nil {
		return e
	}
	n0, e0 := w.Write(b)
	if e0 != nil {
		return e0
	}
	if n0 != n {
		return fmt.Errorf("err - verifyWrite - expected %d  - wrote %d", n, n0)
	}
	return nil
}

/// helpers ///////////////////////////////////////////////////////////////////

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "%s\n", e)
	os.Exit(1)
}
