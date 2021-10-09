package gin

//实用程序
import (
	"os"
	"path"
	"reflect"
	"runtime"
)

//H是创建map[string]interface{}的快捷方式
type H map[string] interface {}

//判断路径，并输出错误信息
func assert1(guard bool,text string)  {
	if !guard{
		panic(text)
	}
}


//输出路径信息最后一位
func lastChar(str string) uint8  {
	if str == ""{
		panic("the length of the string can't be 0")
	}
	return str[len(str)-1]
}

//添加路径
func joinPaths(absolutePath,relativePath string) string{
	if relativePath == ""{
		return absolutePath
	}
	finalPath := path.Join(absolutePath,relativePath)
	if lastChar(relativePath) == '/'&&lastChar(finalPath) != '/'{
		return finalPath + "/"
	}
	return finalPath
}

//解析地址
func resolveAddress(addr []string) string {
	switch len(addr) {
	case 0:
		if port := os.Getenv("PORT"); port != "" {
			debugPrint("Environment variable PORT=\"%s\"", port)
			return ":" + port
		}
		debugPrint("Environment variable PORT is undefined. Using port :8080 by default")
		return ":8080"
	case 1:
		return addr[0]
	default:
		panic("too many parameters")
	}
}

func nameOfFunction(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

