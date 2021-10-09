package gin

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

const (
	noWritten     = -1
	defaultStatus = http.StatusOK
)

//响应输出流接口（公有）
type ResponseWriter interface {
	http.ResponseWriter
	http.Hijacker
	http.Flusher

	//返回当前HTTP请求的响应状态码
	Status() int

	//返回已写入http正文的字节数
	Size() int

	//将字符串写入响应正文
	WriteString(string)(int,error)

	//如果正文已写入，返回true
	Written() bool

	//强制写入 http 标头（状态代码 + 标头）
	WriteHeaderNow()

	//获取用于服务器推送的http.Pusher
	Pusher() http.Pusher

}

//响应输出流结构体（私有）,实现了ResponseWriter接口
type responseWriter struct {
	http.ResponseWriter
	size int
	status int
}

var _ ResponseWriter = &responseWriter{}

//将响应流设置成传入参数,传入Context上下文中
func (w *responseWriter)reset(writer http.ResponseWriter)  {
	w.ResponseWriter = writer
	w.size = noWritten
	w.status = defaultStatus
}


func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	panic("implement me")
}

func (w *responseWriter) Flush() {
	panic("implement me")
}

func (w *responseWriter) Status() int {
	return w.status
}

func (w *responseWriter) Size() int {
	return w.size
}

func (w *responseWriter) WriteString(s string) (n int, err error) {
	w.WriteHeaderNow()
	n, err = io.WriteString(w.ResponseWriter, s)
	w.size += n
	return
}

func (w *responseWriter) Written() bool {
	return w.status != noWritten
}

//状态响应码
func (w *responseWriter)WriterHeader(code int)  {
	if code>0 && w.status != code{
		if w.Written(){
			debugPrint("[WARNING] Headers were already written. Wanted to override status code %d with %d",w.status,code)
		}
		w.status = code
	}
}

func (w *responseWriter) WriteHeaderNow() {
	if !w.Written() {
		w.size = 0
		w.ResponseWriter.WriteHeader(w.status)
	}
}

func (w *responseWriter) Pusher() http.Pusher {
	panic("implement me")
}

