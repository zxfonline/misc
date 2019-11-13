package timefix

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var (
	//TimeFormate 时间格式
	TimeFormate string = "2006-01-02 15:04:05"

	SIMPLE_DAY_FORMAT string = "2006-01-02"
	//TimeFix 时差修正 纳秒差值
	TimeFix int64
	//Server_Location 服务器使用的标准时区
	Server_Location = time.Local
)

func init() {
	time.Local = Server_Location
	if local, err := time.LoadLocation("Asia/Shanghai"); err == nil && local != nil {
		Server_Location = local
		time.Local = Server_Location
		fmt.Println("Location:", Server_Location, ",serverTime:", CurrentTime().Format(TimeFormate))
	} else {
		fmt.Println("load server location[Asia/Shanghai] error:", err)
	}
}

const (
	//MILLISECONDS_OF_DAY 一天的毫秒数
	MILLISECONDS_OF_DAY = 24 * MILLISECONDS_OF_HOUR
	//MILLISECONDS_OF_HOUR 一小时的毫秒数
	MILLISECONDS_OF_HOUR = 60 * MILLISECONDS_OF_MINUTE
	//MILLISECONDS_OF_MINUTE 一分钟的毫秒数
	MILLISECONDS_OF_MINUTE = 60 * MILLISECONDS_OF_SECOND
	//MILLISECONDS_OF_SECOND 一秒的毫秒数
	MILLISECONDS_OF_SECOND = 1000
)

//ResetTime 重置时间，传入utc标准时间UnixNano()
func ResetTime(t int64) {
	TimeFix = time.Now().In(Server_Location).UnixNano() - t
}

//NanosTime 当前本地时间 纳秒 已修正
func NanosTime() int64 {
	return time.Now().In(Server_Location).UnixNano() - TimeFix
}

//CurrentTime 服务器当前时间
func CurrentTime() time.Time {
	return Nanos2Time(NanosTime())
}

//SecondTime 当前本地时间 秒 已修正
func SecondTime() int64 {
	return NanosTime() / int64(time.Second)
}

//MillisTime 当前本地时间 毫秒 已修正
func MillisTime() int64 {
	return NanosTime() / int64(time.Millisecond)
}

//CurrentMS 当前本地时间 毫秒 已修正
func CurrentMS() int64 {
	return NanosTime() / int64(time.Millisecond)
}

//TimeMillis 时间转成毫秒
func TimeMillis(t time.Duration) int64 {
	return t.Nanoseconds() / int64(time.Millisecond)
}

//TimeSecond 时间转成秒
func TimeSecond(t time.Duration) int64 {
	return t.Nanoseconds() / int64(time.Second)
}

//Nanos2Time 纳秒时间转成服务器时间(根据传入的时间的时区在返回的时间上进行时区纠正time.Time().In(tm.Location()))
func Nanos2Time(ns int64) time.Time {
	return time.Unix(0, ns).In(Server_Location)
}

//Second2Time 秒时间转成服务器时间(根据传入的时间的时区在返回的时间上进行时区纠正time.Time().In(tm.Location()))
func Second2Time(second int64) time.Time {
	return time.Unix(second, 0).In(Server_Location)
}

//Time2Midnight 获得指定时间的凌晨时间
func Time2Midnight(tm time.Time) time.Time {
	year, month, day := tm.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, tm.Location())
}

//NextMidnight 获取一个时间点的 x日后的凌晨时间
func NextMidnight(tm time.Time, day int) time.Time {
	midTime := Time2Midnight(tm)
	ns := midTime.UnixNano() + int64(day*MILLISECONDS_OF_DAY)*int64(time.Millisecond)
	return Nanos2Time(ns).In(tm.Location())
}

//NextHour 从一个时间戳获取下一个准点时间
func NextHour(tm time.Time) time.Time {
	year, month, day := tm.Date()
	hour, _, _ := tm.Clock()
	return time.Date(year, month, day, hour+1, 0, 0, 0, tm.Location())
}

//NextHours 从一个时间戳获取下一个准点时间
func NextHours(tm time.Time, n int) time.Time {
	year, month, day := tm.Date()
	hour, _, _ := tm.Clock()
	return time.Date(year, month, day, hour+n, 0, 0, 0, tm.Location())
}

//DeltaDays 获取两个时间跨了多少天 >=0
func DeltaDays(unix1, unix2 int64) int64 {
	t1 := Second2Time(unix1)
	t2 := Second2Time(unix2)
	t1 = Time2Midnight(t1)
	t2 = Time2Midnight(t2)
	days := int64(t2.Sub(t1)/time.Millisecond) / MILLISECONDS_OF_DAY
	if days < 0 {
		days = -days
	}
	return days
}

