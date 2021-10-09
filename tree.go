package gin

import (
	"net/url"
	"strings"
)

//Param是一个URL参数
type Param struct {
	Key string
	Value string
}

//路由器返回的Param切片
type Params []Param

//Get返回与给定参数匹配的第一个Param值，如果找不到匹配的Param，则返回空字符串
func (ps Params)Get(name string)(string,bool){
	for _,entry := range ps{
		if entry.Key == name{
			return entry.Value,true
		}
	}
	return "", false
}

//ByName返回与给定参数匹配的第一个Param值
func (ps Params)ByName(name string)(val string){
	val,_ = ps.Get(name)
	return
}

type nodeType uint8

const (
	//静态路由信息节点，默认值
	static nodeType = iota
	//根节点
	root
	//参数节点
	param
	//表示当前节点已经包含所有的REST参数了,(有*匹配的节点)
	catchAll
)


//路由树的节点信息
type node struct {
	//节点路径
	path string
	//保存分裂节点的第一个字符
	indices string
	//儿子节点
	children []*node
	//处理函数切片
	handlers HandlersChain
	//优先级.子节点，子子节点所注册的handler数量
	priority uint32
	// 节点类型，包括static, root, param, catchAll
	// static: 静态节点（默认）
	// root: 树的根节点
	// catchAll: 有*匹配的节点
	// param: 参数节点
	nType nodeType
	// 路径上最大参数个数
	maxParams uint8
	// 节点是否是参数节点，比如上面的:post
	wildChild bool
	// 完整路径
	fullPath  string
}

//定义方法数结构体
type methodTree struct {
	method string
	//根节点
	root *node
}

type methodTrees []methodTree

//get返回与给定方法匹配的一个根节点
func (trees methodTrees)get(method string)*node {
	for _,tree := range trees{
		if tree.method == method{
			return tree.root
		}
	}
	return nil
}

func min(a,b int)int  {
	if a>=b{
		return b
	}
	return a
}

//计算最长通用前缀长度
func longestCommonPrefix(a,b string)int  {
	i := 0
	max := min(len(a),len(b))
	for i < max && a[i] == b[i]{
		i++
	}
	return i
}

//增加给定孩子的优先级，并在必要时候进行排序
func (n *node) incrementChildPrio(pos int) int{
	cs := n.children
	cs[pos].priority++
	prio := cs[pos].priority
	newPos := pos
	//对新加入的路径信息按照字母顺序排序(通过优先级判断)
	for ; newPos>0 && cs[newPos-1].priority<prio; newPos--{
		//交换节点位置
		cs[newPos-1],cs[newPos] = cs[newPos],cs[newPos-1]
	}
	//新建索引
	if newPos != pos{
		n.indices = n.indices[:newPos] + n.indices[pos:pos+1] + n.indices[newPos:pos] + n.indices[pos+1:]
	}
	return newPos
}

//通配符表示当前参数必须存在，而*表示该参数可有可无。
//当indices为空时，则表示当前节点只有一个子节点，且wildchild为true，此时是用不上indices的，直接取索引0.

//将具有给定句柄的节点添加熬路径中
func (n *node)addRoute(path string,handlers HandlersChain)  {
	//记录完整路径
	fullPath := path
	//随着匹配路径的增加，优先级增大
	n.priority++

	//如果是空树，直接插入现有节点
	if len(n.path)==0 && len(n.children)==0{
		n.insertChild(path,fullPath,handlers)
		n.nType = root
		return
	}

	//初始化父节点的路径长度
	parentFullPathIndex := 0

walk:
	for{
		//寻找公共前缀的下标索引
		 i := longestCommonPrefix(path,n.path)
		 //分裂路径信息
		// 例如一开始path是search，新加入support，s是他们通用的最长前缀部分
		// 那么会将s拿出来作为parent节点，增加earch和upport作为child节点
		if i<len(n.path){
			//先将原有路径节点分裂成一个子节点
			child := node{
				path: n.path[i:], //将除通用前缀之外的路径信息部分作为子节点
				wildChild: n.wildChild,
				indices: n.indices,
				nType: n.nType,
				priority: n.priority-1, //子节点优先级-1
				children: n.children,
				fullPath: n.fullPath,
			}
			//将儿子节点添加进
			n.children = []*node{&child}
			//更新公共前缀的最后一个字符
			n.indices = string([]byte{n.path[i]})
			n.path = path[:i]
			n.handlers = nil
			n.wildChild = false
			n.fullPath = fullPath[:parentFullPathIndex+i]
		}
		//将新加入的路径生成节点，加入刚分裂的父节点中作为子节点
		if i<len(path){
			path = path[i:]
			//取第一个字符，用来与indices做比较
			c := path[0]
			//若当前节点是参数节点且有一个子节点，则进行递归遍历
			if n.nType == param && c =='/' && len(n.children) == 1{
				parentFullPathIndex += len(n.path)
				n = n.children[0]
				n.priority++
				continue walk
			}
			//检查是否有现存子节点与传入路径相匹配，有则进入该节点进行递归
			for i,max := 0,len(n.indices); i<max; i++{
				if c == n.indices[i]{
					parentFullPathIndex += len(n.path)
					i = n.incrementChildPrio(i)
					n = n.children[i]
					continue walk
				}
			}
			//如果传入路径不是以":"、"*"开头，则为普通静态节点
			//直接构造后插入，并添加子节点索引
			if c != ':' && c != '*'{
				n.indices += string([]byte{c})
				child := &node{
					fullPath: fullPath,
				}
				n.children = append(n.children,child)
				n.incrementChildPrio(len(n.indices)-1)
				n = child
			}else if n.wildChild{//判断参数节点
				//插入通配符节点时，需要检查它是否与现有通配符冲突
				n = n.children[len(n.children)-1]
				n.priority++
				//检查当前路径是否还未遍历完
				if len(path) >= len(n.path) && n.path == path[:len(n.path)] {
					//若发现还有子路可遍历，则递归
					if (len(n.path) >= len(path)) || path[len(n.path)] == '/' {
						continue walk
					}
				}
				//通配字符冲突处理
				pathSeg := path
				if n.nType != catchAll{
					pathSeg = strings.SplitN(path, "/", 2)[0]
				}
				prefix := fullPath[:strings.Index(fullPath,pathSeg)] + n.path
				panic("'" + pathSeg +
					"' in new path '" + fullPath +
					"' conflicts with existing wildcard '" + n.path +
					"' in existing prefix '" + prefix +
					"'")

			}
			n.insertChild(path,fullPath,handlers)
			return
		}
		//如果当前节点已经有处理函数，则说明之前已经有注册过这个路由了，发出警告，并更新处理函数
		if n.handlers != nil{
			panic("handlers are already registered for path '" + fullPath + "'")
		}
		n.handlers = handlers
		return
	}

}

