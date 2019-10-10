package keyword

import (
	"strings"
)

const (
	DEFAULT_BASE_SIZE = 2

	CHARS_COUNT = 27
)

var _G *KeyWord

func init() {
	_G = NewKeyWord()
}

//关键字查询
func Search(str string) bool {
	return _G.Search(str)
}

//设置全局关键字
func SetGlobal(key *KeyWord) {
	_G = key
}

type KeyWord struct {
	SHIFT    map[string]*Shift   // key:字符块
	KEYS     []string            //所有匹配串
	PATTERNS map[string]*Pattern // key:匹配串  value:对应的pattern列表
	PREFIX   []*Prefix           //所有前缀

	KeySize    int //匹配串长度
	PrefixSize int //前缀块
	BaseSize   int //字符块长
}

type Shift struct {
	step      int
	preOffset int //PREFIX key
	preCount  int
}

type Prefix struct {
	prefix    string
	keyOffset int
	keyCount  int
}

type Pattern struct {
	list []string
}

func newPattern(value string) *Pattern {
	t := &Pattern{}
	t.list = append(t.list, value)

	return t
}

//初始化临时结构
type tmpShift struct {
	step    int
	prefixs map[string]*tmpPatterns
}

func newTmpShift() *tmpShift {
	t := &tmpShift{}
	t.prefixs = make(map[string]*tmpPatterns)
	return t
}

type tmpPatterns struct {
	tps []string
}

func newTmpPatterns(shift string) *tmpPatterns {
	t := &tmpPatterns{}
	t.tps = append(t.tps, shift)
	return t
}

func NewKeyWord() *KeyWord {
	w := &KeyWord{}
	w.SHIFT = make(map[string]*Shift)
	w.PATTERNS = make(map[string]*Pattern)
	return w
}
func (vk *KeyWord) Init(strs []string) {
	for _, str := range strs {
		if vk.KeySize == 0 || vk.KeySize > len(str) {
			vk.KeySize = len(str)
		}
	}
	//vk.BaseSize =  (2 *length * vk.KeySize)
	vk.BaseSize = 2
	vk.PrefixSize = 2

	tmpSHIFT := make(map[string]*tmpShift)

	shiftOff := vk.KeySize - vk.BaseSize + 1
	tmpStrMap := make(map[string]bool) //用于字符串去重, key:str
	for _, str := range strs {
		if _, exist := tmpStrMap[str]; exist { //处理过的不再处理
			continue
		} else {
			tmpStrMap[str] = true
		}

		key := str[0:vk.KeySize] //这个str的前缀
		//构造PATTERNS表
		if tp, exist := vk.PATTERNS[key]; exist { //这个前缀已经处理过就不需要再处理
			tp.list = append(tp.list, str)
			continue
		} else {
			tp = newPattern(str)
			vk.PATTERNS[key] = tp
		}
		//开始处理key
		for i := 0; i < shiftOff; i++ {
			var sf *tmpShift
			var exist bool

			s := str[i : i+2] //基础字符块
			newStep := vk.KeySize - i - vk.BaseSize
			//填充shift表
			if sf, exist = tmpSHIFT[s]; exist {
				if sf.step > newStep {
					sf.step = newStep
					tmpSHIFT[s] = sf
				}
			} else {
				sf = newTmpShift()
				sf.step = newStep
				tmpSHIFT[s] = sf
			}
			//如果step为0 要填充prefix
			if sf.step == 0 && newStep == 0 {
				pre := str[0:vk.PrefixSize]               //shfit的前缀
				if pts, exist := sf.prefixs[pre]; exist { //获取到这个前缀的shift列表
					pts.tps = append(pts.tps, key)
				} else {
					pts = newTmpPatterns(key) //shift map
					sf.prefixs[pre] = pts
				}
			}
		}
	}

	//构造shift
	preStart := 0
	keyStart := 0
	for key, ts := range tmpSHIFT {
		s := &Shift{step: ts.step, preOffset: preStart, preCount: len(ts.prefixs)}
		vk.SHIFT[key] = s
		for pre, pts := range ts.prefixs {
			p := &Prefix{prefix: pre, keyOffset: keyStart, keyCount: len(pts.tps)}
			vk.PREFIX = append(vk.PREFIX, p)
			preStart++
			for _, tp := range pts.tps {
				vk.KEYS = append(vk.KEYS, tp)
				keyStart++
			}
		}
	}
}

func (vk *KeyWord) Search(str string) bool {
	length := len(str)
	if length < vk.KeySize {
		return false
	}
	end := length - vk.BaseSize + 1
	for i := vk.KeySize - vk.BaseSize; i < end; {
		base := str[i : i+vk.BaseSize]
		var s *Shift
		var exist bool
		if s, exist = vk.SHIFT[base]; exist {
			if s.step > 0 {
				i += s.step
				continue
			}
		} else {
			i += vk.KeySize - vk.BaseSize + 1
			continue
		}
		// 后缀验证完成，开始prefix验证
		preOffset := s.preOffset
		keyStart := (i - vk.KeySize + vk.BaseSize)
		prefix := str[keyStart : keyStart+vk.BaseSize]
		key := str[keyStart:(keyStart + vk.KeySize)]
		for preCount := 0; preCount < s.preCount; preCount++ {
			pre := vk.PREFIX[preOffset]
			if pre.prefix != prefix { //前缀不匹配
				preOffset++
				continue
			}
			// prefix验证完成，开始shift验证
			for keyCount := 0; keyCount < pre.keyCount; keyCount++ {
				matchKey := vk.KEYS[pre.keyOffset+keyCount]
				if key == matchKey {
					//shift验证完成，开始完整验证
					if vk.match(key, str[keyStart:]) == true {
						return true
					}
				}
			}
			preOffset++
		}
		i++
	}

	return false
}

// 根据前缀@shift对value进行完整匹配
func (vk *KeyWord) match(shift string, value string) bool {
	if pt, exist := vk.PATTERNS[shift]; exist {
		for _, p := range pt.list { //依次比较
			if strings.HasPrefix(value, p) {
				return true
			}
		}
	}
	return false
}
