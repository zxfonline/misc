package random

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	//randPool    = make(chan func() (n int, input chan<- int))
	_globalRand *rand.Rand
	_randLock   sync.Mutex
)

func init() {
	rand.Seed(time.Now().UnixNano())
	_globalRand = rand.New(rand.NewSource(rand.Int63()))
	//go func() {
	//	for {
	//		randFunc := <-randPool
	//		n, output := randFunc()
	//		output <- _globalRand.Intn(n)
	//	}
	//}()
}

// Intn returns, as an int, a non-negative pseudo-random number in [0,n).
// It panics if n <= 0.
func Intn(n int) int {
	//output := make(chan int, 1)
	//randPool <- func() (int, chan<- int) {
	//	return n, output
	//}
	//return <-output
	_randLock.Lock()
	defer _randLock.Unlock()
	return _globalRand.Intn(n)
}

//------------------------

//func main() {
//	fmt.Println(GetRandomNumber("1", 2))
//	fmt.Println(GetRandomNumber("1~10", 2))
//	fmt.Println(GetRandomNumber("1~10,44~89,2~5", 2))
//	fmt.Println(GetRandomNumber("1:20,1~4:30,4:500", 2))
//	fmt.Println(GetRandomNumber("1,2,4", 2))
//	fmt.Println(GetRandomNumber("2~10:40", 2))
//	fmt.Println(GetRandomNumber("1:40", 2))
//	fmt.Println(GetRandomNumber("1:20,1~4:30,4:500", 2))
//	fmt.Println(GetRandomNumbers("2~10:40#10:20,10~45:30,40~80:500"))
//}

/**
 * 从给定的一些数值里随机抽取N个数
 *
 * @param numbers,给定的等待抽取的列表
 * @param n 要随机多n个数
 * @return
 */
func GetRandomValues(numbers []int, n int) []int {
	size := len(numbers)
	filter := make([]int, size)
	copy(filter, numbers)
	if size == 0 || n >= size {
		return filter
	}
	list := make([]int, 0, n)
	for i := 0; i < n; i++ {
		index := Intn(len(filter))
		list = append(list, filter[index])
		filter = append(filter[:index], filter[index+1:]...)
	}
	return list
}

/**
 * 从给定的一些数值里随机抽取N个数
 *
 * @param numbers,给定的等待抽取的列表
 * @param n 要随机多n个数
 * @return
 */
func GetRandomValuesInt64(numbers []int64, n int) []int64 {
	size := len(numbers)
	filter := make([]int64, size)
	copy(filter, numbers)
	if size == 0 || n >= size {
		return filter
	}
	list := make([]int64, 0, n)
	for i := 0; i < n; i++ {
		index := Intn(len(filter))
		list = append(list, filter[index])
		filter = append(filter[:index], filter[index+1:]...)
	}
	return list
}

/**
 * 加权随机数 数值抽取器
 *
 * @param args 以“#”劈分数组然后再在数组元素中每一位获取一个随机数
 * @param radom
 * @see #GetRandomNumber(String, Random)
 * @return
 */
func GetRandomNumbers(args string) []int {
	strs := strings.Split(args, "#")
	size := len(strs)
	ints := make([]int, 0, size)
	for i := 0; i < size; i++ {
		if len(strs[i]) > 0 {
			ints = append(ints, GetRandomNumber(strs[i], 1)[0])
		}
	}
	return ints
}

/**
 * 数值抽取器(从枚举值,范围随机值,定值 中随机抽出一个值)
 * 枚举值(支持单个出现概率)：1,2,4 或 1:10,2:30,4:30
 * 范围值： 1~10
 * 定值：12
 * 支持混合使用 如：1~10,44~89,2~5 又如 2~10:40
 * 支持概率后缀 代表该值被抽取出来的几率 值越高被抽出的概率越大 并不限定后缀值的范围 如：1:20,1~4:30,4:500
 *
 * @param args
 * @param n 随机个数
 * @param radom
 * @return
 */
