package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
)

var dictPath string
var substitute, verbose bool

func init() {
	flag.StringVar(&dictPath, "f", "/usr/share/dict/web2", "dictinary file")
	flag.BoolVar(&substitute, "sub", substitute, "substitute with digits")
	flag.BoolVar(&verbose, "verbose", verbose, "emit hex words")
}

const a byte = 'a'
const f byte = 'f'
const l byte = 'l' // -> 1
const o byte = 'o' // -> 0

func main() {
	flag.Parse()

	file, e := os.Open(dictPath)
	if e != nil {
		fmt.Fprintf(os.Stderr, "err - %v\n", e)
		os.Exit(1)
	}
	defer file.Close()

	r := bufio.NewReader(file)
	var done bool
	var word []byte
	for !done {
		word, done = readHexWord(r)
		if word != nil {
			if verbose && len(word) > 1 {
				fmt.Printf("%s\n", word)
			}
		}
	}
}

func readHexWord(r *bufio.Reader) ([]byte, bool) {
	var line []byte
	var onError = func(e error) {
		if e != io.EOF {
			fmt.Fprintf(os.Stderr, "err - %v\n", e)
		}
	}
	for {
		b, e := r.ReadByte()
		if e != nil {
			onError(e)
			return nil, true
		}
		if b == '\n' {
			return line, false
		}
		if substitute && (b == l || b == o) {
			line = append(line, b)
			continue
		} else if b < a || b > f {
			for b != '\n' {
				b, e = r.ReadByte()
				if e != nil {
					onError(e)
					return nil, true
				}
			}
			return nil, false
		}
		line = append(line, b)
	}
}
