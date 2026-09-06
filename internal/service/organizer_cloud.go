package service

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/truewhile/MeBox/internal/model"
	"github.com/truewhile/MeBox/internal/service/cloud"
)

// CloudOrganizeConfig 云盘整理配置。
type CloudOrganizeConfig struct {
	SourceAccountID string `json:"source_account_id"`
	SourceProvider  string `json:"source_provider"` // openlist
	SourcePath      string `json:"source_path"`     // /下载目录

	TargetAccountID string `json:"target_account_id"`
	TargetProvider  string `json:"target_provider"` // openlist
	TargetPath      string `json:"target_path"`     // /媒体库

	VideoExt      string `json:"video_ext"`       // 视频扩展名
	OverwriteMode string `json:"overwrite_mode"`  // always/size/never/latest
}

// CloudOrganizeProgress 云盘整理进度。
type CloudOrganizeProgress struct {
	TotalFiles   int    `json:"total_files"`
	SuccessFiles int    `json:"success_files"`
	FailedFiles  int    `json:"failed_files"`
	CurrentFile  string `json:"current_file"`
	IsRunning    bool   `json:"is_running"`
 ErrorMessage string `json:"error_message,omitempty"`
}

// CloudOrganizeResult 云盘整理结果。
type CloudOrganizeResult struct {
	TotalFiles   int       `json:"total_files"`
	SuccessFiles int       `json:"success_files"`
	FailedFiles  int       `json:"failed_files"`
	Errors       []string  `json:"errors,omitempty"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at"`
}

// CloudOrganizeRepository 云盘整理仓储接口。
type CloudOrganizeRepository interface {
	Create(ctx context.Context, h *model.CloudOrganizeHistory) error
	List(ctx context.Context, limit int) ([]model.CloudOrganizeHistory, error)
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)
	DeleteAll(ctx context.Context) (int64, error)
}

// cloudOrganizeService 云盘整理服务。
type cloudOrganizeService struct {
	repo        CloudOrganizeRepository
	accountProv *cloud.AccountConfigProvider
	progress    CloudOrganizeProgress
	cancel      context.CancelFunc
	mu          sync.Mutex
}

// newCloudOrganizeService 创建云盘整理服务。
func newCloudOrganizeService(repo CloudOrganizeRepository, accountProv *cloud.AccountConfigProvider) *cloudOrganizeService {
	return &cloudOrganizeService{
		repo:        repo,
		accountProv: accountProv,
	}
}

// GetProgress 获取当前整理进度。
func (s *cloudOrganizeService) GetProgress() CloudOrganizeProgress {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.progress
}

// Cancel 取消整理。
func (s *cloudOrganizeService) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

