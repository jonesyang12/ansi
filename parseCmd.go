package ansi

import (
	"strings"
)

type CmdParse struct {
	CmdOutput  chan []map[*S][]*S
	InOutMap   []map[*S][]*S
	RecordChan chan string
	Cmd        *S
	CmdResult  map[*S]*S
}

func NewCmdParse() *CmdParse {
	c := new(CmdParse)
	c.CmdOutput = make(chan []map[*S][]*S)
	c.InOutMap = make([]map[*S][]*S, 0)
	c.RecordChan = make(chan string)
	c.CmdResult = make(map[*S]*S)
	return c
}

// 从outChannel读取数据
func receivedCmdOutChan(c *CmdParse) {
	for inOutSlice := range c.CmdOutput {
		readyParse := make([]*S, 0)
		cmdMap := make(map[*S]*S)
		for _, inOutMap := range inOutSlice {
			for in, outSlice := range inOutMap {
				for _, out := range outSlice {
					if in.Code == "\r" {
						if !notInVim(outSlice) {
							readyParse = []*S{}
						}
					} else {
						readyParse = append(readyParse, in)
						cmdMap[in] = out
					}
				}

			}
		}

		udCmd := parseUpDown(readyParse, cmdMap)
		tabCmd := parseTab(udCmd, cmdMap)
		lCmd, rCmd := parseShortCut(tabCmd)
		lrCmd := parseLeftRight(lCmd, rCmd)
		dCmd := parseDelete(lrCmd)
		//for _, v := range dCmd {
		//	fmt.Printf("---dCmd---%#v\n", v)
		//}

		cmd := ""
		for _, v := range dCmd {
			cmd += string(v.Code)
		}
		if cmd != "" {
			c.RecordChan <- cmd
		}
	}
}

// 将命令字符串转换结构体
func str2Struct(s string) []*S {
	result := make([]*S, 0)
	for _, appendCmd := range s {
		newS := new(S)
		newS.Code = Name(string(appendCmd))
		result = append(result, newS)
	}
	return result
}

// 是否在vim状态
func notInVim(a []*S) bool {
	if len(a) == 1 && a[0].Code == "\r\n" {
		return true
	}
	if len(a) >= 2 {
		t1 := contains(a, "\r\n")
		t2 := contains(a, SM)
		return t1 && t2
	}

	return false
}

// contains 切片是否包含某元素
func contains(s []*S, params Name) bool {
	for _, v := range s {
		if v.Code == params {
			return true
		}
	}
	return false
}

// insertByIndex 指定位置插入
func insertByIndex(slice []*S, index int, element *S) []*S {
	slice = append(slice[:index], append([]*S{element}, slice[index:]...)...)
	return slice

}

// parseUpDown 解析上下键,历史命令
func parseUpDown(s []*S, m map[*S]*S) []*S {
	result := make([]*S, 0)
	//获取最后一个
	var last *S
	for i := len(s) - 1; i >= 0; i-- {
		if s[i].Code == CUD || s[i].Code == CUU {
			last = s[i]
			cmdS := m[last]
			cmd := strings.TrimLeft(string(cmdS.Code), "\b")
			//完整命令解析拆分字节流形式
			lis := str2Struct(cmd)
			result = append(append(result, lis...), s[i+1:]...)
			return result
		}
	}
	return s
}

// parseTab 解析tab键 命令补充
func parseTab(s []*S, m map[*S]*S) []*S {
	result := make([]*S, 0)
	for _, v := range s {
		if v.Code == "\t" {
			outCmd := m[v]
			outCmdStr := string(outCmd.Code)
			switch {
			//情形一:有多个结果展示,但不补全
			case strings.Contains(outCmdStr, "\r\n"):
				continue
			//情形二:后面有字符的补全 eg:j/t(obs)h (命令不能是中文)
			case strings.HasSuffix(outCmdStr, "\b"):
				i := 0
				for _, v := range outCmdStr {
					if string(v) == "\b" {
						i += 1
					}
				}
				outCmdStr = outCmdStr[:len(outCmdStr)-i*2]
				tmplist := str2Struct(outCmdStr)
				result = append(result, tmplist...)
			//和情形二一样,只是多余\a字符
			case strings.HasPrefix(outCmdStr, "\a"):
				outCmdStr = strings.TrimPrefix(outCmdStr, "\a")
				tmplist := str2Struct(outCmdStr)
				result = append(result, tmplist...)
			default:
				//情形三:直接补全
				result = append(result, outCmd)

			}
		} else {
			result = append(result, v)
		}
	}

	return result
}

// parseShortCut 解析快捷键
func parseShortCut(s []*S) ([]*S, []*S) {
	result1 := make([]*S, 0)
	result2 := make([]*S, 0)
	for _, v := range s {
		switch v.Code {
		case "\x01": //CTRL A
			result2 = result1
			result1 = []*S{}
		default:
			result1 = append(result1, v)
		}
	}
	return result1, result2
}

// parseLeftRight 包含左右键时解析
func parseLeftRight(lc []*S, rc []*S) []*S {
	result := make([]*S, 0)
	cursor := 0
	p := 0
	for _, v := range lc {
		switch {
		//左键
		case v.Code == CUB:
			cursor -= 1
		//右键
		case v.Code == CUF:
			//在行尾就不移动
			if len(rc) > p {
				result = append(result, rc[p])
				p += 1
			}
			cursor += 1
		default:
			result = insertByIndex(result, cursor, v)
			cursor = cursor + 1
		}
	}
	if len(rc) > p {
		result = append(result, rc[p:]...)
	}
	return result
}

// parseDelete 解析删除键
func parseDelete(s []*S) []*S {
	result := make([]*S, 0)
	for _, v := range s {
		if v.Code == "\x7f" && (len(result)-1) >= 0 {
			result = result[:len(result)-1]
		} else {
			result = append(result, v)
		}
	}
	return result
}
