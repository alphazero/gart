/* DOOST! The Incomparably Gracious The Incomparably Merciful! */

package clc7

import (
	"flag"
	"fmt"
	"groupcache/lru"
	"kaveh/lib/hash"
	"kaveh/lib/prng"
	"reflect"
	"syscall"
	"time"
	"unsafe"
)

// Memory Object System Flags
const (
	MO_SYS_PROVIDED byte = 1   // ours or yours
	MO_SYS_READ     byte = 2   // read access REVU when is it not?
	MO_SYS_WRITE    byte = 4   // write access
	MO_SYS_PRIVATE  byte = 8   // (virt) process private
	MO_SYS_ANON     byte = 16  // sys prov only -- direct physic mem
	MO_SYS_MLOCK    byte = 32  // sys prov only -- V.M. cache flag
	MO_SYS_RSRV_0   byte = 64  //
	MO_SYS_RSRV_1   byte = 128 //
)

// woulda if CUDA
type clcmemo_ideal_but_unsafe struct {
	mask      uint64   // clc-id mask
	data      uintptr  // physical memory ptr - main containers
	data1     uintptr  // physical memory ptr - type specific optional
	degree    uint8    // number of clcs
	fsys      byte     // system flags
	ftyp      byte     // type flags
	fobj      byte     // object flags
	_reserved [32]byte // maybe fsobj, mutex, etc.
}

func flagIsSet(osf, scf byte) bool {
	return osf&scf == scf
}

func syscallFlags(omsysflags byte) int {
	var scf int
	if flagIsSet(omsysflags, MO_SYS_ANON) {
		scf |= syscall.MAP_ANON
	}
	if flagIsSet(omsysflags, MO_SYS_PRIVATE) {
		scf |= syscall.MAP_PRIVATE
	}
	return scf
}

func syscallProts(omsysflags byte) int {
	var scp int
	if flagIsSet(omsysflags, MO_SYS_READ) {
		scp |= syscall.PROT_READ
	}
	if flagIsSet(omsysflags, MO_SYS_WRITE) {
		scp |= syscall.PROT_WRITE
	}
	return scp
}

// This entire production is centered around this number, which
// may in fact not be correct on some curernt or future machines.
// But for now, make hay while the Sun is shining!
const cacheline_size = 64

// Struct defines a generic CLC memory object, clcmemo for short.
// This is the top level m.o. that has []byte memory backing for
// N >= 1 virtual CLC buckets, corresponding to a 2^degree (d) CLC.
//
// To keep these m.o. structs 64byte aligned, we're being a bit
// parsimonius [get it?] with not storing computable bits.
//    N == mask + 1
//    d == log N / log 2
//    size == N * sizeof(clc) -- aka 64bytes
// degree (d) is the key info but we use mask in hot loops so
// we're holding on to that.
//
// (Later on we can embed this in another struct e.g. persistentCLC
// and add other refs and flags there for more enriched info.)
//
// data and data1 are either user provided or obtained from the
// system. we can about alignment and size mostly, but of course
// user provided slices may present other issues but that's their
// responsibility.
//
// memory object system flags are (presently) mainly of interest
// in context of system provisioned physical memory. See the flag
// defs, e.g. OM_SYS_ANON, for details.
//
// memory object type flags are the (presently) 8 bits left for use
// by the semantic CLC types, such as CLC-7ULL-i.
type clcmemo struct {
	mask uint64 // size, degree and N are all computable from mask
	data []byte //
	//	data1 []byte //
	pdata uintptr // temp experiment
	d, d0 uint8
	fsys  byte //
	ftyp  byte //
}

func (clc *clcmemo) String() string {
	panic("not implemented")
}

func newclcmemoDefault(degree uint8) (*clcmemo, error) {
	var sysflags = MO_SYS_PRIVATE | MO_SYS_READ | MO_SYS_WRITE | MO_SYS_ANON

	//	if degree < 24 {
	//		sysflags |= MO_SYS_MLOCK
	//	}
	return newclcmemo(degree, sysflags)
}

