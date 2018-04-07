// Doost

package bitmap

var clearMask32 = [31]uint32{
	0x7ffffffe,
	0x7ffffffd,
	0x7ffffffb,
	0x7ffffff7,
	0x7fffffef,
	0x7fffffdf,
	0x7fffffbf,
	0x7fffff7f,
	0x7ffffeff,
	0x7ffffdff,
	0x7ffffbff,
	0x7ffff7ff,
	0x7fffefff,
	0x7fffdfff,
	0x7fffbfff,
	0x7fff7fff,
	0x7ffeffff,
	0x7ffdffff,
	0x7ffbffff,
	0x7ff7ffff,
	0x7fefffff,
	0x7fdfffff,
	0x7fbfffff,
	0x7f7fffff,
	0x7effffff,
	0x7dffffff,
	0x7bffffff,
	0x77ffffff,
	0x6fffffff,
	0x5fffffff,
	0x3fffffff,
}

// maxInt returns the maximum of inputs (a, b)
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
