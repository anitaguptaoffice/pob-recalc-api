package util

import (
	"reflect"
	"testing"
)

func TestTemplateRender(t *testing.T) {
	tests := []struct {
		name     string
		template string
		params   map[int]string
		want     string
		panic    bool
	}{
		{
			name:     "无占位符",
			template: "Hello World",
			params:   map[int]string{},
			want:     "Hello World",
		},
		{
			name:     "单个占位符",
			template: "Hello {0}",
			params:   map[int]string{0: "World"},
			want:     "Hello World",
		},
		{
			name:     "多个占位符",
			template: "{0} likes {1}",
			params:   map[int]string{0: "Alice", 1: "Bob"},
			want:     "Alice likes Bob",
		},
		{
			name:     "占位符顺序不一致",
			template: "{1} and {0}",
			params:   map[int]string{0: "first", 1: "second"},
			want:     "second and first",
		},
		{
			name:     "重复占位符",
			template: "{0} says {0}",
			params:   map[int]string{0: "hello"},
			want:     "hello says hello",
		},
		{
			name:     "空参数值",
			template: "Start{0}End",
			params:   map[int]string{0: ""},
			want:     "StartEnd",
		},
		{
			name:     "参数包含特殊字符",
			template: "Name: {0}",
			params:   map[int]string{0: "John <Doe>"},
			want:     "Name: John <Doe>",
		},
		{
			name:     "缺失参数",
			template: "Hello {0} {1}",
			params:   map[int]string{0: "World"},
			panic:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := NewTemplate(tt.template)

			if tt.panic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Render() should panic but didn't")
					}
				}()
			}

			got := tmpl.Render(tt.params)
			if got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTemplateParseParams(t *testing.T) {
	tests := []struct {
		name     string
		template string
		input    string
		want     map[int]string
		wantOk   bool
	}{
		{
			name:     "无占位符",
			template: "Hello World",
			input:    "Hello World",
			want:     nil,
			wantOk:   false,
		},
		{
			name:     "单个占位符",
			template: "Hello {0}",
			input:    "Hello World",
			want:     map[int]string{0: "World"},
			wantOk:   true,
		},
		{
			name:     "多个占位符",
			template: "{0} likes {1}",
			input:    "Alice likes Bob",
			want:     map[int]string{0: "Alice", 1: "Bob"},
			wantOk:   true,
		},
		{
			name:     "占位符顺序不一致",
			template: "{1} and {0}",
			input:    "second and first",
			want:     map[int]string{0: "first", 1: "second"},
			wantOk:   true,
		},
		{
			name:     "重复占位符",
			template: "{0} says {0}",
			input:    "hello says hello",
			want:     map[int]string{0: "hello"},
			wantOk:   true,
		},
		{
			name:     "空参数值",
			template: "Start{0}End",
			input:    "StartEnd",
			want:     map[int]string{0: ""},
			wantOk:   true,
		},
		{
			name:     "参数包含特殊字符",
			template: "Name: {0}",
			input:    "Name: John <Doe>",
			want:     map[int]string{0: "John <Doe>"},
			wantOk:   true,
		},
		{
			name:     "不匹配-静态文本不同",
			template: "Hello {0}",
			input:    "Hi World",
			want:     nil,
			wantOk:   false,
		},
		{
			name:     "不匹配-缺少后续文本",
			template: "{0} World",
			input:    "Hello",
			want:     nil,
			wantOk:   false,
		},
		{
			name:     "最后一个参数可包含多余文本",
			template: "Hello {0}",
			input:    "Hello World!",
			want:     map[int]string{0: "World!"},
			wantOk:   true,
		},
		{
			name:     "不匹配-中间参数后有多余文本",
			template: "Hello {0} World",
			input:    "Hello X World!",
			want:     nil,
			wantOk:   false,
		},
		{
			name:     "复杂模板",
			template: "User {0} has {1} messages in {2}",
			input:    "User Alice has 42 messages in inbox",
			want:     map[int]string{0: "Alice", 1: "42", 2: "inbox"},
			wantOk:   true,
		},
		{
			name:     "参数值包含静态文本",
			template: "{0} middle {1}",
			input:    "abc middle middle def",
			want:     map[int]string{0: "abc", 1: "middle def"},
			wantOk:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := NewTemplate(tt.template)
			got, ok := tmpl.ParseParams(tt.input)

			if ok != tt.wantOk {
				t.Errorf("ParseParams() ok = %v, want %v", ok, tt.wantOk)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseParams() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplateRenderAndParseParams(t *testing.T) {
	// 测试 Render 和 ParseParams 互为逆操作
	tests := []struct {
		name     string
		template string
		params   map[int]string
	}{
		{
			name:     "简单模板",
			template: "Hello {0}",
			params:   map[int]string{0: "World"},
		},
		{
			name:     "多个参数",
			template: "{0} + {1} = {2}",
			params:   map[int]string{0: "1", 1: "2", 2: "3"},
		},
		{
			name:     "乱序索引",
			template: "{2} {0} {1}",
			params:   map[int]string{0: "a", 1: "b", 2: "c"},
		},
		{
			name:     "重复索引",
			template: "{0} {1} {0}",
			params:   map[int]string{0: "x", 1: "y"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := NewTemplate(tt.template)

			// 渲染
			rendered := tmpl.Render(tt.params)

			// 解析
			parsed, ok := tmpl.ParseParams(rendered)
			if !ok {
				t.Errorf("ParseParams() failed for rendered string: %q", rendered)
				return
			}

			// 验证解析结果与原始参数一致
			if !reflect.DeepEqual(parsed, tt.params) {
				t.Errorf("ParseParams() = %v, want %v", parsed, tt.params)
			}
		})
	}
}
