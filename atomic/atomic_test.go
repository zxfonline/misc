package atomic

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInt32(t *testing.T) {
	atom := NewInt32(42)

	require.Equal(t, int32(42), atom.Load(), "Load didn't work.")
	require.Equal(t, int32(46), atom.Add(4), "Add didn't work.")
	require.Equal(t, int32(44), atom.Sub(2), "Sub didn't work.")
	require.Equal(t, int32(45), atom.Inc(), "Inc didn't work.")
	require.Equal(t, int32(44), atom.Dec(), "Dec didn't work.")

	require.True(t, atom.CAS(44, 0), "CAS didn't report a swap.")
	require.Equal(t, int32(0), atom.Load(), "CAS didn't set the correct value.")

	require.Equal(t, int32(0), atom.Swap(1), "Swap didn't return the old value.")
	require.Equal(t, int32(1), atom.Load(), "Swap didn't set the correct value.")

	atom.Store(42)
	require.Equal(t, int32(42), atom.Load(), "Store didn't set the correct value.")
}

func TestInt64(t *testing.T) {
	atom := NewInt64(42)

	require.Equal(t, int64(42), atom.Load(), "Load didn't work.")
	require.Equal(t, int64(46), atom.Add(4), "Add didn't work.")
	require.Equal(t, int64(44), atom.Sub(2), "Sub didn't work.")
	require.Equal(t, int64(45), atom.Inc(), "Inc didn't work.")
	require.Equal(t, int64(44), atom.Dec(), "Dec didn't work.")

	require.True(t, atom.CAS(44, 0), "CAS didn't report a swap.")
	require.Equal(t, int64(0), atom.Load(), "CAS didn't set the correct value.")

	require.Equal(t, int64(0), atom.Swap(1), "Swap didn't return the old value.")
	require.Equal(t, int64(1), atom.Load(), "Swap didn't set the correct value.")

	atom.Store(42)
	require.Equal(t, int64(42), atom.Load(), "Store didn't set the correct value.")
}

func TestUint32(t *testing.T) {
	atom := NewUint32(42)

	require.Equal(t, uint32(42), atom.Load(), "Load didn't work.")
	require.Equal(t, uint32(46), atom.Add(4), "Add didn't work.")
	require.Equal(t, uint32(44), atom.Sub(2), "Sub didn't work.")
	require.Equal(t, uint32(45), atom.Inc(), "Inc didn't work.")
	require.Equal(t, uint32(44), atom.Dec(), "Dec didn't work.")

	require.True(t, atom.CAS(44, 0), "CAS didn't report a swap.")
	require.Equal(t, uint32(0), atom.Load(), "CAS didn't set the correct value.")

	require.Equal(t, uint32(0), atom.Swap(1), "Swap didn't return the old value.")
	require.Equal(t, uint32(1), atom.Load(), "Swap didn't set the correct value.")

	atom.Store(42)
	require.Equal(t, uint32(42), atom.Load(), "Store didn't set the correct value.")
}

func TestUint64(t *testing.T) {
	atom := NewUint64(42)

	require.Equal(t, uint64(42), atom.Load(), "Load didn't work.")
	require.Equal(t, uint64(46), atom.Add(4), "Add didn't work.")
	require.Equal(t, uint64(44), atom.Sub(2), "Sub didn't work.")
	require.Equal(t, uint64(45), atom.Inc(), "Inc didn't work.")
	require.Equal(t, uint64(44), atom.Dec(), "Dec didn't work.")

	require.True(t, atom.CAS(44, 0), "CAS didn't report a swap.")
	require.Equal(t, uint64(0), atom.Load(), "CAS didn't set the correct value.")

	require.Equal(t, uint64(0), atom.Swap(1), "Swap didn't return the old value.")
	require.Equal(t, uint64(1), atom.Load(), "Swap didn't set the correct value.")

	atom.Store(42)
	require.Equal(t, uint64(42), atom.Load(), "Store didn't set the correct value.")
}

func TestBool(t *testing.T) {
	atom := NewBool(false)
	require.False(t, atom.Toggle(), "Expected swap to return previous value.")
	require.True(t, atom.Load(), "Unexpected state after swap.")

	atom.Store(false)
	require.False(t, atom.Load(), "Unexpected state after store.")

	prev := atom.Swap(false)
	require.False(t, prev, "Expected Swap to return previous value.")

	prev = atom.Swap(true)
	require.False(t, prev, "Expected Swap to return previous value.")
}

func TestFloat64(t *testing.T) {
	atom := NewFloat64(4.2)

	require.Equal(t, float64(4.2), atom.Load(), "Load didn't work.")

	require.True(t, atom.CAS(4.2, 0.5), "CAS didn't report a swap.")
	require.Equal(t, float64(0.5), atom.Load(), "CAS didn't set the correct value.")
	require.False(t, atom.CAS(0.0, 1.5), "CAS reported a swap.")

	atom.Store(42.0)
	require.Equal(t, float64(42.0), atom.Load(), "Store didn't set the correct value.")
}

func TestFloat32(t *testing.T) {
	atom := NewFloat32(4.2)

	require.Equal(t, float32(4.2), atom.Load(), "Load didn't work.")

	require.True(t, atom.CAS(4.2, 0.5), "CAS didn't report a swap.")
	require.Equal(t, float32(0.5), atom.Load(), "CAS didn't set the correct value.")
	require.False(t, atom.CAS(0.0, 1.5), "CAS reported a swap.")

	atom.Store(42.0)
	require.Equal(t, float32(42.0), atom.Load(), "Store didn't set the correct value.")
}