// REVU: just need to extract the syscall stuff and allow for user slices
// REVU: need to check system memory IFF MO_SYS_MLOCK is specified and size is > available
func newclcmemo(degree uint8, omsysflags byte) (*clcmemo, error) {
	var n = 1 << degree
	var size = n * cacheline_size
	var flags = syscallFlags(omsysflags)
	var prots = syscallProts(omsysflags)

	// sys alloc the memory per spec'd flags and protections
	// this version returns a slice[] AND is OS agnostic
	// & we can use the syscall.Munmap/Munlock w/out []hdr concerns.
	data, err := syscall.Mmap(0, 0, int(size), prots, flags)
	if err != nil {
		return nil, fmt.Errorf("err - syscall.Mmap fail - %v\n", err)
	}
	var dshdr = (*reflect.SliceHeader)(unsafe.Pointer(&data))
	verifyDataSliceHeader(dshdr, int(size)) // panics

	/* panics */
	if flagIsSet(omsysflags, MO_SYS_MLOCK) {
		fmt.Printf("warn - locking memory pages\n")
		const fmterror = "sys-fault: syscall.Mlock failed <%v>"
		const fmtpanic = "sys-fault: syscall.Munmap failed <%v> : on Mlock error <%v>"
		if e := syscall.Mlock(data); e != nil {
			if e2 := syscall.Munmap(data); e2 != nil {
				panic(fmt.Errorf(fmtpanic, e2, e))
			}
			return nil, fmt.Errorf(fmterror, e)
		}
	}

	// wrap it up and ...
	var c clcmemo
	c.data = data
	c.pdata = dshdr.Data
	c.d = degree
	c.d0 = 64 - degree
	c.mask = uint64(n - 1)
	//	fmt.Printf("debug - newclcmemo - c.mask:%08x %32b\n", c.mask, c.mask)
	c.fsys = omsysflags | MO_SYS_PROVIDED

	/* tamoom shod. happy birthday! */
	return &c, nil
}

// panics
func verifyDataSliceHeader(dshdr *reflect.SliceHeader, size int) {
	const fmtpanic = "sys-fault: syscall.Mmap: sliceHeader.%s: exp:%v have:%v"
	if dshdr.Len != size {
		panic(fmt.Errorf(fmtpanic, "Len", size, dshdr.Len))
	}
	if dshdr.Cap != size {
		panic(fmt.Errorf(fmtpanic, "Cap", size, dshdr.Cap))
	}
	// check non-zero and 64byte alignment
	var zvUintptr uintptr
	if dshdr.Data == zvUintptr {
		panic(fmt.Errorf(fmtpanic, "Data", "non-zero", dshdr.Data))
	}
	if (dshdr.Data % cacheline_size) != 0 {
		panic(fmt.Errorf(fmtpanic, "Data alignment", cacheline_size, dshdr.Data))
	}
}

func (clc *clcmemo) Dispose() error {
	panic("not implemented!")
}

/// adhoc lru //////////////////////////////////////////////////
type c7uli_rec struct {
	H uint32 // meta state
	L uint32 // systolic state
}
type C7ULi clcmemo

var hfn = hash.TT32

func NewC7ULiWithHash(degree uint8, hfn0 func(uint32) uint32) (*C7ULi, error) {
	if hfn0 != nil {
		hfn = hfn0
	}
	return NewC7ULi(degree)
}
func NewC7ULi(degree uint8) (*C7ULi, error) {
	var c, e = newclcmemoDefault(degree)
	if e == nil {
		var n = uint(1 << c.d)
		var p = uintptr(unsafe.Pointer(&c.data[0]))
		for i := uint(0); i < n; i++ {
			var r0 = (*c7uli_rec)(unsafe.Pointer(p + uintptr(i<<6)))
			(*r0).H = 0xBAABDAAD
			(*r0).L = 0x07654321
		}
	}
	return (*C7ULi)(c), e
}

func (c *C7ULi) Put(key, value uint32) (removed bool, k0, v0 uint32) {
	return (*clcmemo)(c).put_lru(key, value)
}
func (c *clcmemo) put_lru(key, value uint32) (removed bool, k0, v0 uint32) {

	//	var p = c.pdata + uintptr(hash.TT32(key)&uint32(c.mask)<<6)
	var p = c.pdata + uintptr(hfn(key)&uint32(c.mask)<<6) // use hfn
	var SR *uint32 = (*uint32)(unsafe.Pointer(p + 4))

	var sr_7 = *SR << 4 >> 28 // LRU always at SR-7
	*SR = *SR<<8>>4 | sr_7    // systolic shift

	var r7p = (*c7uli_rec)(unsafe.Pointer(p + uintptr((sr_7 << 3))))
	k0 = r7p.H
	r7p.H = key
	v0 = r7p.L
	r7p.L = value

	removed = k0 > 0

	return
}