func GetRandomNumber(args string, n int) []int {
	var err error
	//	if strings.Index(args, ",") > 0 {
	values := strings.Split(args, ",")
	size := len(values)
	if n > size {
		n = size
	}
	numbers := make([]int, size)
	if strings.Index(args, ":") > 0 { // 1:20,1~4:30,4:500
		weights := make([]int, size)
		var valuesStr, weightStr string
		var weightSum int
		var value int
		for i := 0; i < size; i++ {
			if endIndex := strings.Index(values[i], ":"); endIndex <= 0 {
				panic(fmt.Errorf("invalid args:%v,err:%v", args, values[i]))
			} else {
				valuesStr = string(values[i][:endIndex])
			}
			weightStr = string(values[i][strings.Index(values[i], ":")+1:])
			if value, err = strconv.Atoi(weightStr); err != nil {
				panic(fmt.Errorf("invalid args:%v,err:%v", args, err))
			} else {
				weights[i] = value
				weightSum += value
			}
			if numbers[i], err = average(valuesStr); err != nil {
				panic(fmt.Errorf("invalid args:%v,err:%v", args, err))
			}
		}
		// 随机多个
		rd := make([]int, 0, n)
		for j := 0; j < n; j++ {
			ranNum := Intn(weightSum)
			for i := 0; i < size; i++ {
				ranNum -= weights[i]
				if ranNum < 0 {
					rd = append(rd, numbers[i])
					size--
					weightSum -= weights[i]
					weights = append(weights[:i], weights[i+1:]...)
					numbers = append(numbers[:i], numbers[i+1:]...)
					break
				}
			}
		}
		return rd
	} else { // 1~10,44~89,2~5
		for i := 0; i < size; i++ {
			if numbers[i], err = average(values[i]); err != nil {
				panic(fmt.Errorf("invalid args:%v,err:%v", args, err))
			}
		}
		if n == 1 {
			return []int{numbers[Intn(size)]}
		} else {
			return GetRandomValues(numbers, n)
		}
	}
}

func average(args string) (int, error) { // 1~10
	if strings.Index(args, "~") > 0 {
		tmp := strings.Split(args, "~")
		if v1, err1 := strconv.Atoi(tmp[0]); err1 != nil {
			return 0, err1
		} else if v2, err2 := strconv.Atoi(tmp[1]); err2 != nil {
			return 0, err2
		} else {
			return v1 + Intn(v2+1-v1), nil
		}
	} else {
		return strconv.Atoi(args)
	}
}

type RandItem struct {
	ItemID  int32 `json:"itemId"`  //奖励物品ID
	Num     int32 `json:"num"`     //数量
	TimeOut int32 `json:"timeout"` //过期时间
	Weight  int32 `json:"weight"`  //权重
}

func GetRandomItems(items []RandItem, n int) []RandItem {
	size := len(items)
	if n > size {
		n = size
	}
	var weightSum int
	for _, item := range items {
		weightSum += int(item.Weight)
	}
	rd := make([]RandItem, 0, n)
	for j := 0; j < n; j++ {
		ranNum := Intn(weightSum)
		for i := 0; i < size; i++ {
			ranNum -= int(items[i].Weight)
			if ranNum < 0 {
				rd = append(rd, items[i])
				size--
				weightSum -= int(items[i].Weight)
				items = append(items[:i], items[i+1:]...)
				break
			}
		}
	}
	return rd

}

//RandInterface 随机数选取器接口
type RandInterface interface {
	Len() int
	Weight(i int) int
	SubValue(indexs []int) interface{}
}

//GetRandomWeight 随机数选取
func GetRandomWeight(data RandInterface, n int) interface{} {
	size := data.Len()
	if n > size {
		n = size
	}
	var weightSum int
	for j := 0; j < size; j++ {
		weightSum += data.Weight(j)
	}

	var indexs []int

	for j := 0; j < n; j++ {
		ranNum := Intn(weightSum)
		for i := 0; i < size; i++ {
			find := false
			for _, index := range indexs {
				if index == i {
					find = true
					break
				}
			}

			if find {
				continue
			}

			ranNum -= data.Weight(i)
			if ranNum < 0 {
				indexs = append(indexs, i)
				weightSum -= data.Weight(i)
				break
			}
		}
	}

	return data.SubValue(indexs)
}
