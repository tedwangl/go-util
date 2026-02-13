package storage

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// HashURL 计算 URL 哈希（MD5）
func HashURL(url string) string {
	hash := md5.Sum([]byte(url))
	return hex.EncodeToString(hash[:])
}

// HashContent 计算内容哈希（SHA256）
func HashContent(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// ShouldSkipTask 判断任务是否应该跳过
func ShouldSkipTask(storage Storage, url string, strategy DuplicateStrategy) (bool, *Task, error) {
	switch strategy {
	case DuplicateStrategyURL:
		// URL 精确匹配
		task, err := storage.GetTaskByURL(url)
		if err == nil {
			// 已存在且已完成或运行中，跳过
			if task.Status == TaskStatusCompleted || task.Status == TaskStatusRunning {
				return true, task, nil
			}
		}
		return false, nil, nil

	case DuplicateStrategyURLHash:
		// URL 哈希匹配
		urlHash := HashURL(url)
		task, err := storage.GetTaskByURLHash(urlHash)
		if err == nil {
			if task.Status == TaskStatusCompleted || task.Status == TaskStatusRunning {
				return true, task, nil
			}
		}
		return false, nil, nil

	case DuplicateStrategyNone:
		// 不去重
		return false, nil, nil

	default:
		return false, nil, fmt.Errorf("不支持的去重策略: %s", strategy)
	}
}

// ShouldSkipItem 判断内容是否应该跳过
func ShouldSkipItem(storage Storage, contentHash string) (bool, *Item, error) {
	if contentHash == "" {
		return false, nil, nil
	}

	item, err := storage.GetItemByContentHash(contentHash)
	if err == nil {
		// 已存在且已保存，跳过
		if item.Status == ItemStatusSaved {
			return true, item, nil
		}
	}
	return false, nil, nil
}
