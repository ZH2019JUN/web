package gin

import (
	"fmt"
	"strings"
)

//处理错误信息

func IsDebugging() bool {
	return ginMode == debugCode
}


//新建Engine对象时的错误处理函数
func debugPrintWARNINGNew() {
	debugPrint(`[WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:	export GIN_MODE=release
 - using code:	gin.SetMode(gin.ReleaseMode)

`)
}

// 调试日志输出格式
var DebugPrintRouteFunc func(httpMethod, absolutePath, handlerName string, nuHandlers int)

//打印配置的路由信息
func debugPrintRoute(httpMethod, absolutePath string, handlers HandlersChain) {
	if IsDebugging() {
		nuHandlers := len(handlers)
		handlerName := nameOfFunction(handlers.Last())
		if DebugPrintRouteFunc == nil {
			debugPrint("%-6s %-25s --> %s (%d handlers)\n", httpMethod, absolutePath, handlerName, nuHandlers)
		} else {
			DebugPrintRouteFunc(httpMethod, absolutePath, handlerName, nuHandlers)
		}
	}
}

func debugPrint(format string, values ...interface{}) {
	if IsDebugging() {
		if !strings.HasSuffix(format, "\n") {
			format += "\n"
		}
		fmt.Fprintf(DefaultWriter, "[GIN-debug] "+format, values...)
	}
}

func debugPrintError(err error) {
	if err != nil {
		if IsDebugging() {
			fmt.Fprintf(DefaultErrorWriter, "[GIN-debug] [ERROR] %v\n", err)
		}
	}
}