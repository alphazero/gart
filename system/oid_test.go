// Doost

package system

import (
	"github.com/alphazero/gart/syslib/digest"
	"testing"
)

func TestParseOid(t *testing.T) {
	mds := [][OidSize]byte{
		digest.Sum([]byte("Salaam Samad Sultan of LOVE")),
		digest.Sum([]byte("Given Order Giving Love")),
		digest.Sum([]byte("Ever Present Ever Near")),
		digest.Sum([]byte("Be Timely Be True")),
	}

	for _, md := range mds {
		expected, e := NewOid(md[:])
		if e != nil {
			t.Fatalf("%v", e)
		}
		oidstr := expected.String()
		oid, e := ParseOid(oidstr)
		if e != nil {
			t.Fatalf("%v", e)
		}
		if oid.String() != oidstr {
			t.Fatalf("have:%s - expect:%s", oid.String(), oidstr)
		}
	}
}