// Organize 执行云盘整理。
func (s *cloudOrganizeService) Organize(ctx context.Context, config CloudOrganizeConfig) (*CloudOrganizeResult, error) {
	s.mu.Lock()
	if s.progress.IsRunning {
		s.mu.Unlock()
		return nil, fmt.Errorf("cloud organize already running")
	}
	s.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancel = cancel
	s.progress = CloudOrganizeProgress{IsRunning: true}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.cancel = nil
		s.progress.IsRunning = false
		s.mu.Unlock()
	}()

	startTime := time.Now()

	// 获取源 Provider
	srcProvider, err := s.accountProv.GetProvider(ctx, config.SourceAccountID)
	if err != nil {
		return nil, fmt.Errorf("get source provider: %w", err)
	}

	// 获取目标 Provider
	dstProvider, err := s.accountProv.GetProvider(ctx, config.TargetAccountID)
	if err != nil {
		return nil, fmt.Errorf("get target provider: %w", err)
	}

	// 检查是否为 MovableProvider
	srcMovable, ok := srcProvider.(cloud.MovableProvider)
	if !ok {
		return nil, fmt.Errorf("source provider does not support move")
	}
	_ = dstProvider // 目标 provider 用于检查目录存在

	// 列出源目录文件
	srcFiles, err := srcProvider.List(ctx, config.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("list source directory: %w", err)
	}

	// 过滤视频文件
	videoExtMap := parseVideoExtMap(config.VideoExt)
	var videoFiles []cloud.FileEntry
	for _, f := range srcFiles {
		if f.IsDir {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f.Name))
		if videoExtMap[ext] {
			videoFiles = append(videoFiles, f)
		}
	}

	s.mu.Lock()
	s.progress.TotalFiles = len(videoFiles)
	s.mu.Unlock()

	result := &CloudOrganizeResult{
		TotalFiles: len(videoFiles),
		StartedAt:  startTime,
	}

	// 构建目标前缀
	srcPrefix := config.SourcePath
	dstPrefix := config.TargetPath

	// 逐个移动文件
	for _, file := range videoFiles {
		select {
		case <-ctx.Done():
			result.Errors = append(result.Errors, "cancelled by user")
			result.CompletedAt = time.Now()
			s.mu.Lock()
			s.progress.ErrorMessage = "cancelled by user"
			s.mu.Unlock()
			return result, nil
		default:
		}

		s.mu.Lock()
		s.progress.CurrentFile = file.Name
		s.mu.Unlock()

		// 构建目标路径
		relPath := strings.TrimPrefix(file.ID, srcPrefix)
		if relPath == file.ID {
			// 如果 ID 不包含路径前缀，使用文件名
			relPath = "/" + file.Name
		}
		dstPath := dstPrefix + relPath
		dstDir := filepath.Dir(dstPath)
		dstName := filepath.Base(dstPath)

		// 检查目标是否已存在
		dstEntries, err := dstProvider.List(ctx, dstDir)
		if err != nil {
			// 目录可能不存在，继续
			dstEntries = nil
		}

		var dstEntry *cloud.FileEntry
		for _, e := range dstEntries {
			if e.Name == dstName {
				dstEntry = &e
				break
			}
		}

		// 检查覆盖策略
		overwriteMode := CloudOverwriteMode(config.OverwriteMode)
		if !CheckCloudOverwrite(&file, dstEntry, overwriteMode) {
			s.mu.Lock()
			s.progress.SuccessFiles++ // 跳过也算成功
			s.mu.Unlock()
			continue
		}

		// 执行移动
		_, err = srcMovable.Move(ctx, file.ID, dstDir, dstName)
		if err != nil {
			errMsg := fmt.Sprintf("failed to move %s: %v", file.Name, err)
			result.Errors = append(result.Errors, errMsg)
			log.Printf("[CloudOrganize] %s", errMsg)
			s.mu.Lock()
			s.progress.FailedFiles++
			s.mu.Unlock()
			continue
		}

		s.mu.Lock()
		s.progress.SuccessFiles++
		s.mu.Unlock()
	}

	result.SuccessFiles = s.progress.SuccessFiles
	result.FailedFiles = s.progress.FailedFiles
	result.CompletedAt = time.Now()

	// 保存整理历史
	history := &model.CloudOrganizeHistory{
		SourceAccountID: config.SourceAccountID,
		SourceProvider:  config.SourceProvider,
		SourcePath:      config.SourcePath,
		TargetAccountID: config.TargetAccountID,
		TargetProvider:  config.TargetProvider,
		TargetPath:      config.TargetPath,
		TotalFiles:      result.TotalFiles,
		SuccessFiles:    result.SuccessFiles,
		FailedFiles:     result.FailedFiles,
		Status:          "completed",
		StartedAt:       result.StartedAt,
		CompletedAt:     &result.CompletedAt,
	}
	if len(result.Errors) > 0 {
		history.ErrorMessage = strings.Join(result.Errors, "; ")
	}
	_ = s.repo.Create(ctx, history)

	return result, nil
}

// parseVideoExtMap 解析视频扩展名配置为 map。
func parseVideoExtMap(exts string) map[string]bool {
	result := make(map[string]bool)
	if exts == "" {
		// 默认扩展名
		exts = "mkv,mp4,avi,rmvb,rm,mov,ts,wmv,flv,m4v,iso,mpg,mpeg,webm"
	}
	for _, ext := range strings.Split(exts, ",") {
		ext = strings.TrimSpace(ext)
		if ext != "" {
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			result[strings.ToLower(ext)] = true
		}
	}
	return result
}