//搜索通配字符，并检查名称中是否存在无效字符,未找到通配符，返回-1作为索引
func findWildcard(path string) (wildcard string, i int, valid bool) {
	for start,c := range []byte(path){
		//通配字符一般":","*"
		if c != ':' && c != '*'{
			continue
		}
		//检查结尾并判断无效字符
		valid := true
		for end,c := range []byte(path[start+1:]){
			switch c {
			case '/':
				return path[start:start+1+end],start,valid
			case ':','*':
				valid = false
			}
		}
		return path[start:],start,valid
	}
	return "",-1,false
}

//添加节点的函数，主要处理包含参数的节点
func (n *node) insertChild(path string, fullPath string, handlers HandlersChain) {
	for  {
		//查找第一个通配符
		wildcard,i,valid := findWildcard(path)
		if i<0 {
			//没有找到通配符
			break
		}
		//通配符的名称必须包括":","*"
		if !valid{
			panic("only one wildcard per path segment is allowed, has: '" +
				wildcard + "' in path '" + fullPath + "'")
		}
		//检查通配符是否有名称
		if len(wildcard)<2{
			panic("wildcards must be named with a non-empty name in path '" + fullPath + "'")
		}
		//检查这个节点是否有已经存在的子节点
		//如果在这里插入通配符，子节点将无法访问
		if len(n.children)>0{
			panic("wildcard segment '" + wildcard +
				"' conflicts with existing children in path '" + fullPath + "'")
		}

		if wildcard[0] == ':'{ //参数节点
			if i>0 {
				//在当前通配符前插入前缀
				n.path = path[:i]
				path = path[i:]
			}
			n.wildChild = true
			child := &node{
				nType: param,
				path: wildcard,
				fullPath: fullPath,
			}
			n.children = []*node{child}
			n.wildChild = true
			n = child
			n.priority++

			//如果没有以通配字符结束，将有另一个以'/'开始的非通配符子路径
			if len(wildcard)<len(path){
				path = path[len(wildcard):]

				child := &node{
					priority: 1,
					fullPath: fullPath,
				}
				n.children = []*node{child}
				n = child //继续下一轮循环
				continue
			}
			//否则完成插入，将处理函数插入新子叶中
			n.handlers = handlers
			return
		}

		// 当节点为catchAll类型
		if i+len(wildcard) != len(path) {
			panic("catch-all routes are only allowed at the end of the path in path '" + fullPath + "'")
		}

		if len(n.path) > 0 && n.path[len(n.path)-1] == '/' {
			panic("catch-all conflicts with existing handle for the path segment root in path '" + fullPath + "'")
		}

		//'/'的当前固定宽度为1
		i--
		if path[i] != '/'{
			panic("no / before catch-all in path '" + fullPath + "'")
		}

		n.path = path[:i]

		//第一个节点，路径为空的catchAll节点
		child := &node{
			wildChild: true,
			nType:     catchAll,
			fullPath:  fullPath,
		}
		n.children = []*node{child}
		n.indices = string('/')
		n = child
		n.priority++

		//第二个节点，保存变量的节点
		child = &node{
			path: path[i:],
			nType: catchAll,
			handlers: handlers,
			priority: 1,
			fullPath: fullPath,
		}
		n.children = []*node{child}
		return
	}
	//如果没有找到通配符，只需要插入路径和句柄
	n.path = path
	n.handlers = handlers
	n.fullPath = fullPath
}

