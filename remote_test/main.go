package main

import (
	"fmt"
	
	"github.com/tedwangl/go-util/pkg/utils"
)

func main() {
	text := "Testing remote import!"
	
	// 测试Reverse函数
	reversed := utils.Reverse(text)
	fmt.Printf("Original: %s\n", text)
	fmt.Printf("Reversed: %s\n", reversed)
	
	// 测试ToUpper函数
	upper := utils.ToUpper(text)
	fmt.Printf("Uppercase: %s\n", upper)
	
	// 测试ToLower函数
	lower := utils.ToLower(text)
	fmt.Printf("Lowercase: %s\n", lower)
}