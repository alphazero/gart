// Doost!
package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/alphazero/gart/index"
	"github.com/alphazero/gart/syslib/fs"
	"github.com/alphazero/gart/system"
)

func exitOnError(e error) {
	fmt.Fprintf(os.Stderr, "%s\n", e)
	os.Exit(1)
}

func allIsWell(ok bool, e error) {
	if !ok {
		exitOnError(fmt.Errorf("not ok!"))
	}
	if e != nil {
		exitOnError(fmt.Errorf("has error: %s", e))
	}
}

func main() {
	fmt.Printf("Salaam Samad Sultan of LOVE!\n")
	fmt.Printf("\t=> L.L. :)\n")

	var paths = index.NewPaths()
	paths.Print(os.Stdout)
	fmt.Println()

	allIsWell(paths.Add("/Users/alphazero"))
	allIsWell(paths.Add("/Users/alphazero/Code"))
	allIsWell(paths.Add("/Users/alphazero/Code/go"))
	allIsWell(paths.Add("/Users/alphazero/Code/go/src"))
	allIsWell(paths.Add("/Users/alphazero/Code/go/src/gart"))
	paths.Print(os.Stdout)
	fmt.Println()

	testCodec(paths)

	if ok, e := paths.Add("/Users/alphazero/Code/go/src/gart"); ok {
		panic("bug - add returned true")
	} else if e != nil {
		panic("bug - add returned error - it should just return false")
	}

	allIsWell(paths.Remove("/Users/alphazero/Code/go/src"))
	allIsWell(paths.Remove("/Users/alphazero"))
	allIsWell(paths.Remove("/Users/alphazero/Code/go/src/gart"))
	paths.Print(os.Stdout)
	fmt.Println()

	if ok, e := paths.Remove("/Users/alphazero/Code/go/src/gart"); ok {
		panic("bug - remove returned true")
	} else if e != nil {
		panic("bug - remove returned error - it should just return false")
	}
}

func testCodec(paths *index.Paths) {

	/// create and encode ///////////////////////////////////////////

	file, e := fs.OpenNewFile("paths.dat", os.O_RDWR)
	if e != nil {
		fmt.Printf("err - testCodec - os.OpenFile - %s\n", e)
		return
	}
	filename := file.Name()
	defer os.Remove(filename) // clean up

	if e := file.Truncate(int64(paths.Buflen())); e != nil {
		fmt.Printf("err - testCodec - file.Truncate(%d) - %s\n", paths.Buflen(), e)
		return
	}
	buf, e := syscall.Mmap(int(file.Fd()), 0, paths.Buflen(), syscall.PROT_WRITE, syscall.MAP_SHARED)
	if e != nil {
		fmt.Printf("err - testCodec - syscall.Mmap -  %s\n", e)
		file.Close()
	}
	defer func() {
		syscall.Munmap(buf)
		file.Close()
	}()

	system.Debugf("testCode - created & mapped file name: %q\n", filename)

	if e := paths.Encode(buf); e != nil {
		fmt.Printf("err - testCodec - paths.Encode - buflen:%d - %s", len(buf), e)
		return
	}

	/// read and decode /////////////////////////////////////////////

	buf0, e := fs.ReadFull(filename)
	if e != nil {
		fmt.Printf("err - testCodec - fs.ReadFull -  %s\n", e)
		return
	}
	system.Debugf("testCodec - read: %d\n", len(buf0))

	var paths0 index.Paths
	if e := paths0.Decode(buf); e != nil {
		fmt.Printf("err - testCodec - fs.ReadFull -  %s\n", e)
		return
	}
	paths0.Print(os.Stdout)
}
