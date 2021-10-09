package gin

import (
	"html/template"
	"net"
	"net/http"
	"path"
	"strings"
	"sync"
)

//初始化gin对象Engine

var (
	default404Body = []byte("404 page not found")
	default405Body = []byte("405 method not allowed")
)

type HandlerFunc func(*Context)

type HandlersChain []HandlerFunc

func (c HandlersChain) Last() HandlerFunc {
	if length := len(c); length > 0 {
		return c[length-1]
	}
	return nil
}

type Engine struct {
	//路由组,存储中间件信息
	RouterGroup
	//是否启动自动重定向
	RedirectTrailingSlash bool

	//是否起动请求路由修复功能
	//RedirectFixedPath bool
	//启动后，如果找不到当前路由匹配HTTP方法，而在其他HTTP方法中找到，则返回405响应码
	HandleMethodNotAllowed bool
	//是否获取真正的客户端IP，而不是代理服务器IP，
	// 开启后将会从"X-Real-IP和X-Forwarded-For"中解析得到客户端IP
	ForwardedByClientIP bool
	//启用后将在头部加入"X-AppEngine"标识，以便与PaaS集成
	AppEngine bool
	//启用后，将使用原有的URL.rawPath(没有对转义字符进行处理的，
	// 如%/+等)地址来进行解析，而不是使用URL.path来解析，默认为false
	UseRawPath bool
	//启用后，路径中的转义字符将不会被转义
	UnescapePathValues bool
	//设置用来缓存客户端发送的文件的缓存区大小,默认：32m
	MaxMultipartMemory int64
	//RemoveExtraSlash 即使带有额外的斜杠，也可以从 URL 解析参数
	RemoveExtraSlash bool
	//用于获取客户端IP时的标头列表
	RemoteIPHeaders []string
	//用于保存网络来源列表（IPv4 地址、IPv4 CIDR、IPv6 地址或 IPv6 CIDR）
	TrustedProxies []string


	////用来保存tmpl文件中用来引用变量的界定符，默认"{{}}"
	//delims render.Delims
	////设置防止JSON劫取，在json字符串前加的逻辑代码，默认为while(1)
	//secureJsonPrefix string
	////Html文件解析器
	//HTMLRender render.HTMLRender
	//tmpl文件的内建函数列表，可以在tmpl文件中调用函数，使用
	//    //router.SetFuncMap(template.FuncMap{
	//    //      "formatAsDate": formatAsDate,
	//    //})可设置
	FuncMap template.FuncMap


	// HandlersChain就是func(*Context)数组
	// 以下四个调用链中保存的就是在不同情况下回调的处理函数
	// 找不到匹配路由(404)
	allNoRoute HandlersChain
	//返回405状态时回调
	allNoMethod HandlersChain
	//没有匹配路由时回调，主要是测试代码时使用
	noRoute HandlersChain
	//没有配置映射方法时回调
	noMethod HandlersChain
	//连接池用于保存与客户端的连接上下文(Context)
	pool sync.Pool
	//路径搜索树
	trees        methodTrees
	//最大参数
	maxParams    uint16
	trustedCIDRs []*net.IPNet

}

//初始化Engine对象
func New() *Engine {
	debugPrintWARNINGNew()
	engine := &Engine{
		//实例化RouterGroup
		RouterGroup: RouterGroup{
			Handlers: nil,
			basePath: "/",
			root: true,
		},
		FuncMap: template.FuncMap{},
		//tress负责存储路由和handle方法的映射,只有9种handle方法
		trees:make(methodTrees,0,9),
	}
	engine.RouterGroup.engine = engine
	//这里采用 sync/pool 实现context池,减少频繁context实例化带来的资源消耗
	engine.pool.New = func() interface{} {
		return engine.allocateContext()
	}
	return engine
}

//Default返回已连接Logger和Recovery中间件的Engine实例。
func (engine *Engine)Default()*Engine {

	return nil
}

func (engine *Engine)allocateContext()*Context {
	v := make(Params,0,engine.maxParams)
	return &Context{engine: engine,params: &v}
}

