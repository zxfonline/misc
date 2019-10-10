package csvconfig

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"

	"topcrown.com/centerserver/misc/fileutil"
	"topcrown.com/centerserver/misc/log"
)

var (
	//已经加载的文件缓存
	_tables map[string]*Table
	//文件路径
	pathPre string
	//文件后缀
	filesuffix string
)

type Record struct {
	Fields map[string]string
}

type Table struct {
	Records []*Record
}

type Query struct {
	Key   string
	Value string
}

//初始化参数
func Init(pathpre, suffix string) {
	filesuffix = suffix
	if filesuffix == "" {
		filesuffix = ".csv"
	}
	pathPre = pathpre
}

func findFile(table string) (*os.File, error) {
	return fileutil.FindFile(fileutil.PathJoin(pathPre, table+filesuffix), os.O_RDONLY, 0)
}

//加载所有配置文件，如果已经加载则覆盖
func Load(_table_list []string) error {
	_tables = make(map[string]*Table)
	for _, table := range _table_list {
		log.Infof("Load csv config: %v.", table)
		f, err := findFile(table)
		if err != nil {
			return err
		}
		defer f.Close()
		err = _initTable(table, f)
		if err != nil {
			return err
		}
	}
	return nil
}

func _initTable(table string, f *os.File) error {
	reader := csv.NewReader(f)
	title, err := reader.Read()
	if err != nil {
		return fmt.Errorf("parse csv config:%v ,error:%v", table, err)
	}
	for idx, val := range title {
		title[idx] = val
	}
	t := Table{Records: make([]*Record, 0)}
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if nil != err {
			return fmt.Errorf("parse csv config:%v ,error:%v", table, err)
		}

		rec := Record{Fields: make(map[string]string)}
		for idx, val := range line {
			rec.Fields[title[idx]] = val
		}
		t.Records = append(t.Records, &rec)
	}
	_tables[table] = &t
	return nil
}

func GetString(table string, queryField string, val string, field string) string {
	rec := getOne(table, queryField, val)
	if rec == nil {
		return ""
	}
	v, ok := rec.Fields[field]
	if !ok {
		return ""
	}
	return v
}

func getOne(tableName string, queryField string, val string) *Record {
	table, ok := _tables[tableName]
	if !ok {
		return nil
	}
	for _, rec := range table.Records {
		v, ok := rec.Fields[queryField]
		if !ok {
			return nil
		}
		if v == val {
			return rec
		}
	}
	return nil
}

func GetLines(table string, querys []*Query) []*Record {
	return getLines(table, querys)
}

func GetLine(table string, querys []*Query) *Record {
	ret := getLines(table, querys)
	if len(ret) > 0 {
		return ret[0]
	}
	return nil
}
func getLines(tableName string, querys []*Query) []*Record {
	table, ok := _tables[tableName]
	if !ok {
		return nil
	}
	var match bool
	lines := make([]*Record, 0)
	for _, rec := range table.Records {
		match = true
		for _, query := range querys {
			v, ok := rec.Fields[query.Key]
			if !ok {
				match = false
				break
			}
			if v != query.Value {
				match = false
				break
			}
		}
		if match {
			lines = append(lines, rec)
		}
	}
	return lines
}

func GetAll(tableName string) []*Record {
	table, ok := _tables[tableName]
	if !ok {
		return nil
	}
	lines := make([]*Record, len(table.Records))
	idx := 0
	for _, rec := range table.Records {
		lines[idx] = rec
		idx++
	}
	return lines
}

// Int8Slice attaches the methods of Interface to []int8, sorting in increasing order.
type Int8Slice []int8

func (p Int8Slice) Len() int           { return len(p) }
func (p Int8Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p Int8Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// Sort is a convenience method.
func (p Int8Slice) Sort() { sort.Sort(p) }

// Ints8 sorts a slice of int8 in increasing order.
func Ints8(a []int8) { sort.Sort(Int8Slice(a)) }

// Int16Slice attaches the methods of Interface to []int16, sorting in increasing order.
type Int16Slice []int16

func (p Int16Slice) Len() int           { return len(p) }
func (p Int16Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p Int16Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// Sort is a convenience method.
func (p Int16Slice) Sort() { sort.Sort(p) }

// Ints32 sorts a slice of int16 in increasing order.
func Ints16(a []int16) { sort.Sort(Int16Slice(a)) }

// Int32Slice attaches the methods of Interface to []int32, sorting in increasing order.
type Int32Slice []int32

func (p Int32Slice) Len() int           { return len(p) }
func (p Int32Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p Int32Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// Sort is a convenience method.
func (p Int32Slice) Sort() { sort.Sort(p) }

// Ints32 sorts a slice of int32 in increasing order.
func Ints32(a []int32) { sort.Sort(Int32Slice(a)) }

// Int64Slice attaches the methods of Interface to []int64, sorting in increasing order.
type Int64Slice []int64

func (p Int64Slice) Len() int           { return len(p) }
func (p Int64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p Int64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// Sort is a convenience method.
func (p Int64Slice) Sort() { sort.Sort(p) }

// Ints64 sorts a slice of int64 in increasing order.
func Ints64(a []int64) { sort.Sort(Int64Slice(a)) }

// Float32Slice attaches the methods of Interface to []float32, sorting in increasing order.
type Float32Slice []float32

func (p Float32Slice) Len() int           { return len(p) }
func (p Float32Slice) Less(i, j int) bool { return p[i] < p[j] || isNaN32(p[i]) && !isNaN32(p[j]) }
func (p Float32Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// isNaN32 is a copy of math.IsNaN to avoid a dependency on the math package.
func isNaN32(f float32) bool {
	return f != f
}

// Sort is a convenience method.
func (p Float32Slice) Sort() { sort.Sort(p) }

// Float64Slice attaches the methods of Interface to []float64, sorting in increasing order.
type Float64Slice []float64

func (p Float64Slice) Len() int           { return len(p) }
func (p Float64Slice) Less(i, j int) bool { return p[i] < p[j] || isNaN64(p[i]) && !isNaN64(p[j]) }
func (p Float64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// isNaN64 is a copy of math.IsNaN to avoid a dependency on the math package.
func isNaN64(f float64) bool {
	return f != f
}

// Sort is a convenience method.
func (p Float64Slice) Sort() { sort.Sort(p) }
