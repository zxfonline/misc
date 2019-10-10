package random

const (
	LCG_A uint32 = 1664525
	LCG_C uint32 = 1013904223
)

type LCGRand struct {
	seed uint32
	skip uint32
	x    uint32
}

// 范围[0,n)
func (lcg *LCGRand) RandN(n uint32) uint32 {
	if n == 0 {
		return 0
	}

	return lcg.NextRand() % n
}

func (lcg *LCGRand) RandFloat64() float64 {
	f := lcg.NextRand()
	return float64(f) / float64(1<<32)
}

func (lcg *LCGRand) NextRand() uint32 {
	lcg.x = LCG_A*lcg.x + LCG_C
	lcg.skip++
	return lcg.x
}

func NewLCGRand(seed uint32) *LCGRand {
	return &LCGRand{seed, 0, seed}
}
