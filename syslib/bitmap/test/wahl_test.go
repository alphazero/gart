// Doost!

package test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/alphazero/gart/syslib/bitmap"
)

// func WahlBlock(v uint32) wahlBlock
// func blockRange(b uint32, max0 int) (uint, uint, int)
// not sure how to test these without effectively reproducing the same code in the
// test.

// test func NewWahl() *Wahl
// must:
//	- not return nil
//	- have max 0
//	- have len 0
//	- be compressible
//	- be decompressible
//  - have Bits() []int{}
//	- be encodable
//  - be decodable
func TestNewWahl(t *testing.T) {
	var w = bitmap.NewWahl()
	if w == nil {
		t.Error("NewWahl returned nil")
	}
	if wmax := w.Max(); wmax != 0 {
		t.Errorf("NewWahl().Max: %d - expected:%d", wmax, 0)
	}
	if wlen := w.Max(); wlen != 0 {
		t.Errorf("NewWahl().Len: %d - expected:%d", wlen, 0)
	}
	if ok := w.Compress(); ok {
		t.Error("NewWahl().Compress() returned true")
	}
	if ok := w.Decompress(); ok {
		t.Error("NewWahl().Decompress() returned true")
	}
	if bits := []int(w.Bits()); len(bits) > 0 {
		t.Errorf("NewWahl().Bits() returned %v", bits)
	}
	if e := w.Encode([]byte{}); e != nil {
		t.Errorf("NewWahl().Encode() returned %v", e)
	}
	if e := w.Decode([]byte{}); e != nil {
		t.Errorf("NewWahl().Decode() returned %v", e)
	}
}

/// test bitwise ops ////////////////////////////////////////////////////////
//var _ = time.Now
var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

const maxBit = 1 << 20 // large bitmaps to increase prob of testing all edge cases.
var w0 = NewRandomWahl(rnd, maxBit)
var w1 = NewRandomWahl(rnd, maxBit)

func TestNot(t *testing.T) {
	w0_not := w0.Not()
	if e := verifyNot(w0, w0_not); e != nil {
		t.Fatal(e)
	}
}

func TestAnd(t *testing.T) {
	w_and, e := bitmap.And(w0, w1)
	if e != nil {
		t.Fatal(e)
	}
	if e := verifyAnd(w0, w1, w_and); e != nil {
		t.Fatal(e)
	}
}

func TestOr(t *testing.T) {
	w_or, e := bitmap.Or(w0, w1)
	if e != nil {
		t.Fatal(e)
	}
	if e := verifyOr(w0, w1, w_or); e != nil {
		t.Fatal(e)
	}
}

func TestXor(t *testing.T) {
	w_xor, e := bitmap.Xor(w0, w1)
	if e != nil {
		t.Fatal(e)
	}
	if e := verifyXor(w0, w1, w_xor); e != nil {
		t.Fatal(e)
	}
}

// TODO verifyXor & testXor
// TODO test basic ops, clear, set, etc per below

// func NewWahlInit(bits ...uint) *Wahl {
// func AND(bitmaps ...*Wahl) (*Wahl, error) {
// func OR(bitmaps ...*Wahl) (*Wahl, error) {

// wahl.go:func (w *Wahl) Len() int { return len(w.arr) }
// wahl.go:func (w *Wahl) Size() int { return len(w.arr) << 2 }
// wahl.go:func AND(bitmaps ...*Wahl) (*Wahl, error) {
// wahl.go:func OR(bitmaps ...*Wahl) (*Wahl, error) {
// wahl.go:func (w *Wahl) Set(bits ...uint) bool {
// wahl.go:func (w *Wahl) Clear(bits ...uint) bool {
// wahl.go:func (w *Wahl) set(bitval bool, bits ...uint) bool {
// wahl.go:func (w *Wahl) DecompressTo(buf []uint32) (int, error) {
// wahl.go:func (w *Wahl) Decompress() bool {
// wahl.go:func (w *Wahl) Compress() bool {
// wahl.go:func (w Wahl) And(other *Wahl) (*Wahl, error) {
// wahl.go:func (w Wahl) Or(other *Wahl) (*Wahl, error) {
// wahl.go:func (w Wahl) Xor(other *Wahl) (*Wahl, error) {
// wahl.go:func (w Wahl) Bitwise(op bitwiseOp, other *Wahl) (*Wahl, error) {
// wahl.go:func (w *Wahl) Bits() Bitnums {
// wahl.go:func (w *Wahl) Max() int {
// wahl.go:func (w *Wahl) Encode(buf []byte) error {
// wahl.go:func (w *Wahl) Decode(buf []byte) error {
// wahl.go:func (w *Wahl) apply(visit visitFn) error {
