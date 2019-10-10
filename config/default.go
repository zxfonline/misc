package config

import (
	"os"
)

var (
	_default     *Config
	BASE_SECTION string
)

//初始化全局配置文件
func InitConfig(fname string, base_section string) (cfg *Config, err error) {
	cfg, err = ReadDefault(fname)
	if err != nil {
		return
	}
	_default = cfg
	BASE_SECTION = base_section
	return
}
func Merge(source *Config) {
	_default.Merge(source)
}
func String(section string, option string) (value string, err error) {
	if section == "" {
		section = BASE_SECTION
	}
	return _default.String(section, option)
}
func RawStringDefault(option string) (value string, err error) {
	return _default.RawStringDefault(option)
}
func RawString(section string, option string) (value string, err error) {
	if section == "" {
		section = BASE_SECTION
	}
	return _default.RawString(section, option)
}
func Int(section string, option string) (value int, err error) {
	if section == "" {
		section = BASE_SECTION
	}
	return _default.Int(section, option)
}
func Float(section string, option string) (value float64, err error) {
	if section == "" {
		section = BASE_SECTION
	}
	return _default.Float(section, option)
}
func Bool(section string, option string) (value bool, err error) {
	if section == "" {
		section = BASE_SECTION
	}
	return _default.Bool(section, option)
}

func HasSection(section string) bool {
	return _default.HasSection(section)
}
func Sections() (sections []string) {
	return _default.Sections()
}
func RemoveSection(section string) bool {
	return _default.RemoveSection(section)
}
func AddSection(section string) bool {
	return _default.AddSection(section)
}
func WriteFile(fname string, perm os.FileMode, header string) error {
	return _default.WriteFile(fname, perm, header)
}
func AddOption(section string, option string, value string) bool {
	return _default.AddOption(section, option, value)
}
func RemoveOption(section string, option string) bool {
	return _default.RemoveOption(section, option)
}
func HasOption(section string, option string) bool {
	return _default.HasOption(section, option)
}
func Options(section string) (options []string, err error) {
	return _default.Options(section)
}
func SectionOptions(section string) (options []string, err error) {
	return _default.SectionOptions(section)
}
