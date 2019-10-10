/**
 * 四则混合运算类
 * *初始化的时候需要传入公式：
 * *只支持加减乘除和括号
 *
 * *execute计算结算方法
 * *参数一个变量和数值的字符串数组
 * *AVAILABLE_PARAMETER_CODE（变量定义，只支持单字节变量）
 * *例：公式a+b*4
 * *参数String[]{"a","1","b","2"}
 * *计算结果9
 *
 */
package operate

import (
	"errors"
	"math"
	"strconv"
	"strings"

	. "github.com/zxfonline/misc/operate/stack"
)

const (
	//四则运算符号
	AVAILABLE_CODE = "+-*/()^√"
	//小数符号
	AVAILABLE_DECIMAL_CODE = "1234567890.E"
	//参数符配置
	AVAILABLE_PARAMETER_CODE = "abcdefhijklmn"
)

/** 四则运算类 */
type Operate struct {
	//文字表达式
	sentence string
	//表达式存储
	opers []string
	//后缀表达式
	suffixExpression []string
}

func NewOperate(sentence string) (o *Operate, err error) {
	o = &Operate{
		opers:            make([]string, 0, len(sentence)),
		suffixExpression: make([]string, 0, len(sentence)),
		sentence:         sentence,
	}
	o.init()
	err = o.setSuffixExpression()
	return
}

//将输入的字符串分割,存放到opers中
func (this *Operate) init() {
	value := make([]rune, 0, len(this.sentence))
	flag := false
	for _, c := range this.sentence {
		if isBelongToDecimal(string(c)) {
			value = append(value, c)
			flag = true
			if c == rune('E') {
				flag = false
			}
		} else if (c == rune('-') || c == rune('+')) && !flag {
			value = append(value, c)
			flag = true
		} else if isAvailableCode(string(c)) {
			if flag && len(value) > 0 {
				this.opers = append(this.opers, string(value))
				flag = false
			}
			value = value[:0]
			this.opers = append(this.opers, string(c))
		} else if isParameterCode(string(c)) {
			if flag && len(value) > 0 {
				this.opers = append(this.opers, string(value))
			}
			value = value[:0]
			this.opers = append(this.opers, string(c))
			flag = true
		} else if c == rune('I') {
			if flag && len(value) > 0 {
				this.opers = append(this.opers, string(value))
			}
			value = value[:0]
			this.opers = append(this.opers, string(c))
			flag = false
		}
	}
	if flag && len(value) > 0 {
		this.opers = append(this.opers, string(value))
	}
}

//将opers中存放的表达式转化成后缀形式,存放至suffixExpression中
func (this *Operate) setSuffixExpression() error {
	op := NewStack()
	var top interface{}
	for i := 0; i < len(this.opers); i++ {
		current := this.opers[i]
		if current == "(" {
			op.Push(current)
		} else if current == ")" {
			for {
				top = op.Pop()
				if top == "(" {
					break
				}
				this.suffixExpression = append(this.suffixExpression, M2string(top))
			}
		} else if current == "+" || current == "-" || current == "*" || current == "/" || current == "^" || current == "√" {
			for {
				if op.Empty() || op.Peak() == "(" || ((current == "*" || current == "/") && (op.Peak() == "+" || op.Peak() == "-")) || ((current == "^" || current == "√") && (op.Peak() == "*" || op.Peak() == "/" || op.Peak() == "+" || op.Peak() == "-")) {
					op.Push(current)
					break
				} else {
					top = op.Pop()
					this.suffixExpression = append(this.suffixExpression, M2string(top))
				}
			}
		} else {
			this.suffixExpression = append(this.suffixExpression, M2string(current))
		}
	}
	for {
		if op.Empty() {
			break
		}
		top = op.Pop()
		if top != "(" {
			this.suffixExpression = append(this.suffixExpression, M2string(top))
		} else {
			return errors.New("invalid operate sentence")
		}
	}
	return nil
}

//计算后缀表达式的值
func (this *Operate) Execute(str []string) (value float64, err error) {
	defer PanicToErr(&err)
	temp := NewStack()
	for i := 0; i < len(this.suffixExpression); i++ {
		st := changeParameter(this.suffixExpression[i], str)
		if val, err := strconv.ParseFloat(strings.TrimSpace(st), 64); err == nil {
			temp.Push(val)
		} else {
			exp := this.suffixExpression[i]
			if exp == "I" {
				v1 := temp.Pop()
				temp.Push(M2int64(v1))
			} else {
				rights := temp.Pop()
				right := M2float64(rights)
				lefts := temp.Pop()
				left := M2float64(lefts)
				temp.Push(executeSingleExpression(left, right, exp))
			}
		}
	}
	value = M2float64(temp.Pop())
	return
}

//参数替换
func changeParameter(parameter string, str []string) string {
	for i := 0; i < len(str); i += 2 {
		if parameter == str[i] {
			return str[i+1]
		}
	}
	return parameter
}

//计算
func executeSingleExpression(left, right float64, exp string) float64 {
	if exp == "+" {
		return left + right
	} else if exp == "-" {
		return left - right
	} else if exp == "*" {
		return left * right
	} else if exp == "/" {
		return left / right
	} else if exp == "^" {
		return math.Pow(left, right)
	} else if exp == "√" {
		return math.Pow(left, 1/right)
	}
	return 0.0
}

//判断计算符
func isAvailableCode(c string) bool {
	return strings.IndexAny(AVAILABLE_CODE, c) != -1
}

//判断小数
func isBelongToDecimal(c string) bool {
	return strings.IndexAny(AVAILABLE_DECIMAL_CODE, c) != -1
}

//判断参数符号
func isParameterCode(c string) bool {
	return strings.IndexAny(AVAILABLE_PARAMETER_CODE, c) != -1
}