//将全局中间件添加到路由中（使用Use添加的中间件将作用于所有的请求）
func (engine *Engine)Use(middleware ...HandlerFunc) IRoutes {
	engine.RouterGroup.Use(middleware...)
	engine.rebuild404Handlers()
	engine.rebuild405Handlers()
	return engine
}

//找不到匹配路由时，返回404
func (engine *Engine)rebuild404Handlers()  {
	engine.allNoRoute = engine.combineHandlers(engine.noRoute)
}

//找不到匹配HTTP方法时，返回405
func (engine *Engine)rebuild405Handlers()  {
	engine.allNoMethod = engine.combineHandlers(engine.noMethod)
}

//Run 将路由器连接到 http.Server 并开始侦听和服务 HTTP 请求。
func (engine *Engine)Run(addr ...string) (err error) {
	defer func() { debugPrintError(err) }()

	trustedCIDRs,err := engine.prepareTrustedCIDRs()
	if err != nil{
		return err
	}
	engine.trustedCIDRs = trustedCIDRs
	address := resolveAddress(addr)
	debugPrint("Listening and serving HTTP on %s\n", address)
	err = http.ListenAndServe(address,engine)
	return
}

func (engine *Engine)prepareTrustedCIDRs()([]*net.IPNet,error)  {
	//TrustedProxies用于保存网络来源列表
	if engine.TrustedProxies == nil {
		return nil, nil
	}
	cidr := make([]*net.IPNet,0,len(engine.TrustedProxies))
	for _,trustedProxy := range engine.TrustedProxies{
		if !strings.Contains(trustedProxy,"/"){
			ip := parseIP(trustedProxy)
			if ip == nil{
				return cidr, &net.ParseError{Type: "IP address", Text: trustedProxy}
			}
			switch len(ip){
			case net.IPv4len:
				trustedProxy += "/32"
			case net.IPv6len:
				trustedProxy += "/128"
			}
		}
		_,cidrNet,err := net.ParseCIDR(trustedProxy)
		if err != nil{
			return cidr,err
		}
		cidr = append(cidr,cidrNet)
	}
	return cidr,nil
}

func parseIP(ip string) net.IP {
	parsedIP := net.ParseIP(ip)
	if ipv4 := parsedIP.To4();ipv4 != nil{
		return ipv4
	}
	return parsedIP
}


//向engine对象中添加路由信息
func (engine *Engine)addRouter(method,path string,handlers HandlersChain)  {
	//对添加路由信息进行判断
	assert1(path[0]=='/',"path must begin with '/'")
	assert1(method != "","HTTP method can't be empty")
	assert1(len(handlers) > 0,"there must be at least one handler")
	//输出路由信息
	debugPrintRoute(method,path,handlers)

	//构建方法请求你树
	root := engine.trees.get(method)
	if root == nil{
		//新建请求树节点
		root = new(node)
		root.fullPath = "/"
		//添加请求方法
		engine.trees = append(engine.trees,methodTree{method: method,root: root})
	}
	//正式添加路由
	root.addRoute(path,handlers)
}

//
func (engine *Engine)ServeHTTP(w http.ResponseWriter,req *http.Request)  {
	//从pool中取出一个上写文对象
	c := engine.pool.Get().(*Context)
	//将上下文对象中的响应流设置成传入参数
	c.writermen.reset(w)
	//将上下文请求数据结构设置成传入参数
	c.Request = req
	//初始化上下文对象
	c.reset()
	//正式处理请求
	engine.handleHTTPRequest(c)
	//使用完毕后，放回连接池
	engine.pool.Put(c)
	
}

