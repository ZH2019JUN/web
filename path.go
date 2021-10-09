package gin

//cleanPath 是 path.Clean 的 URL 版本，它返回 p 的规范 URL 路径，消除 . 和.. 元素。
// 迭代应用以下规则，直到无法进行进一步处理：
// 1. 用一个斜杠替换多个斜杠。
// 2. 消除每个 . 路径名元素（当前目录）。
// 3. 消除每个内部 .. 路径名元素（父目录）以及它之前的非 .. 元素。
// 4.删除以root开头的..元素：也就是说，在路径的开头将“ / ..”替换为“ /”。

func cleanPath(p string) string  {
	const stackBufSize = 128  //预先分配内存
	if p == "/"{
		return "/"
	}
	buf := make([]byte,0,stackBufSize)

	n := len(p)

	//r是要处理的下一个字符的索引
	r := 1
	//w是要写入buf的下一个字符的索引
	w := 1

	//路径必须以'/'作为开头
	if p[0] != '/'{
		r = 0
		if n>stackBufSize{
			buf = make([]byte,n+1)
		}else {
			buf = buf[:n]
		}
		buf[0] = '/'
	}

	trailing := n > 1 && p[n-1] == '/'
	
	//循环修正路径
	for r<n{
		switch  {
		case p[r]=='/':
			r++
		case p[r]=='.' && r+1==n:
			trailing = true
			r++
		case p[r]=='.' && p[r+1]=='/':
			r+=2
		case p[r] == '.' && p[r+1] == '.' && (r+2 == n || p[r+2] == '/'):
			r+=3

			if w > 1 {

				w--

				if len(buf) == 0 {
					for w > 1 && p[w] != '/' {
						w--
					}
				} else {
					for w > 1 && buf[w] != '/' {
						w--
					}
				}
			}

		default:

			if w > 1 {
				bufApp(&buf, p, w, '/')
				w++
			}


			for r < n && p[r] != '/' {
				bufApp(&buf, p, w, p[r])
				w++
				r++
			}

		}
	}
	if trailing && w > 1 {
		bufApp(&buf, p, w, '/')
		w++
	}

	if len(buf) == 0 {
		return p[:w]
	}
	return string(buf[:w])
}

func bufApp(buf *[]byte, s string, w int, c byte){
	b := *buf
	if len(b) == 0 {

		if s[w] == c {
			return
		}

		length := len(s)
		if length > cap(b) {
			*buf = make([]byte, length)
		} else {
			*buf = (*buf)[:length]
		}
		b = *buf

		copy(b, s[:w])
	}
	b[w] = c
}
