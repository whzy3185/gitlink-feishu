package output

import (
	"os"
)

// ANSI 颜色码（提升终端输出体验，彩色化）
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

// useColor 判断是否启用彩色输出（非终端/管道时关闭，避免乱码）
var useColor = shouldUseColor()

func shouldUseColor() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	// 只在交互式终端启用彩色
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// colorize 给字符串加颜色（非终端时原样返回）
func colorize(s, color string) string {
	if !useColor {
		return s
	}
	return color + s + colorReset
}

// 便捷函数
func red(s string) string    { return colorize(s, colorRed) }
func green(s string) string  { return colorize(s, colorGreen) }
func yellow(s string) string { return colorize(s, colorYellow) }
func cyan(s string) string   { return colorize(s, colorCyan) }
func bold(s string) string   { return colorize(s, colorBold) }
func dim(s string) string    { return colorize(s, colorDim) }

// colorForKey 根据字段名给值着色（状态/ok 等用语义色）
func colorForKey(key, val string) string {
	switch key {
	case "ok":
		if val == "true" {
			return green("✓ " + val)
		}
		return red("✗ " + val)
	case "status", "state":
		switch val {
		case "open", "opened", "active":
			return green(val)
		case "closed", "merged":
			return cyan(val)
		case "failed", "error":
			return red(val)
		default:
			return yellow(val)
		}
	default:
		return val
	}
}