//处理请求
func (engine *Engine)handleHTTPRequest(c *Context)  {
	//获取客户端的http请求方法
	httpMethod := c.Request.Method
	//获取请求的URL地址，这里的URL经过处理的
	rPath := c.Request.URL.Path
	//是否不启动字符转义
	unescape := false
	//判断是否启用原有URL，为转义字符
	if engine.UseRawPath && len(c.Request.URL.RawPath)>0{
		rPath = c.Request.URL.RawPath
		unescape = engine.UnescapePathValues
	}
	//判断是否需要移除多余的分隔符"/"
	if engine.RemoveExtraSlash{
		rPath = cleanPath(rPath)
	}

	t := engine.trees
	//遍历请求树
	for i,t1 := 0,len(t); i<t1; i++{
		if t[i].method != httpMethod{
			continue
		}
		//首先获取到指定HTTP方法的请求树的根节点
		root := t[i].root
		//从根节点开始搜索匹配路径的节点
		value := root.getValue(rPath,c.params,unescape)
		//将节点中存储的信息拷贝到Context上下文中
		if value.params != nil {
			c.Params = *value.params
		}
		if value.handlers != nil {
			c.handlers = value.handlers
			c.fullPath = value.fullPath
			c.Next()
			c.writermen.WriteHeaderNow()
			return
		}
		//如果没有找到匹配点，则考虑如下特殊情况
		if httpMethod != "CONNECT" && rPath != "/"{
			//若启动自动重定向，则删除最后的'/'并重定向
			if value.tsr && engine.RedirectTrailingSlash{
				redirectTrailingSlash(c)
				return
			}
			//启动路径修复后，当/.../foo找不到匹配路由时
			//会自动删除/.../部分，找到匹配路由，并重定向
			if  engine.RedirectFixedPath && redirectFixedPath(c, root, engine.RedirectFixedPath){
				return
			}
		}
		break
	}
	//HTTP方法不匹配，而路径匹配，则返回405
	if engine.HandleMethodNotAllowed{
		for _,tree := range engine.trees{
			if tree.method == httpMethod{
				continue
			}
			if value := tree.root.getValue(rPath,nil,unescape);value.handlers != nil{
				c.handlers = engine.allNoMethod
				serverError(c,http.StatusMethodNotAllowed,default405Body)
				return
			}
		}
	}
	//未找到路由，则返回404
	c.handlers = engine.allNoRoute
	serverError(c,http.StatusNotFound,default404Body)
}

var mimePlain = []string{MIMEPlain}

func serverError(c *Context,code int,defaultMessage []byte)  {
	c.writermen.status = code
	c.Next()
	if c.writermen.Written(){
		return
	}
	if c.writermen.Status() == code{
		c.writermen.Header()["Content-Type"] = mimePlain
		_,err := c.writermen.Write(defaultMessage)
		if err != nil{
			debugPrint("cannot write message to writer during serve error: %v", err)
		}
		return
	}
	c.writermen.WriteHeaderNow()
}



//路由重定向函数
func redirectTrailingSlash(c *Context) {
	req := c.Request
	p := req.URL.Path
	if prefix := path.Clean(c.Request.Header.Get("X-Forwarded-Prefix"));prefix != "."{
		p = prefix + "/" + req.URL.Path
	}
	req.URL.Path = p +"/"
	if length := len(p); length > 1 && p[length-1] == '/' {
		req.URL.Path = p[:length-1]
	}

	redirectRequest(c)
}

//当/.../foo找不到匹配路由时，会自动删除/.../部分，找到匹配路由，并重定向
func redirectFixedPath(c *Context,root *node,trailingSlash bool) bool {
	req := c.Request
	rPath := req.URL.Path
	if fixPath, ok := root.findCaseInsensitivePath(cleanPath(rPath),trailingSlash);ok{
		req.URL.Path = string(fixPath)
		redirectRequest(c)
		return true
	}
	return false
}

//打印重定向后的路由信息
func redirectRequest(c *Context)  {
	req := c.Request
	rPath := req.URL.Path
	rURL := req.URL.String()
	code := http.StatusMovedPermanently
	if req.Method != http.MethodGet{
		code = http.StatusTemporaryRedirect
	}
	debugPrint("redirecting request %d: %s --> %s", code, rPath, rURL)
	http.Redirect(c.Writer,req,rURL,code)
	c.writermen.WriteHeaderNow()
}