//nodeValue结构体用于保存领节点中的主要信息
type nodeValue struct {
	handlers HandlersChain
	params *Params
	tsr bool
	fullPath string
}

//搜索匹配路径的节点
func (n *node)getValue(path string,params *Params,unescape bool)(value nodeValue)  {
walk:
	for{
		//保存当前节点路径
		prefix := n.path
		//如果该路径与当前节点路径刚好匹配
		if prefix == path{
			//如果该路径的处理函数一样，则说明已经搜索过了，更新后直接跳出
			if value.handlers = n.handlers;value.handlers != nil{
				value.fullPath = n.fullPath
				return
			}
			//重定向的情况
			if path == "/" && n.wildChild && n.nType == root{
				//重定向后能找到满足条件的节点
				value.tsr = true
				return
			}
			//不满足上述条件时，根据索引搜索子节点
			indices := n.indices
			for i,max := 0,len(indices);i<max;i++{
				if indices[i] == '/'{
					n = n.children[i]
					value.tsr = len(n.path) == 1 && n.handlers != nil ||
						(n.nType == catchAll && n.children[0].handlers != nil)
					return
				}
			}
			return
		}
		//path的前缀与该节点吻合，进入子节点进行搜索
		if len(path) > len(prefix) && path[:len(prefix)] == prefix{
			path = path[len(prefix):]
			//若没有通配符子节点，则根据索引查找子节点
			if !n.wildChild{
				c := path[0]
				indices :=n.indices
				for i,max := 0,len(indices);i<max;i++{
					if c == indices[i]{
						n= n.children[i]
						continue walk
					}
				}
				//如果没有匹配的子节点，则重定向搜索
				value.tsr = path == "/" && n.handlers != nil
				return
			}
			//子节点是通配节点的情况
			//根据传入URL对路径参数进行解析
			//若wildChild为true，则n就只有一个子节点
			n = n.children[0]
			switch n.nType{
			//子节点为参数节点时
			case param:
				//寻找参数的字符长度
				end := 0
				for end < len(path) && path[end] != '/'{
					end++
				}
				//保存参数值
				if params != nil{
					if value.params == nil{
						value.params = params
					}
				}
				//在预先分配的容量内拓展切片
				i := len(*value.params)
				*value.params = (*value.params)[:i+1]
				val := path[:end]
				if unescape{
					if v,err := url.QueryUnescape(val);err == nil{
						val = v
					}
				}
				(*value.params)[i] = Param{
					Key: n.path[1:],
					Value: val,
				}
				//如果path还没有解析完成
				if end < len(path){
					//进入其他节点
					if len(n.children)>0{
						path = path[end:]
						n = n.children[0]
						continue walk
					}
					//若仅仅是多了一个'/',则重定向
					value.tsr = len(path) == end +1
				}

				if value.handlers = n.handlers;value.handlers != nil{
					value.fullPath = n.fullPath
					return
				}

				if len(n.children) == 1{
					//若果子节点有匹配'/',则重定向
					n = n.children[0]
					value.tsr = n.path == "/" && n.handlers != nil
				}
				return

				//这个类型表明所有参数都已匹配完
			case catchAll:
				//保存参数值
				if params != nil{
					if value.params == nil{
						value.params = params
					}
				}
				//在预先分配的容量内拓展切片
				i := len(*value.params)
				*value.params = (*value.params)[:i+1]
				val := path
				if unescape{
					if v,err := url.QueryUnescape(val);err == nil{
						val = v
					}
				}
				(*value.params)[i] = Param{
					Key: n.path[2:],
					Value: val,
				}
				value.handlers = n.handlers
				value.fullPath = n.fullPath
				return
			default:
				panic("invalid node type")
			}
		}
		//推荐重定向
		 value.tsr = (path == "/") ||
			(len(prefix) == len(path)+1 && prefix[len(path)] == '/' &&
				path == prefix[:len(prefix)-1] && n.handlers != nil)
		 return
	}//for

}

//对给定路径进行不区分大小写的查找并尝试查找处理程序 
//它还可以选择修复尾部斜杠,返回大小写更正的路径和指示查找是否成功的bool值
func (n *node)findCaseInsensitivePath(path string,fixTrailingSlash bool)([]byte,bool)  {
	const stackBufSize = 128
	//分配缓冲区域
	buf := make([]byte,0,stackBufSize)
	if length := len(path)+1; length>stackBufSize{
		buf = make([]byte,0,length)
	}
	//[4]byte{}作用是清空缓冲区
	ciPath := n.findCaseInsensitivePathRec(path,buf,[4]byte{},fixTrailingSlash)

	return ciPath,ciPath!=nil
}

func (n *node) findCaseInsensitivePathRec(path string, ciPath []byte, rb [4]byte, fixTrailingSlash bool) []byte {
	return nil
}