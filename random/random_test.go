package random

import (
	"fmt"
	//	"math/rand"
	"sync"
	"testing"
	//	"time"
)

func Test_RandInt(t *testing.T) {
	//	rand.Seed(time.Now().UTC().UnixNano())
	wc := make(chan int, 20000)
	fmt.Println("---=", len(wc), cap(wc))
	v := RandInt(0, 0)
	if v != 0 {
		t.Fatal("v must be 0")
	}
	//	fmt.Println(v)
	v = RandInt(-10, -20) //-10,-9
	if v != -10 {
		t.Fatal("v must be -10")
	}
	//	fmt.Println(v)
	for k := 0; k < 100; k++ {
		go func() {
			for i := 0; i < 100; i++ {
				v = RandInt(-10, 20)
				if v < -10 || v >= 20 {
					t.Fatal("v must be [-10,20)")
				}
				//				fmt.Println(v)
				wc <- 1
			}
		}()
	}
	for k := 0; k < 100; k++ {
		go func() {
			for i := 0; i < 100; i++ {
				v = RandInt(-100, -10)
				if v < -100 || v >= -10 {
					t.Fatal("v must be [-100,-10)")
				}
				//				fmt.Println(v)
				wc <- 1
			}
		}()
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		num := 0
		for q := false; !q; {
			select {
			case <-wc:
				num++
				if num == cap(wc) {
					q = true
					fmt.Println("num=", num)
				}
			default:
			}
		}
		fmt.Println("done")
		wg.Done()
	}()
	wg.Wait()
}
