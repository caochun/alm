package domain

import (
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"time"
)

// GenerateID 生成唯一ID
func GenerateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GetCurrentTime 获取当前时间
func GetCurrentTime() time.Time {
	return time.Now()
}

// JoinPath 连接路径
func JoinPath(elem ...string) string {
	return filepath.Join(elem...)
}

