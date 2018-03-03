// Doost

package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"
	"unsafe"
)

var random = rand.New(rand.NewSource(0))

// test updating a \n delimited var-rec file in place.

/// test record ///////////////////////////////////////////////////////////////

// Record is variable length and persisted with \n record delim.
type Record struct {
	flags  byte
	id     uint32
	refcnt uint32
	val    string // REVU TODO must insure that the byte len is < 251
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
		if random.Intn(10) == 0 {
			val[i] = ' '
		} else if random.Intn(20) == 0 {
			val[i] = '-'
		} else {
			val[i] = 'v'
		}
	}
	return Record{0x00, 0, 0, string(val)}
}

func (r Record) String() string {
	return fmt.Sprintf("%08b %d %d %q", r.flags, r.id, r.refcnt, r.val)
}

func (r Record) Encode(w io.Writer) (int, error) {
	var wlen int

	var aflag = [1]byte{r.flags}
	if n, e := w.Write(aflag[:]); e != nil || n < 1 {
		return n, fmt.Errorf("Record.Encode - flag - wrote %d bytes with error %v", n, e)
	} else {
		wlen += n
	}
	var apid = *(*[4]byte)(unsafe.Pointer(&(r.id)))
	if n, e := w.Write(apid[:]); e != nil || n < 4 {
		return n, fmt.Errorf("Record.Encode - pid- wrote %d bytes with error %v", n, e)
	} else {
		wlen += n
	}
	var arefcnt = *(*[4]byte)(unsafe.Pointer(&(r.refcnt)))
	if n, e := w.Write(arefcnt[:]); e != nil || n < 4 {
		return n, fmt.Errorf("Record.Encode - refcnt- wrote %d bytes with error %v", n, e)
	} else {
		wlen += n
	}
	var vlen = [1]byte{byte(len(r.val))} // REVU insure vlen is always less than 256? TODO ..
	if n, e := w.Write(vlen[:]); e != nil || n < 1 {
		return n, fmt.Errorf("Record.Encode - vlen - wrote %d bytes with error %v", n, e)
	} else {
		wlen += n
	}
	if n, e := w.Write([]byte(r.val)); e != nil || n < len(r.val) {
		return n, fmt.Errorf("Record.Encode - val - wrote %d bytes with error %v", n, e)
	} else {
		wlen += n
	}
	return wlen, nil
}

func (p *Record) Decode(r io.Reader) (int, error) {
	var rlen int

	var aflags [1]byte
	if n, e := r.Read(aflags[:]); e != nil || n < 1 {
		return n, fmt.Errorf("Record.Decode - flag read %d bytes with error %v", n, e)
	} else {
		rlen += n
	}
	var apid [4]byte
	if n, e := r.Read(apid[:]); e != nil || n < 4 {
		return rlen + n, fmt.Errorf("Record.Decode - pid read %d bytes with error %v", n, e)
	} else {
		rlen += n
	}
	var arefcnt [4]byte
	if n, e := r.Read(arefcnt[:]); e != nil || n < 4 {
		return rlen + n, fmt.Errorf("Record.Decode - refcnt read %d bytes with error %v", n, e)
	} else {
		rlen += n
	}
	var avlen [1]byte
	if n, e := r.Read(avlen[:]); e != nil || n < 1 {
		return rlen + n, fmt.Errorf("Record.Decode - vlen read %d bytes with error %v", n, e)
	} else {
		rlen += n
	}
	vlen := int(avlen[0])
	//	fmt.Printf("debug - vlen %d\n", vlen)
	var val = make([]byte, vlen)
	if n, e := r.Read(val[:]); e != nil || n < vlen {
		//		println(len(val))
		return rlen + n, fmt.Errorf("Record.Decode - val read %d bytes of %d with error %v", n, vlen, e)
	} else {
		rlen += n
	}
	p.flags = aflags[0]
	p.id = *(*uint32)(unsafe.Pointer(&apid[0]))
	p.refcnt = *(*uint32)(unsafe.Pointer(&arefcnt[0]))
	p.val = string(val)
	return rlen, nil
}

/// test //////////////////////////////////////////////////////////////////////

func main() {

	var items = 1 << 10
	const fname = "tempfile"

	// create file, write rand records, and close
	var start = time.Now().UnixNano()
	if n, e := writeRandomFile(fname, items); e != nil {
		exitOnError(fmt.Errorf("err - wrote %d records - error %v", n, e))
	} else {
		var delta = time.Now().UnixNano() - start
		fmt.Printf("write: n:%d delta-ns:%v ns/page:%f\n", n, time.Duration(delta), float64(delta)/float64(n))
		fmt.Printf("wrote %d records\n", n)
	}
	start = time.Now().UnixNano()
	if n, e := verifyFileWrite(fname, items); e != nil {
		exitOnError(fmt.Errorf("read %d records - error %v", n, e))
	} else {
		var delta = time.Now().UnixNano() - start
		fmt.Printf("read: n:%d delta-ns:%v ns/page:%f\n", n, time.Duration(delta), float64(delta)/float64(n))
		fmt.Printf("verified %d records\n", n)
	}
	os.Exit(0)
	// open file in O_RDWR and emit records.
}

var E_IllegalArgument = fmt.Errorf("err - illegal argument")
var recterm = []byte("\n")

func verifyFileWrite(fname string, items int) (int, error) {

	//	const flags = os.O_RDONLY
	fi, e := os.Stat(fname)
	if e != nil {
		panic(e)
	}

	file, e := os.OpenFile(fname, os.O_RDONLY, 0644)
	if e != nil {
		return 0, e
	}
	defer file.Close()

	var tot int
	var r = bufio.NewReaderSize(file, int(fi.Size()))

	var hdr [8]byte
	if n, e := io.ReadFull(r, hdr[:]); e != nil {
		return n, fmt.Errorf("read-file - header - %s\n", e)
	}
	var rcnt = *(*int)(unsafe.Pointer(&hdr[0]))
	fmt.Printf("rcnt: %d\n", rcnt)
	for i := 0; i < items; i++ {
		var record Record
		n, e := record.Decode(r)
		//n, e := record.Decode(file)
		if e != nil {
			return i, e
		}
		tot += n
		//		fmt.Printf("%s\n", record)
		//		fmt.Printf("tot: %d\n", tot)
	}
	return items, nil
}

func writeRandomFile(fname string, items int) (int, error) {
	const flags = os.O_CREATE | os.O_EXCL | os.O_WRONLY | os.O_APPEND //REVU is O_SYNC necessary?
	file, e := os.OpenFile(fname, flags, 0644)
	if e != nil {
		return 0, e
	}
	defer file.Close()
	//	defer file.Sync()

	var w = bufio.NewWriter(file)
	defer w.Flush()

	var hdr = *(*[8]byte)(unsafe.Pointer(&(items)))
	if n, e := w.Write(hdr[:]); e != nil || n != len(hdr) {
		return n, fmt.Errorf("write-file - header - %s", e)
	}

	for i := 0; i < items; i++ {
		var record Record
		record = record.randomRecord(256)
		record.id = uint32(i)
		record.refcnt = 1
		record.flags = 0xff
		if _, e := record.Encode(w); e != nil {
			return i, e
		}
		if e := file.Sync(); e != nil {
			panic(e)
		}
	}
	return items, nil
}

/// helpers ///////////////////////////////////////////////////////////////////

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "%s\n", e)
	os.Exit(1)
}
