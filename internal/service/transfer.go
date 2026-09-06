// Package service — file transfer strategies for the organizer.
//
// 整理媒体时支持四种转移方式：
//
//	move     移动（同盘 rename，跨盘 copy+删除源）——会移除源文件
//	copy     复制（保留源文件）
//	hardlink 硬链接（同盘零额外占用，保留源文件；做种不受影响）
//	symlink  软链接（保留源文件，指向源）
//
// 除 move 外，其余方式都保留源文件，因此下载器仍能在原
// 路径找到数据继续做种上传。
package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/truewhile/MeBox/internal/service/cloud"
)

var linkFile = os.Link

// TransferMode 表示整理时文件的转移方式。
type TransferMode string

const (
	// TransferMove 移动：同盘 rename，跨盘 copy+删除源。
	TransferMove TransferMode = "move"
	// TransferCopy 复制：保留源文件。
	TransferCopy TransferMode = "copy"
	// TransferHardlink 硬链接：同盘零额外占用并保留源文件，做种不受影响。
	TransferHardlink TransferMode = "hardlink"
	// TransferSymlink 软链接：保留源文件，目标指向源。
	TransferSymlink TransferMode = "symlink"
)

// parseTransferMode 解析转移方式字符串，无法识别时回退为默认的移动。
func parseTransferMode(s string) TransferMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "copy", "复制":
		return TransferCopy
	case "hardlink", "hard", "link", "硬链接", "硬连接":
		return TransferHardlink
	case "symlink", "soft", "softlink", "软链接", "软连接", "符号链接":
		return TransferSymlink
	default:
		return TransferMove
	}
}

// keepsSource 报告该转移方式是否会保留源文件（用于做种）。
func (m TransferMode) keepsSource() bool {
	return m == TransferCopy || m == TransferHardlink || m == TransferSymlink
}

// transferFile 按指定方式把 src 转移到 dst。
// dst 已存在时一律报错，绝不覆盖（防止不同 release 改名后互相覆盖）。
func transferFile(src, dst string, mode TransferMode) error {
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("destination already exists: %s", dst)
	}
	switch mode {
	case TransferCopy:
		return copyFile(src, dst)
	case TransferHardlink:
		if err := linkFile(src, dst); err != nil {
			// Docker 部署里下载目录和媒体目录往往是两个独立的 bind mount，
			// 即使在宿主机上同属一块盘，容器内 os.Link 也会因跨文件系统
			// (EXDEV) 失败。hardlink 模式必须保持零额外数据占用语义，不能
			// 自动降级为复制；需要复制时请显式选择 copy。
			return fmt.Errorf("hardlink failed: %w; source and target must be on the same filesystem/subvolume from inside the container. If you selected move, disable keep_seeding first because keep_seeding upgrades move to hardlink; choose copy if you want to keep seeding across mounts", err)
		}
		return nil
	case TransferSymlink:
		target := src
		if abs, err := filepath.Abs(src); err == nil {
			target = abs
		}
		return os.Symlink(target, dst)
	default: // TransferMove
		return moveFile(src, dst)
	}
}

// copyFile 流式复制 src 到 dst（保留源文件）。O_EXCL 保证不覆盖已存在目标。
func copyFile(src, dst string) error {
	in, err := os.Open(src) // #nosec G304 -- src is selected from configured media/download roots by the organizer.
	if err != nil {
		return err
	}
	defer in.Close()
	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644) // #nosec G304,G302 -- dst is organizer-generated; media files must remain readable by local players.
	if err != nil {
		return err
	}
	if _, werr := io.Copy(f, in); werr != nil {
		_ = f.Close()
		_ = os.Remove(dst)
		return werr
	}
	return f.Close()
}

// moveFile tries os.Rename first (instant on same fs), then falls back
// to copy + remove for cross-device moves.
//
// If dst already exists, moveFile returns an error instead of overwriting it.
// OrganizeMedia checks this before calling transferFile; this remains the
// second line of defense against different releases collapsing to one name.
func moveFile(src, dst string) error {
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("destination already exists: %s", dst)
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src) // #nosec G304 -- src is selected from configured media/download roots by the organizer.
	if err != nil {
		return err
	}
	defer in.Close()
	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644) // #nosec G304,G302 -- dst is organizer-generated; media files must remain readable by local players.
	if err != nil {
		return err
	}
	if _, werr := io.Copy(f, in); werr != nil {
		_ = f.Close()
		_ = os.Remove(dst)
		return werr
	}
	if cerr := f.Close(); cerr != nil {
		return cerr
	}
	return os.Remove(src)
}

func transferDirectory(src, dst string, mode TransferMode) error {
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("destination already exists: %s", dst)
	}
	switch mode {
	case TransferSymlink:
		return transferFile(src, dst, mode)
	case TransferMove:
		if err := os.Rename(src, dst); err == nil {
			return nil
		}
		if err := transferDirectoryTree(src, dst, TransferCopy); err != nil {
			return err
		}
		return os.RemoveAll(src)
	case TransferHardlink, TransferCopy:
		return transferDirectoryTree(src, dst, mode)
	default:
		return transferDirectory(src, dst, TransferMove)
	}
}

func transferDirectoryTree(src, dst string, mode TransferMode) error {
	if err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o755) // #nosec G301 -- media folders must remain readable by local players.
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755) // #nosec G301 -- media folders must remain readable by local players.
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { // #nosec G301 -- media folders must remain readable by local players.
			return err
		}
		return transferFile(path, target, mode)
	}); err != nil {
		_ = os.RemoveAll(dst)
		return err
	}
	return nil
}

// ─── 云盘文件操作 ─────────────────────────────────────────────────────────────

// CloudOverwriteMode 云盘文件覆盖策略。
type CloudOverwriteMode string

const (
	CloudOverwriteAlways CloudOverwriteMode = "always" // 总是覆盖
	CloudOverwriteSize   CloudOverwriteMode = "size"   // 按大小覆盖（新文件更大时覆盖，默认）
	CloudOverwriteNever  CloudOverwriteMode = "never"  // 从不覆盖
	CloudOverwriteLatest CloudOverwriteMode = "latest" // 仅保留最新版本
)

// CloudMovableProvider 云盘移动接口（由 cloud.MovableProvider 实现）。
type CloudMovableProvider interface {
	List(ctx context.Context, dirID string) ([]cloud.FileEntry, error)
	Move(ctx context.Context, ref, targetDir, name string) (*cloud.FileEntry, error)
}

// CheckCloudOverwrite 检查云盘文件覆盖策略。
// 返回 true 表示应该覆盖/移动，false 表示跳过。
func CheckCloudOverwrite(srcEntry, dstEntry *cloud.FileEntry, mode CloudOverwriteMode) bool {
	if dstEntry == nil {
		// 目标不存在，直接移动
		return true
	}
	switch mode {
	case CloudOverwriteAlways:
		return true
	case CloudOverwriteSize:
		// 新文件更大时覆盖
		return srcEntry.Size > dstEntry.Size
	case CloudOverwriteLatest:
		// 保留最新版本（总是覆盖）
		return true
	default: // CloudOverwriteNever
		return false
	}
}

// TransferCloudMove 云盘文件移动（同盘）。
// srcRef: 源文件引用（路径或 ID）
// dstDir: 目标目录路径
// dstName: 目标文件名
func TransferCloudMove(ctx context.Context, provider CloudMovableProvider, srcRef, dstDir, dstName string) error {
	_, err := provider.Move(ctx, srcRef, dstDir, dstName)
	return err
}
