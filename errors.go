package gin

import (
	"fmt"
	"strings"
)

type ErrorType uint64

const (
	// ErrorTypeBind is used when Context.Bind() fails.
	ErrorTypeBind ErrorType = 1 << 63
	// ErrorTypeRender is used when Context.Render() fails.
	ErrorTypeRender ErrorType = 1 << 62
	// ErrorTypePrivate indicates a private error.
	ErrorTypePrivate ErrorType = 1 << 0
	// ErrorTypePublic indicates a public error.
	ErrorTypePublic ErrorType = 1 << 1
	// ErrorTypeAny indicates any other error.
	ErrorTypeAny ErrorType = 1<<64 - 1
	// ErrorTypeNu indicates any other error.
	ErrorTypeNu = 2
)

//错误规范
type Error struct {
	Err error
	Type ErrorType
	Meta interface{}
}

type errorMsgs []*Error

var _ error = &Error{}

//判断一个错误
func(msg *Error) IsType(flags ErrorType) bool {
	return (msg.Type & flags) >0
}

func (msg Error) Error() string {
	return msg.Err.Error()
}

//返回过滤后的字节的只读副本
func (a errorMsgs)ByType(typ ErrorType) errorMsgs  {
	if len(a) == 0{
		return nil
	}
	if typ == ErrorTypeAny{
		return a
	}
	var result errorMsgs
	for _,msg := range a{
		if msg.IsType(typ){
			result = append(result,msg)
		}
	}
	return result
}

func (a errorMsgs) String() string {
	if len(a) == 0 {
		return ""
	}
	var buffer strings.Builder
	for i, msg := range a {
		fmt.Fprintf(&buffer, "Error #%02d: %s\n", i+1, msg.Err)
		if msg.Meta != nil {
			fmt.Fprintf(&buffer, "     Meta: %v\n", msg.Meta)
		}
	}
	return buffer.String()
}