package gin

import (
	"Go_web/web11/binding"
	"Go_web/web11/render"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	MIMEPlain             = binding.MIMEPlain
)

const abortIndex int8 = math.MaxInt8 / 2

//Context用于处理上下文请求和回应
//定义context用于简化处理单个请求的多个goroutine之间与请求域数据，取消信号，截止时间等相关操作，这些操作可能涉及多个API的调用
type Context struct {
	//响应输出流（私有，供框架内部数据写出）
	writermen responseWriter
	//保存客户端发送的所有信息
	Request *http.Request
	//响应输出流（公有，供处理函数使用）
	//初始化后，由writermen克隆而来
	Writer ResponseWriter

	//保存解析得到的参数
	Params Params
	//该请求对应的处理函数链，从树节点中获取
	handlers HandlersChain
	//记录已经被处理的函数个数
	index int8
	//当前请求的完整路径
	fullPath string
	//核心引擎
	engine *Engine
	params *Params
	//并发读写锁
	KeysMutex *sync.Mutex

	//用于保存当前会话的键值对，用于在不同处理函数中传递
	Keys map[string]interface{}

	//处理函数链输出的错误信息
	//（待定）

	//客户端希望接收的数据类型
	Accepted []string

	//存储URL中的查询参数，如:/test?name=jhon&age=11
	queryCache url.Values

	//用于存储POST请求等提交的body参数
	formCache url.Values

	//用来限制第三方 Cookie，一个int值，有Strict、Lax、None
	// Strict:只有当前网页的 URL 与请求目标一致，才会带上 Cookie
	// Lax规则稍稍放宽，大多数情况也是不发送第三方 Cookie，
	// 但是导航到目标网址的 Get 请求除外
	// 设置了Strict或Lax以后，基本就杜绝了 CSRF 攻击
	sameSite http.SameSite

	//Errors 是附加到所有使用此上下文的处理程序/中间件的错误列表
	Errors errorMsgs
}

//初始化Context
func (c *Context)reset()  {
	c.Writer = &c.writermen
	c.Params = c.Params[0:0]
	c.handlers = nil
	c.index = -1
	c.fullPath = ""
	c.Keys = nil
	c.Accepted = nil
	c.queryCache = nil
	c.formCache = nil
	*c.params = (*c.params)[0:0]
}

func (c *Context)Next()  {
	c.index++
	for c.index < int8(len(c.handlers)){
		c.handlers[c.index](c)
		c.index++
	}
}

//bodyAllowedForStatus是http.bodyAllowedForStatus 非导出函数的副本
func bodyAllowedForStatus(status int) bool {
	switch{
	case status >= 100 && status <= 199:
		return false
	case status == http.StatusNoContent:
		return false
	case status == http.StatusNotModified:
		return false
	}
	return true
}

//设置HTTP状态响应代码
func (c *Context)Status(code int)  {
	c.writermen.WriterHeader(code)
}

//Render 写入响应头并调用 render.Render 来呈现数据
func (c *Context)Render(code int,r render.Render)  {
	c.Status(code)

	if !bodyAllowedForStatus(code){
		r.WriteContentType(c.Writer)
		c.Writer.WriteHeaderNow()
		return
	}
	if err := r.Render(c.Writer); err != nil{
		panic(err)

	}
}

//JSON方法
func (c *Context)JSON(code int,obj interface{})  {
	c.Render(code,render.JSON{Data: obj})
}

func (c *Context) ClientIP() string {
	if c.engine.AppEngine {
		if addr := c.requestHeader("X-Appengine-Remote-Addr"); addr != "" {
			return addr
		}
	}

	remoteIP, trusted := c.RemoteIP()
	if remoteIP == nil {
		return ""
	}

	if trusted && c.engine.ForwardedByClientIP && c.engine.RemoteIPHeaders != nil {
		for _, headerName := range c.engine.RemoteIPHeaders {
			ip, valid := validateHeader(c.requestHeader(headerName))
			if valid {
				return ip
			}
		}
	}
	return remoteIP.String()
}

func (c *Context) RemoteIP() (net.IP, bool) {
	ip, _, err := net.SplitHostPort(strings.TrimSpace(c.Request.RemoteAddr))
	if err != nil {
		return nil, false
	}
	remoteIP := net.ParseIP(ip)
	if remoteIP == nil {
		return nil, false
	}

	if c.engine.trustedCIDRs != nil {
		for _, cidr := range c.engine.trustedCIDRs {
			if cidr.Contains(remoteIP) {
				return remoteIP, true
			}
		}
	}

	return remoteIP, false
}

func validateHeader(header string) (clientIP string, valid bool) {
	if header == "" {
		return "", false
	}
	items := strings.Split(header, ",")
	for i, ipStr := range items {
		ipStr = strings.TrimSpace(ipStr)
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return "", false
		}

		// We need to return the first IP in the list, but,
		// we should not early return since we need to validate that
		// the rest of the header is syntactically valid
		if i == 0 {
			clientIP = ipStr
			valid = true
		}
	}
	return
}


func (c *Context) requestHeader(key string) string {
	return c.Request.Header.Get(key)
}
