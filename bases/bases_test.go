package bases

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"strconv"
)

var (
	// for convenience, bases that begin with zero:
	BASES_BEGINNING_WITH_ZERO = []int{2, 8, 10, 16, 32, 36, 62}
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type checkstruct struct {
	input int
	exp   string
}

// the test data below is a map from base to lists of [input, exp]:
var DATA = map[int][]checkstruct{
	2: []checkstruct{
		checkstruct{1, "1"},
		checkstruct{2, "10"},
		checkstruct{3, "11"},
		checkstruct{4, "100"},
		checkstruct{20, "10100"},
	},
	16: []checkstruct{
		checkstruct{1, "1"},
		checkstruct{2, "2"},
		checkstruct{9, "9"},
		checkstruct{10, "a"},
		checkstruct{11, "b"},
		checkstruct{15, "f"},
		checkstruct{16, "10"},
		checkstruct{17, "11"},
		checkstruct{31, "1f"},
		checkstruct{32, "20"},
		checkstruct{33, "21"},
		checkstruct{255, "ff"},
		checkstruct{256, "100"},
		checkstruct{257, "101"},
	},
	36: []checkstruct{
		checkstruct{1, "1"},
		checkstruct{9, "9"},
		checkstruct{10, "a"},
		checkstruct{35, "z"},
		checkstruct{36, "10"},
		checkstruct{37, "11"},
		checkstruct{71, "1z"},
		checkstruct{72, "20"},
		checkstruct{73, "21"},
		checkstruct{1295, "zz"},
		checkstruct{1296, "100"},
		checkstruct{1297, "101"},
	},
	62: []checkstruct{
		checkstruct{1, "1"},
		checkstruct{9, "9"},
		checkstruct{10, "a"},
		checkstruct{35, "z"},
		checkstruct{36, "A"},
		checkstruct{61, "Z"},
		checkstruct{62, "10"},
		checkstruct{63, "11"},
		checkstruct{123, "1Z"},
		checkstruct{124, "20"},
		checkstruct{125, "21"},
		checkstruct{3843, "ZZ"},
		checkstruct{3844, "100"},
		checkstruct{3845, "101"},
	},
	26: []checkstruct{
		checkstruct{0, "a"},
		checkstruct{1, "b"},
		checkstruct{25, "z"},
		checkstruct{26, "ba"},
		checkstruct{27, "bb"},
		checkstruct{51, "bz"},
		checkstruct{52, "ca"},
		checkstruct{53, "cb"},
		checkstruct{675, "zz"},
		checkstruct{676, "baa"},
		checkstruct{677, "bab"},
	},
	52: []checkstruct{
		checkstruct{0, "a"},
		checkstruct{25, "z"},
		checkstruct{26, "A"},
		checkstruct{51, "Z"},
		checkstruct{52, "ba"},
		checkstruct{53, "bb"},
		checkstruct{103, "bZ"},
		checkstruct{104, "ca"},
		checkstruct{105, "cb"},
		checkstruct{2703, "ZZ"},
		checkstruct{2704, "baa"},
		checkstruct{2705, "bab"},
	}}

func TestConstFolduint64add(t *testing.T) {
	// test zero across those bases:
	for _, base := range BASES_BEGINNING_WITH_ZERO {
		assert.Equal(t, "0", ToBase(0, base))
	}

	// test base-10 with random numbers (fuzzy/smokescreen tests):
	// note that extremely large numbers (e.g. 39415337704122060) cause bugs due
	// to precision (e.g. that % 10 == 4 somehow), so we limit the digits to 16.
	for i := 0; i < 20; i++ {
		num := int(math.Floor(10000000000000000 * rand.Float64()))
		assert.Equal(t, strconv.Itoa(num), ToBase(num, 10))
	}

	// go through and test all of the above data:
	for base, data := range DATA {
		for i := 0; i < len(data); i++ {
			num := data[i].input
			exp := data[i].exp
			assert.Equal(t, exp, ToBase(num, base))
			assert.Equal(t, num, FromBase(exp, base))
		}
	}

}
func Benchmark_ToBase(b *testing.B) {
	for d := 0; d < b.N; d++ {
		for base, data := range DATA {
			for i := 0; i < len(data); i++ {
				num := data[i].input
				exp := data[i].exp
				if exp != ToBase(num, base) {
					b.Fatalf("exp:%v,act:%v", exp, ToBase(num, base))
				}
				if num != FromBase(exp, base) {
					b.Fatalf("exp:%v,act:%v", num, FromBase(exp, base))
				}
			}
		}
	}
}