func (c *C7ULi) Get(key uint32) (found bool, value uint32) {
	return (*clcmemo)(c).get_lru(key)
}

func (c *clcmemo) get_lru(key uint32) (found bool, value uint32) {
	//	var p = c.pdata + uintptr(hash.TT32(key)&uint32(c.mask)<<6)
	var p = c.pdata + uintptr(hfn(key)&uint32(c.mask)<<6) // use hfn
	var SR *uint32 = (*uint32)(unsafe.Pointer(p + 4))

	// use FARSI to get RS
	var RS [8]uint8
	var rs = uintptr(unsafe.Pointer(&RS[0]))
	*(*uint8)(unsafe.Pointer(rs + uintptr(*SR<<28>>28))) = 1
	*(*uint8)(unsafe.Pointer(rs + uintptr(*SR<<24>>28))) = 2
	*(*uint8)(unsafe.Pointer(rs + uintptr(*SR<<20>>28))) = 3
	*(*uint8)(unsafe.Pointer(rs + uintptr(*SR<<16>>28))) = 4
	*(*uint8)(unsafe.Pointer(rs + uintptr(*SR<<12>>28))) = 5
	*(*uint8)(unsafe.Pointer(rs + uintptr(*SR<<8>>28))) = 6
	*(*uint8)(unsafe.Pointer(rs + uintptr(*SR<<4>>28))) = 7

	// lookup key
	var i uint8
	p += 8
	if key == *(*uint32)(unsafe.Pointer(p)) {
		i = *(*uint8)(unsafe.Pointer(rs + 1)) << 2
		value = *(*uint32)(unsafe.Pointer(p + 4))
		goto found
	}
	p += 8
	if key == *(*uint32)(unsafe.Pointer(p)) {
		i = *(*uint8)(unsafe.Pointer(rs + 2)) << 2
		value = *(*uint32)(unsafe.Pointer(p + 4))
		goto found
	}
	p += 8
	if key == *(*uint32)(unsafe.Pointer(p)) {
		i = *(*uint8)(unsafe.Pointer(rs + 3)) << 2
		value = *(*uint32)(unsafe.Pointer(p + 4))
		goto found
	}
	p += 8
	if key == *(*uint32)(unsafe.Pointer(p)) {
		i = *(*uint8)(unsafe.Pointer(rs + 4)) << 2
		value = *(*uint32)(unsafe.Pointer(p + 4))
		goto found
	}
	p += 8
	if key == *(*uint32)(unsafe.Pointer(p)) {
		i = *(*uint8)(unsafe.Pointer(rs + 5)) << 2
		value = *(*uint32)(unsafe.Pointer(p + 4))
		goto found
	}
	p += 8
	if key == *(*uint32)(unsafe.Pointer(p)) {
		i = *(*uint8)(unsafe.Pointer(rs + 6)) << 2
		value = *(*uint32)(unsafe.Pointer(p + 4))
		goto found
	}
	p += 8
	if key == *(*uint32)(unsafe.Pointer(p)) {
		i = *(*uint8)(unsafe.Pointer(rs + 7)) << 2
		value = *(*uint32)(unsafe.Pointer(p + 4))
		goto found
	}
	return // not found

found:
	var m_get_lru = [32]uint32{0, 0, 0, 0,
		0xfffffff0, 0x0000000f, 0, 0x00000000,
		0xffffff00, 0x000000f0, 4, 0x0000000f,
		0xfffff000, 0x00000f00, 8, 0x000000ff,
		0xffff0000, 0x0000f000, 12, 0x00000fff,
		0xfff00000, 0x000f0000, 16, 0x0000ffff,
		0xff000000, 0x00f00000, 20, 0x000fffff,
		0xf0000000, 0x0f000000, 24, 0x00ffffff}
	// systolic shift for GET:LRU @ R[i]
	var mup = uintptr(unsafe.Pointer(&m_get_lru[i]))
	_SRs := *SR & *(*uint32)(unsafe.Pointer(mup))
	mup += 4
	_SRp := *SR & *(*uint32)(unsafe.Pointer(mup))
	mup += 4
	_SRp >>= *(*uint32)(unsafe.Pointer(mup))
	mup += 4
	_SRc := (*SR & *(*uint32)(unsafe.Pointer(mup))) << 4
	*SR = _SRs | _SRp | _SRc

	return true, value
}

/// adhoc test /////////////////////////////////////////////////

var degree = uint(9)
var iters = uint(1000000)
var ggc_nocap bool

