package xml

import "fmt"

// Build 构建信息
type Build struct {
	Level           int
	ClassName       string
	AscendClassName string
}

func NewBuild() *Build {
	return &Build{
		ClassName:       "None",
		AscendClassName: "None",
	}
}

// String 返回XML字符串
func (b *Build) String() string {
	return fmt.Sprintf(`<Build level="%d" className="%s" ascendClassName="%s" targetVersion="3_0" mainSocketGroup="1" viewMode="ITEMS">
</Build>`, b.Level, b.ClassName, b.AscendClassName)
}
