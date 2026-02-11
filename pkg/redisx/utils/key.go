package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultSeparator = ":"
)

type KeyBuilder struct {
	prefix     string
	separator  string
	namespace  string
}

func NewKeyBuilder(prefix string) *KeyBuilder {
	return &KeyBuilder{
		prefix:    prefix,
		separator: DefaultSeparator,
	}
}

func (kb *KeyBuilder) WithSeparator(separator string) *KeyBuilder {
	kb.separator = separator
	return kb
}

func (kb *KeyBuilder) WithNamespace(namespace string) *KeyBuilder {
	kb.namespace = namespace
	return kb
}

func (kb *KeyBuilder) Build(parts ...string) string {
	var result []string
	if kb.namespace != "" {
		result = append(result, kb.namespace)
	}
	if kb.prefix != "" {
		result = append(result, kb.prefix)
	}
	result = append(result, parts...)
	return strings.Join(result, kb.separator)
}

func (kb *KeyBuilder) BuildWithID(id string) string {
	return kb.Build(id)
}

func (kb *KeyBuilder) BuildWithType(keyType, id string) string {
	return kb.Build(keyType, id)
}

func (kb *KeyBuilder) BuildWithTimestamp(parts ...string) string {
	result := append(parts, strconv.FormatInt(time.Now().Unix(), 10))
	return kb.Build(result...)
}

func (kb *KeyBuilder) BuildHash(key string) string {
	hash := md5.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}

func BuildKey(prefix string, parts ...string) string {
	if prefix == "" {
		return strings.Join(parts, DefaultSeparator)
	}
	allParts := append([]string{prefix}, parts...)
	return strings.Join(allParts, DefaultSeparator)
}

func BuildNamespaceKey(namespace, prefix string, parts ...string) string {
	if namespace == "" {
		return BuildKey(prefix, parts...)
	}
	allParts := append([]string{namespace, prefix}, parts...)
	return strings.Join(allParts, DefaultSeparator)
}

func BuildUserKey(userID string, parts ...string) string {
	allParts := append([]string{"user", userID}, parts...)
	return strings.Join(allParts, DefaultSeparator)
}

func BuildSessionKey(sessionID string) string {
	return BuildKey("session", sessionID)
}

func BuildLockKey(resource string) string {
	return BuildKey("lock", resource)
}

func BuildCacheKey(category, key string) string {
	return BuildKey("cache", category, key)
}

func BuildCounterKey(name string) string {
	return BuildKey("counter", name)
}

func BuildRateLimitKey(identifier string) string {
	return BuildKey("ratelimit", identifier)
}

func BuildQueueKey(queueName string) string {
	return BuildKey("queue", queueName)
}

func BuildSetKey(setName string) string {
	return BuildKey("set", setName)
}

func BuildZSetKey(zsetName string) string {
	return BuildKey("zset", zsetName)
}

func BuildHashKey(hashName string) string {
	return BuildKey("hash", hashName)
}

func BuildStreamKey(streamName string) string {
	return BuildKey("stream", streamName)
}

func ParseKey(key string, separator string) []string {
	if separator == "" {
		separator = DefaultSeparator
	}
	return strings.Split(key, separator)
}

func GetKeyPrefix(key string, separator string) string {
	parts := ParseKey(key, separator)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func GetKeySuffix(key string, separator string) string {
	parts := ParseKey(key, separator)
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func MatchPattern(pattern, key string) bool {
	patternParts := ParseKey(pattern, DefaultSeparator)
	keyParts := ParseKey(key, DefaultSeparator)

	if len(patternParts) != len(keyParts) {
		return false
	}

	for i := 0; i < len(patternParts); i++ {
		if patternParts[i] != "*" && patternParts[i] != keyParts[i] {
			return false
		}
	}

	return true
}

func GenerateUniqueKey(prefix string) string {
	timestamp := time.Now().UnixNano()
	return BuildKey(prefix, fmt.Sprintf("%d", timestamp))
}

func BuildVersionedKey(key string, version int) string {
	return fmt.Sprintf("%s:v%d", key, version)
}

func GetKeyVersion(key string) (int, error) {
	parts := strings.Split(key, ":v")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid versioned key format")
	}
	version, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}
	return version, nil
}
