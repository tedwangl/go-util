package utils

import (
	"strings"
)

// Reverse 反转字符串
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// ToUpper 将字符串转换为大写
func ToUpper(s string) string {
	return strings.ToUpper(s)
}

// ToLower 将字符串转换为小写
func ToLower(s string) string {
	return strings.ToLower(s)
}

// CountWords 计算字符串中的单词数
func CountWords(s string) int {
	if s == "" {
		return 0
	}
	return len(strings.Fields(s))
}