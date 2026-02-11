package hashx

import (
	"crypto/md5"
	"encoding/hex"

	"github.com/spaolacci/murmur3"
	"golang.org/x/crypto/bcrypt"
)

// Hash returns the hashx value of data.
// 一致性hash | 布隆过滤器
func Hash(data []byte) uint64 {
	return murmur3.Sum64(data)
}

// Md5 returns the md5 bytes of data.
func Md5(data []byte) []byte {
	digest := md5.New()
	digest.Write(data)
	return digest.Sum(nil)
}

// Md5Hex returns the md5 hex string of data.
// This function is optimized for better performance than fmt.Sprintf.
func Md5Hex(data []byte) string {
	return hex.EncodeToString(Md5(data))
}

// BcryptHash 使用 bcrypt 对密码进行加密
// 注意：bcrypt 算法会自动生成随机盐值并将其与哈希值一起存储，无需手动管理盐值
func BcryptHash(password string) string {
	bytes, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes)
}

// BcryptCheck 对比明文密码和数据库的哈希值
// 注意：bcrypt 在哈希值中包含了盐值信息，因此能够正确验证密码
func BcryptCheck(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// MD5WithSalt 计算给定数据的MD5哈希值，支持可选的附加数据（类似盐值的作用）
// 注意：MD5算法已不推荐用于安全相关场景，仅适用于数据校验等非安全场景
// @function: MD5WithSalt
// @description: 计算MD5哈希值，可附加额外数据
// @param: data []byte 要计算哈希的主数据
// @param: salt ...byte 可选的附加数据（类似盐值）
// @return: string MD5哈希的十六进制表示
func MD5WithSalt(data []byte, salt ...byte) string {
	h := md5.New()
	h.Write(data)                          // 先写入主要数据
	return hex.EncodeToString(h.Sum(salt)) // 然后将可选的 salt 数据追加到哈希中
}