func main() {
	fmt.Printf("Salaam!\n")
	flag.UintVar(&degree, "d", degree, "degree")
	flag.UintVar(&iters, "i", iters, "iters")
	flag.BoolVar(&ggc_nocap, "ggc-nocap", ggc_nocap, "don't set a limit for GGC-LRU")
	flag.Parse()

	var capacity = int((1 << degree) * 7)
	// ------- bench it
	if false {
		// bench others before allocating the large locked pages
		var ggc_cap = capacity
		if ggc_nocap {
			ggc_cap = 0
		}
		fmt.Printf("\n* * * GGC-LRU * * * [capacity:%d] \n", ggc_cap)
		benchGCLRU(ggc_cap)
	}
	var c, e = NewC7ULi(uint8(degree))
	if e != nil {
		return
	}
	//	examine(clc)

	fmt.Printf("\n* * * CLC-LRU * * * [capacity:%d] \n", capacity)
	benchCLC((*clcmemo)(c))
	return
}

func benchCLC(clc *clcmemo) {
	var rand = prng.NewXorShiftStar("FRIEND")

	var t0 = time.Now().UnixNano()
	var k uint32
	for i := uint(0); i < iters; i++ {
		k = rand.Uint32()
	}
	var dtrand = time.Now().UnixNano() - t0

	t0 = time.Now().UnixNano()
	for i := uint(0); i < iters; i++ {
		k = rand.Uint32()
		_, _, _ = clc.put_lru(k, k)
	}
	var dtput = time.Now().UnixNano() - t0 - dtrand
	report("put", iters, uint(dtput))

	t0 = time.Now().UnixNano()
	for i := uint(0); i < iters; i++ {
		k = rand.Uint32()
		_, _, _ = clc.put_lru(k, k)
		_, _ = clc.get_lru(k)
	}
	var dt = time.Now().UnixNano() - t0 - dtrand - dtput
	report("get", iters, uint(dt))

	return
}
func benchGCLRU(capacity int) {
	var rand = prng.NewXorShiftStar("FRIEND")
	var gclru = lru.New(capacity)

	var t0 = time.Now().UnixNano()
	var k uint32
	for i := uint(0); i < iters; i++ {
		k = rand.Uint32()
	}
	var dtrand = time.Now().UnixNano() - t0

	t0 = time.Now().UnixNano()
	for i := uint(0); i < iters; i++ {
		k = rand.Uint32()
		gclru.Add(k, k)
	}
	var dtput = time.Now().UnixNano() - t0 - dtrand
	report("gc-lru put", iters, uint(dtput))

	t0 = time.Now().UnixNano()
	for i := uint(0); i < iters; i++ {
		k = rand.Uint32()
		gclru.Add(k, k)
		_, _ = gclru.Get(k)
	}
	var dt = time.Now().UnixNano() - t0 - dtrand - dtput
	report("gc-lru get", iters, uint(dt))

	return
}
func examine(c *clcmemo) {
	var usp = unsafe.Pointer(c)
	var dusp = unsafe.Pointer(&c.data)
	var sh = (*reflect.SliceHeader)(unsafe.Pointer(&c.data))
	fmt.Printf("clc:  %v\n", c)
	fmt.Printf("usp:  %v\n", usp)
	fmt.Printf("dusp: %08x\n", dusp)
	fmt.Printf("sh:   ptr:%08x len:%d cap:%d\n", sh.Data, sh.Len, sh.Cap)

	var foo interface{} = ""
	fmt.Printf("sizeof-unsafe.Pointer: %d\n", unsafe.Sizeof(usp))
	fmt.Printf("sizeof-uintptr:        %d\n", unsafe.Sizeof(uintptr(usp)))
	fmt.Printf("sizeof-interface{}:    %d\n", unsafe.Sizeof(foo))
	fmt.Printf("sizeof-clc:            %d\n", unsafe.Sizeof(*c))
	fmt.Printf("alignof-clc:           %d\n", unsafe.Alignof(*c))
	fmt.Printf("alignof-clc:           %d\n", unsafe.Alignof(c.data))

	return
}
func report(info string, n uint, dt uint) {
	var rate = (n * 1000) / dt // ops/usec
	var nspo = float64(dt) / float64(n)

	fmt.Printf("=== report ================\n")
	fmt.Printf("%d %s ops @ %d nsecs\n", n, info, dt)
	fmt.Printf("%d ops / usec\n", rate)
	fmt.Printf("%.3f nsecs/op\n", nspo)

	return
}
