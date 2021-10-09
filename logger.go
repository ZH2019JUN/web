package gin

import (
	"fmt"
	"github.com/mattn/go-isatty"
	"io"
	"net/http"
	"os"
	"time"
)

type consoleColorModeValue int

const (
	autoColor consoleColorModeValue = iota
	disableColor
	forceColor
)

const (
	green   = "\033[97;42m"
	white   = "\033[90;47m"
	yellow  = "\033[90;43m"
	red     = "\033[97;41m"
	blue    = "\033[97;44m"
	magenta = "\033[97;45m"
	cyan    = "\033[97;46m"
	reset   = "\033[0m"
)

var consoleColorMode = autoColor


type LoggerConfig struct {
	Formatter LogFormatter
	//写入日志的写入器
	Output io.Writer
	//不写入日志的url路径数组
	SkipPaths []string
}

//LogFormatter 给出传递给 LoggerWithFormatter 的格式化函数的签名
type LogFormatter func(params LogFormatterParams) string

//LogFormatterParams 是任何格式化程序将在记录时间到来时传递的结构
type LogFormatterParams struct {
	Request *http.Request
	//显示服务器返回响应后的时间
	TimeStamp time.Time
	//HTTP响应码
	StatusCode int
	//记录处理请求话费的时间
	Latency time.Duration
	ClientIP string
	//请求的HTTP方法
	Method string
	//请求路径
	Path string
	//处理请求发生错误时，设置ErrorMessage
	ErrorMessage string
	//显示 gin 的输出描述符是否引用终端
	isTerm bool
	//Response Body的大小
	BodySize int
	//请求上下文设置的键
	Keys map[string]interface{}
}

//使用不同颜色字体将http状态代码记录到终端
func (p *LogFormatterParams)StatusCodeColor() string {
	code := p.StatusCode

	switch  {
	case code >= http.StatusOK && code <= http.StatusMultipleChoices:
		return green
	case code >= http.StatusMultipleChoices && code <= http.StatusBadRequest:
		return white
	case code >= http.StatusBadRequest && code <= http.StatusInternalServerError:
		return yellow
	default:
		return red
	}
}

//用于将http方法记录到终端
func (p *LogFormatterParams)MethodColor() string {
	method := p.Method

	switch method {
	case http.MethodGet:
		return blue
	case http.MethodPost:
		return cyan
	default:
		return reset
	}
}

//重置所有转义属性
func (p *LogFormatterParams)ResetColor() string {
	return reset
}

//指示是否可以将颜色输出到日志
func (p *LogFormatterParams)IsOutputColor() bool {
	return consoleColorMode == forceColor || (consoleColorMode == autoColor && p.isTerm)
}

//Logger中间件使用的默认日志函数
var defaultLogFormatter = func(param LogFormatterParams) string{
	var statusColor,methodColor,resetColor string
	if param.IsOutputColor(){
		statusColor = param.StatusCodeColor()
		methodColor = param.MethodColor()
		resetColor = param.ResetColor()
	}
	if param.Latency > time.Minute{
		param.Latency = param.Latency - param.Latency%time.Second
	}
	return fmt.Sprintf("[GIN] %v |%s %3d %s| %13v | %15s |%s %-7s %s %#v\n%s",
		param.TimeStamp.Format("2006/01/02 - 15:04:05"),
		statusColor, param.StatusCode, resetColor,
		param.Latency,
		param.ClientIP,
		methodColor, param.Method, resetColor,
		param.Path,
		param.ErrorMessage,
	)
}

//禁用控制台中的颜色输出
func DisableConsoleColor()  {
	consoleColorMode = disableColor
}

//强制在控制台中输出颜色
func ForceConsoleColor()  {
	consoleColorMode = forceColor
}

//ErrorLogger为任何错误返回一个handlerFunc
func ErrorLogger() HandlerFunc {
	return ErrorLoggerT(ErrorTypeAny)
}

func ErrorLoggerT(typ ErrorType) HandlerFunc {
	return func(c *Context) {
		c.Next()
		errors := c.Errors.ByType(typ)
		if len(errors)>0{
			c.JSON(-1,errors)
		}
	}
}

//Logger 实例一个 Logger 中间件，它将日志写入 gin.DefaultWriter
func Logger() HandlerFunc {
	return LoggerWithConfig(LoggerConfig{})
}

//实例具有指定日志格式功能的 Logger 中间件
func LoggerWithFormatter(f LogFormatter) HandlerFunc {
	return LoggerWithConfig(LoggerConfig{
		Formatter: f,
	})
}

//实例具有指定写入器缓冲区的 Logger 中间件
func LoggerWithWriter(out io.Writer,notlogged ...string) HandlerFunc {
	return LoggerWithConfig(LoggerConfig{
		Output: out,
		SkipPaths: notlogged,
	})
}

//实例一个带有配置的 Logger 中间件
func LoggerWithConfig(conf LoggerConfig) HandlerFunc {
	formatter := conf.Formatter
	if formatter == nil{
		formatter = defaultLogFormatter
	}

	out := conf.Output
	if out == nil{
		out = DefaultWriter
	}

	notlogged := conf.SkipPaths

	isTerm := true

	if w,ok := out.(*os.File);!ok ||os.Getenv("TERM") == "dumb"||
		(!isatty.IsTerminal(w.Fd()) && !isatty.IsCygwinTerminal(w.Fd())){
		isTerm = false
	}

	var skip map[string]struct{}

	if length := len(notlogged);length>0{
		skip = make(map[string]struct{},length)

		for _,path := range notlogged{
			skip[path] = struct{}{}
		}
	}
	return func(c *Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		if _,ok := skip[path];ok{
			param := LogFormatterParams{
				Request: c.Request,
				isTerm: isTerm,
				Keys:c.Keys,
			}
			param.TimeStamp = time.Now()
			param.Latency = param.TimeStamp.Sub(start)
			param.ClientIP = c.ClientIP()
			param.Method = c.Request.Method
			param.StatusCode = c.Writer.Status()
			param.ErrorMessage = c.Errors.ByType(ErrorTypeAny).String()

			param.BodySize = c.Writer.Size()

			if raw != ""{
				path = path + "?" + raw
			}
			param.Path = path

			fmt.Fprint(out,formatter(param))
		}
	}
}