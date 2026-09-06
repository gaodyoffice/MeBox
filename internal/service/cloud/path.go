package cloud

import "strings"

// IsCloudPath 判断路径是否为云盘路径（带 cloud:// 前缀）。
func IsCloudPath(path string) bool {
	return strings.HasPrefix(path, "cloud://")
}

// ParseCloudPath 解析 cloud://provider/path 格式，返回 (provider, path, ok)。
// 示例：cloud://openlist/电影/xxx.mkv → ("openlist", "/电影/xxx.mkv", true)
func ParseCloudPath(raw string) (provider, path string, ok bool) {
	if !IsCloudPath(raw) {
		return "", "", false
	}
	rest := strings.TrimPrefix(raw, "cloud://")
	if rest == "" {
		return "", "", false
	}
	// 按首个 "/" 分割 provider 和 path
	idx := strings.Index(rest, "/")
	if idx < 0 {
		return rest, "", true
	}
	return rest[:idx], rest[idx:], true
}

// BuildCloudPath 构建 cloud://provider/path 格式。
func BuildCloudPath(provider, path string) string {
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return "cloud://" + provider
	}
	return "cloud://" + provider + "/" + path
}

// CloudPathPrefix 返回 cloud://provider 前缀（用于路径替换）。
func CloudPathPrefix(provider string) string {
	return "cloud://" + provider
}

// ReplaceCloudPathPrefix 将路径的前缀替换为目标前缀。
// 例：cloud://openlist/下载目录/xxx.mkv → cloud://openlist/媒体库/xxx.mkv
func ReplaceCloudPathPrefix(srcPath, srcPrefix, dstPrefix string) string {
	rel := strings.TrimPrefix(srcPath, srcPrefix)
	return dstPrefix + rel
}
