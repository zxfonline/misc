package operate

import "testing"

func TestOperate(t *testing.T) {
	op, err := NewOperate("(-2.2+a-(l/20)I)I")
	if err != nil {
		t.Fatalf("NewOperate failed. err:%v.", err)
	}
	value, err := op.Execute([]string{"l", "60", "a", "-10"})
	if err != nil {
		t.Fatalf("Execute failed. err:%v.", err)
	}
	if value != -15 {
		t.Errorf("Execute failed. Got %v, expected -15.", value)
	}
	value, err = op.Execute([]string{"l", "8e2", "a", "0"})
	if err != nil {
		t.Fatalf("Execute failed. err:%v.", err)
	}
	if value != -42 {
		t.Errorf("Execute failed. Got %v, expected -42.", value)
	}
}
func TestOperate2(t *testing.T) {
	op, err := NewOperate("a^b")
	if err != nil {
		t.Fatalf("NewOperate failed. err:%v.", err)
	}

	value, err := op.Execute([]string{"a", "10", "b", "1"})
	if err != nil {
		t.Fatalf("Execute failed. err:%v.", err)
	}
	if value != 10 {
		t.Errorf("Execute failed. Got %v, expected 10.", value)
	}

	value, err = op.Execute([]string{"a", "10", "b", "2"})
	if err != nil {
		t.Fatalf("Execute failed. err:%v.", err)
	}
	if value != 100 {
		t.Errorf("Execute failed. Got %v, expected 100.", value)
	}

	value, err = op.Execute([]string{"a", "100", "b", "2"})
	if err != nil {
		t.Fatalf("Execute failed. err:%v.", err)
	}
	if value != 10000 {
		t.Errorf("Execute failed. Got %v, expected 10000.", value)
	}
}

func TestOperate3(t *testing.T) {
	op, err := NewOperate("a√b")
	if err != nil {
		t.Fatalf("NewOperate failed. err:%v.", err)
	}

	value, err := op.Execute([]string{"a", "100", "b", "1"})
	if err != nil {
		t.Fatalf("Execute failed. err:%v.", err)
	}
	if value != 100 {
		t.Errorf("Execute failed. Got %v, expected 100.", value)
	}

	value, err = op.Execute([]string{"a", "100", "b", "2"})
	if err != nil {
		t.Fatalf("Execute failed. err:%v.", err)
	}
	if value != 10 {
		t.Errorf("Execute failed. Got %v, expected 10.", value)
	}

	value, err = op.Execute([]string{"a", "10000", "b", "2"})
	if err != nil {
		t.Fatalf("Execute failed. err:%v.", err)
	}
	if value != 100 {
		t.Errorf("Execute failed. Got %v, expected 100.", value)
	}
}
func Benchmark_Execute(b *testing.B) {
	op, err := NewOperate("(-2.2-(l/2E1))I")
	for d := 0; d < b.N; d++ {
		if err != nil {
			b.Fatalf("NewOperate failed. err:%v.", err)
		}
		value, err := op.Execute([]string{"l", "60"})
		if err != nil {
			b.Fatalf("Execute failed. err:%v.", err)
		}
		if value != -5 {
			b.Errorf("Execute failed. Got %v, expected -5.", value)
		}
		value, err = op.Execute([]string{"l", "8e2"})
		if err != nil {
			b.Fatalf("Execute failed. err:%v.", err)
		}
		if value != -42 {
			b.Errorf("Execute failed. Got %v, expected -42.", value)
		}
	}
}

func Benchmark_Execute2(b *testing.B) {
	op, err := NewOperate("a^b")
	for d := 0; d < b.N; d++ {
		if err != nil {
			b.Fatalf("NewOperate failed. err:%v.", err)
		}

		value, err := op.Execute([]string{"a", "10", "b", "1"})
		if err != nil {
			b.Fatalf("Execute failed. err:%v.", err)
		}
		if value != 10 {
			b.Errorf("Execute failed. Got %v, expected 10.", value)
		}

		value, err = op.Execute([]string{"a", "10", "b", "2"})
		if err != nil {
			b.Fatalf("Execute failed. err:%v.", err)
		}
		if value != 100 {
			b.Errorf("Execute failed. Got %v, expected 100.", value)
		}

		value, err = op.Execute([]string{"a", "100", "b", "2"})
		if err != nil {
			b.Fatalf("Execute failed. err:%v.", err)
		}
		if value != 10000 {
			b.Errorf("Execute failed. Got %v, expected 10000.", value)
		}
	}
}
func Benchmark_Execute3(b *testing.B) {
	op, err := NewOperate("a√b")
	for d := 0; d < b.N; d++ {
		if err != nil {
			b.Fatalf("NewOperate failed. err:%v.", err)
		}

		value, err := op.Execute([]string{"a", "100", "b", "1"})
		if err != nil {
			b.Fatalf("Execute failed. err:%v.", err)
		}
		if value != 100 {
			b.Errorf("Execute failed. Got %v, expected 100.", value)
		}

		value, err = op.Execute([]string{"a", "100", "b", "2"})
		if err != nil {
			b.Fatalf("Execute failed. err:%v.", err)
		}
		if value != 10 {
			b.Errorf("Execute failed. Got %v, expected 10.", value)
		}

		value, err = op.Execute([]string{"a", "10000", "b", "2"})
		if err != nil {
			b.Fatalf("Execute failed. err:%v.", err)
		}
		if value != 100 {
			b.Errorf("Execute failed. Got %v, expected 100.", value)
		}
	}
}
