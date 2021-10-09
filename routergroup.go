package gin

import (
	"net/http"
	"regexp"
)

//定义路由组结构体
type RouterGroup struct {
	//用于存储中间件
	Handlers HandlersChain
	//用于储存路径
	basePath string
	engine *Engine
	//判断路有树是否存在
	root bool
}

//IRouter定义了所有路由器句柄接口，包括单路由器和组路由器。
type IRouter interface {
	IRoutes
	//创建新的路由组，将所有公共的中间件和路由添加进去
	Group (string ,...HandlerFunc)*RouterGroup
}

//RouterGroup实现了IRouter接口
//用于绑定路由信息的函数
type IRoutes interface {
	//将中间件添加到路由组中
	Use (...HandlerFunc) IRoutes
	//用给定的路径和方法注册一个新的请求句柄和中间件
	//此功能用于批量加载，并允许使用不太常用、非标准化或自定义的方法（例如，用于与代理的内部通信）
	//对于GET, POST, PUT, PATCH ,DELETE请求，可以使用各自的快捷创建方式
	Handle (string,string,...HandlerFunc) IRoutes
	//注册一个匹配所有HTTP方法的路由
	Any (string,...HandlerFunc) IRoutes

	GET (string,...HandlerFunc) IRoutes
	POST (string, ...HandlerFunc) IRoutes
	PUT (string,...HandlerFunc) IRoutes
	PATCH(string,...HandlerFunc) IRoutes
	DELETE(string,...HandlerFunc) IRoutes
	OPTIONS(string, ...HandlerFunc) IRoutes
	HEAD(string, ...HandlerFunc) IRoutes

	//注册单个路由，以便为本地文件提供服务
	StaticFile(string, string) IRoutes
	Static(string, string) IRoutes
	StaticFS(string, http.FileSystem) IRoutes
}

var _ IRouter = &RouterGroup{}

//用于添加中间件
func (group *RouterGroup)Use(middleware ...HandlerFunc) IRoutes {
	group.Handlers = append(group.Handlers,middleware...)
	return group.returnObj()
}

//Group创建一个新的路由器组,添加所有具有通用中间件或相同路径前缀的路由
func (group *RouterGroup) Group(relativePath string, handlerFunc ...HandlerFunc) *RouterGroup {
	return &RouterGroup{
		Handlers:group.combineHandlers(handlerFunc), //添加通用中间件
		basePath: group.calculateAbsolutePath(relativePath), //添加相同路径前缀路由
		engine: group.engine,
	}
}

//定义handle方法
func (group *RouterGroup)handle(httpMethod,relativePath string,handlers HandlersChain) IRoutes {
	//计算最简洁的相对路径
	absolutePath := group.calculateAbsolutePath(relativePath)
	//合并函数，将自己编写的函数和中间件结合
	handlers = group.combineHandlers(handlers)
	//向engine对象中添加路由信息
	group.engine.addRouter(httpMethod,absolutePath,handlers)
	return group.returnObj()
}

func (group *RouterGroup) Handle(httpMethod string, relativePath string, handlers ...HandlerFunc) IRoutes {
	if matches, err := regexp.MatchString("^[A-Z]+$", httpMethod); !matches || err != nil {
		panic("http method " + httpMethod + " is not valid")
	}
	return group.handle(httpMethod, relativePath, handlers)
}

func (group *RouterGroup) Any(relativePath string, handlers ...HandlerFunc) IRoutes {
	group.handle(http.MethodGet, relativePath, handlers)
	group.handle(http.MethodPost, relativePath, handlers)
	group.handle(http.MethodPut, relativePath, handlers)
	group.handle(http.MethodPatch, relativePath, handlers)
	group.handle(http.MethodHead, relativePath, handlers)
	group.handle(http.MethodOptions, relativePath, handlers)
	group.handle(http.MethodDelete, relativePath, handlers)
	return group.returnObj()
}

func (group *RouterGroup) GET(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(http.MethodGet,relativePath,handlers)
}

func (group *RouterGroup) POST(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(http.MethodPost,relativePath,handlers)
}

func (group *RouterGroup) PUT(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(http.MethodPut,relativePath,handlers)
}

func (group *RouterGroup) PATCH(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(http.MethodPatch,relativePath,handlers)
}

func (group *RouterGroup) DELETE(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(http.MethodDelete,relativePath,handlers)
}

func (group *RouterGroup) OPTIONS(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(http.MethodOptions,relativePath,handlers)
}

func (group *RouterGroup) HEAD(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(http.MethodHead,relativePath,handlers)
}

func (group *RouterGroup) StaticFile(s string, s2 string) IRoutes {
	panic("implement me")
}

func (group *RouterGroup) Static(s string, s2 string) IRoutes {
	panic("implement me")
}

func (group *RouterGroup) StaticFS(s string, system http.FileSystem) IRoutes {
	panic("implement me")
}

//向路由组中添加通用中间件
func (group *RouterGroup) combineHandlers(handlers HandlersChain) HandlersChain {
	finalSize := len(group.Handlers) + len(handlers)
	//判断中间件数量是否超出限制(一般在Context中做出限制)
	if finalSize >= int(abortIndex){
		panic("too many handlers")
	}
	//合并中间件
	mergedHandlers := make(HandlersChain,finalSize)
	//拷贝
	copy(mergedHandlers,group.Handlers)
	copy(mergedHandlers,handlers)
	return mergedHandlers
}

//向路由组中添加相同路径前缀的路由
func (group *RouterGroup) calculateAbsolutePath(relativePath string) string {
	return joinPaths(group.basePath,relativePath)
}

//当根节点不为空时，输出Engine
func (group *RouterGroup) returnObj() IRoutes {
	if group.root{
		return group.engine
	}
	return group
}


