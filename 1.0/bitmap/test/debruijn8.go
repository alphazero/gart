package main

import (
	"fmt"
	"testing"
)

const debruijn8 = 0x1d

var debruijn8Tab = [8]int{0, 1, 6, 2, 7, 5, 4, 3}

func oneBitIndex(v int) int {
	return debruijn8Tab[v*debruijn8>>5&7]
}

func BenchmarkTab(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if oneBitIndex(i) > 8 {
			return
		}
	}
}
func main() {
	//func TestNil(t *testing.T) {
	var a = []int{1, 2, 4, 8, 16, 32, 64, 128}

	for i, n := range a {
		if oneBitIndex(n) != i {
			panic(fmt.Sprintf("bug - oneBitIndex(%d) is not %d", i, i))
		}
		fmt.Printf("%08b %d\n", n, oneBitIndex(n))
	}
}