//S2UnixTime date format: "2006-01-02 15:04:05"
func S2UnixTime(value string, loc *time.Location) (*time.Time, error) {
	re := regexp.MustCompile(`([\d]+)-([\d]+)-([\d]+) ([\d]+):([\d]+):([\d]+)`)
	slices := re.FindStringSubmatch(value)
	if slices == nil || len(slices) != 7 {
		return nil, fmt.Errorf("time[%s] format error, expect format: 2006-01-02 15:04:05 \n", value)
	}
	year, _ := strconv.Atoi(slices[1])
	month, _ := strconv.Atoi(slices[2])
	day, _ := strconv.Atoi(slices[3])
	hour, _ := strconv.Atoi(slices[4])
	min, _ := strconv.Atoi(slices[5])
	sec, _ := strconv.Atoi(slices[6])
	tt := time.Date(year, time.Month(month), day, hour, min, sec, 0, loc)
	return &tt, nil
}

//NextSundayMS 获取指定时间的下一个周末时间,自己处理好时区的问题,周一凌晨作为跨周，如果传入的时间没有超过周一凌晨，则返回周一凌晨的时间点，否则返回下一周的周一凌晨
func NextSundayMS(tm time.Time) time.Time {
	mt := Time2Midnight(tm)
	weekday := tm.Weekday()
	if weekday == time.Sunday {
		return Nanos2Time(mt.UnixNano()).In(tm.Location()).Add(24 * time.Hour)
	}
	return Nanos2Time(mt.UnixNano() + int64((7-int64(weekday))*MILLISECONDS_OF_DAY*int64(time.Millisecond))).In(tm.Location()).Add(24 * time.Hour)
}

//SharpClock 返回当前的整点时间
func SharpClock(tm time.Time) time.Time {
	year, month, day := tm.Date()
	hour, _, _ := tm.Clock()
	return time.Date(year, month, day, hour, 0, 0, 0, tm.Location())
}

//func main() {
//	t1 := S2UnixTime("2018-11-05 00:00:00", Server_Location)
//	t2 := NextSundayMS(t1.Local())
//	fmt.Println(t1.Format(TimeFormate))
//	fmt.Println(t2.Local().Format(TimeFormate))
//}

//NowToSharpClock 返回从整点到现在的差值
func NowToSharpClock(tm time.Time) time.Duration {
	return time.Duration(tm.UnixNano() - SharpClock(tm).UnixNano())
}

//CheckCrossWeek 检查是否跨周
func CheckCrossWeek(baset time.Time, now time.Time) bool {
	year1, week1 := baset.ISOWeek()
	year2, week2 := now.ISOWeek()
	if (year1 == year2 && week2-week1 > 0) || ((year1 < year2) && (year2-year1 > 1 || week2 > 1 || now.Unix() > NextSundayMS(baset).Unix())) { //跨周
		return true
	}
	return false
}

//CheckCrossDay 检查是否跨天
func CheckCrossDay(baset time.Time, now time.Time) bool {
	year1, month1, day1 := baset.Date()
	year2, month2, day2 := now.Date()
	if (year1 == year2 && ((month1 < month2) || (month1 == month2 && day2-day1 >= 1))) || (year1 < year2) { //跨天
		return true
	}
	return false
}

//CheckCrossMonth 检查是否跨月
func CheckCrossMonth(baset time.Time, now time.Time) bool {
	year1, month1, _ := baset.Date()
	year2, month2, _ := now.Date()
	if (year1 == year2 && month1 < month2) || (year1 < year2) { //跨天
		return true
	}
	return false
}

var days = [12]int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

//GetDayInMon 获取月份的天数
func GetDayInMon(year int, mon int) int {
	var day int
	if 2 == mon {
		if (year%4 == 0 && year%100 != 0) || year%400 == 0 {
			day = 29
		} else {
			day = 28
		}
	} else {
		day = days[mon-1]
	}

	return day
}

// short string format
func ShortTimeFormat(d time.Duration) string {
	u := uint64(d)
	if u < uint64(time.Second) {
		switch {
		case u == 0:
			return "0"
		case u < uint64(time.Microsecond):
			return fmt.Sprintf("%.2fns", float64(u))
		case u < uint64(time.Millisecond):
			return fmt.Sprintf("%.2fus", float64(u)/1000)
		default:
			return fmt.Sprintf("%.2fms", float64(u)/1000/1000)
		}
	} else {
		switch {
		case u < uint64(time.Minute):
			return fmt.Sprintf("%.2fs", float64(u)/1000/1000/1000)
		case u < uint64(time.Hour):
			return fmt.Sprintf("%.2fm", float64(u)/1000/1000/1000/60)
		default:
			return fmt.Sprintf("%.2fh", float64(u)/1000/1000/1000/60/60)
		}
	}
}

func AvgTime(items []time.Duration) time.Duration {
	var sum time.Duration
	for _, item := range items {
		sum += item
	}
	return time.Duration(int64(sum) / int64(len(items)))
}

// format bytes number friendly
func BytesFormat(bytes uint64) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%dB", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.2fK", float64(bytes)/1024)
	case bytes < 1024*1024*1024:
		return fmt.Sprintf("%.2fM", float64(bytes)/1024/1024)
	default:
		return fmt.Sprintf("%.2fG", float64(bytes)/1024/1024/1024)
	}
}
