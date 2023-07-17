package ansi

import (
	"strings"
)

type CmdParse struct {
	CmdInput chan []*S
	//CmdOutput chan map[*S]*S
	RecordChan chan string
	Cmd        *S
	Exec       []*S
	CmdResult  map[*S]*S
}

func NewCmdParse() *CmdParse {
	c := new(CmdParse)
	c.CmdInput = make(chan []*S)
	//c.CmdOutput = make(chan map[*S]*S, 1024)
	c.RecordChan = make(chan string)
	c.Exec = make([]*S, 0)
	c.CmdResult = make(map[*S]*S)
	return c
}

// GetPs2 获取prompt
func (c *CmdParse) GetPs2(a *S) string {
	code := c.CmdResult[a].Code
	ps1Slice := strings.Split(string(code), "\r\n\r\n")
	return ps1Slice[len(ps1Slice)-1]
}

func ReceivedCmdInChan(c *CmdParse) {
	for cmds := range c.CmdInput {
		//for k, cmd := range cmds {
		//	fmt.Printf("--input--%v-%#v\n", k, cmd.Code)
		//}
		if len(cmds) != 0 {
			// 换行符分割命令
			cmdsp := splitElement(cmds, "\r")
			parseCmd := cmdsp[0]
			//CTRLC不记录
			if contains(parseCmd, "\x03") {
				continue
			}
			//优先级: ↑↓历史命令 -> tab命令提示 -> 快捷键 -> ←→左右光标移动 -> 删除
			res1 := parseUpDown(parseCmd, c)
			res2 := parseTab(res1, c)

			res3, oapres3 := parseShortCut(res2)
			//先解析oapres3中的命令
			napres3 := parseLeftRight(oapres3)
			apres3 := parseDelete(napres3)

			res4 := parseLeftRight2(res3, apres3)
			res5 := parseDelete(res4)
			cmd := ""
			for _, v := range res5 {
				cmd += string(v.Code)
			}
			cmd = strings.TrimLeft(cmd, " ")
			c.RecordChan <- cmd

		}

	}
}

// parseTab 解析tab键 命令补充
func parseTab(s []*S, c *CmdParse) []*S {
	result := make([]*S, 0)
	//获取最后一个
	var last *S
	for i := len(s) - 1; i >= 0; i-- {
		if s[i].Code == "\t" {
			last = s[i]
			ps1 := c.GetPs2((*S)(nil))
			splitCmd := strings.Split(string(c.CmdResult[last].Code), ps1)
			tabCmd := splitCmd[len(splitCmd)-1]
			for _, appendCmd := range tabCmd {
				newS := new(S)
				newS.Code = Name(string(appendCmd))
				result = append(result, newS)
			}
			result = append(result, s[i+1:]...)
			return result
		}
	}
	return s
}

// splitElement 切片分割
func splitElement(slice []*S, ele Name) [][]*S {
	var result [][]*S
	var subSlice []*S

	for _, v := range slice {
		if v.Code == ele {
			result = append(result, subSlice)
			subSlice = nil
		} else {
			subSlice = append(subSlice, v)
		}
	}

	if subSlice != nil {
		result = append(result, subSlice)
	}

	return result
}

// parseLeftRight 包含左右键时解析
func parseLeftRight(s []*S) []*S {
	result := make([]*S, 0, len(s))
	cursor := 0
	if contains(s, CUB) || contains(s, CUF) {
		for _, v := range s {
			l := len(result)
			switch {
			//左键
			case v.Code == CUB:
				//在行首就不移动
				if cursor > 0 {
					cursor -= 1
				}
			//右键
			case v.Code == CUF:
				//在行尾就不移动
				if cursor < l {
					cursor += 1
				}
			default:
				result = insertByIndex(result, cursor, v)
				cursor = cursor + 1
			}
		}
		return result
	}
	return s
}

// parseLeftRight 包含左右键时解析
func parseLeftRight2(s []*S, ns []*S) []*S {
	result := make([]*S, 0, len(s))
	cursor := 0
	p := 0
	for _, v := range s {
		switch {
		//排除行首无用干扰的删除信号
		case v.Code == "\x7f" && cursor == 0:
		//左键
		case v.Code == CUB:
			//在行首就不移动
			if cursor > 0 {
				cursor -= 1
			}
		//右键
		case v.Code == CUF:
			//在行尾就不移动
			if len(ns) > p {
				result = append(result, ns[p])
				p += 1
			}
			if cursor < len(result) {
				cursor += 1
			}
		default:
			result = insertByIndex(result, cursor, v)
			cursor = cursor + 1
		}
	}
	if len(ns) > p {
		result = append(result, ns[p:]...)
	}
	return result
}

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

// parseUpDown 解析上下键,历史命令
func parseUpDown(s []*S, c *CmdParse) []*S {
	result := make([]*S, 0)
	//获取最后一个
	var last *S
	for i := len(s) - 1; i >= 0; i-- {
		if s[i].Code == CUD || s[i].Code == CUU {
			last = s[i]
			outCmd := strings.Trim(string(c.CmdResult[(*S)(last)].Code), "\r")
			ps1 := c.GetPs2((*S)(nil))
			splitCmd := strings.ReplaceAll(outCmd, ps1, "")
			for _, appendCmd := range splitCmd {
				newS := new(S)
				newS.Code = Name(string(appendCmd))
				result = append(result, newS)
			}
			result = append(result, s[i+1:]...)
			return result
		}
	}
	return s
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

// insertByIndex 指定位置插入
func insertByIndex(slice []*S, index int, element *S) []*S {
	slice = append(slice[:index], append([]*S{element}, slice[index:]...)...)
	return slice

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
