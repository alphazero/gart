// Doost!

package util

func ShortenStr(s string, n int) string {
	n0 := n - 2
	if len(s) > n0 {
		return s[:n0] + ".."
	}
	return s
}
