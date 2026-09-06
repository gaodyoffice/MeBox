package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// jobScanLibraries re-walks every enabled library.
//
// 默认关闭：文件变更由 WatcherService 增量入库，无需周期性全量重扫。
// 仅当用户在设置中显式开启 scan.periodic_enabled 时才执行整库重扫，
// 避免对硬盘的高频反复读取造成损伤（用户明确要求）。
func (s *SchedulerService) jobScanLibraries(ctx context.Context) error {
	manual, _ := ctx.Value(schedulerManualRunKey{}).(bool)
	now := s.currentTime()
	if !manual && !s.periodicScanDue(ctx, now) {
		return nil
	}
	libs, err := s.repo.Library.List(ctx)
	if err != nil {
		return err
	}
	for _, l := range libs {
		if !l.Enabled {
			continue
		}
		if _, err := s.scanner.ScanLibrary(ctx, l.ID); err != nil {
			s.log.Warn("scheduled scan failed",
				zap.String("library", l.ID), zap.Error(err))
		}
	}
	if !manual {
		_ = s.markPeriodicScanCompleted(ctx, now)
	}
	return nil
}

// periodicScanEnabled reports whether the operator opted into periodic full
// library re-scans. Defaults to false so the incremental watcher is the only
// thing touching the disk under normal operation.
func (s *SchedulerService) periodicScanEnabled(ctx context.Context) bool {
	if s.repo == nil || s.repo.Setting == nil {
		return false
	}
	v, err := s.repo.Setting.Get(ctx, "scan.periodic_enabled")
	if err != nil {
		return false
	}
	return parseBoolSetting(v, false)
}

func (s *SchedulerService) periodicScanDue(ctx context.Context, now time.Time) bool {
	if !s.periodicScanEnabled(ctx) {
		return false
	}
	if s.repo == nil || s.repo.Setting == nil {
		return true
	}
	last, err := s.repo.Setting.Get(ctx, localLastPeriodicScanDateKey)
	if err != nil {
		return true
	}
	return strings.TrimSpace(last) != now.In(time.Local).Format("2006-01-02")
}

func (s *SchedulerService) markPeriodicScanCompleted(ctx context.Context, now time.Time) error {
	if s.repo == nil || s.repo.Setting == nil {
		return nil
	}
	return s.repo.Setting.Set(ctx, localLastPeriodicScanDateKey, now.In(time.Local).Format("2006-01-02"))
}

// jobOrganizeSource periodically organizes the configured staging/download
// source directory into the configured media destination. It is intentionally
// opt-in: manual file management remains available, but background disk walking
// only starts after the operator enables organize.auto.
func (s *SchedulerService) jobOrganizeSource(ctx context.Context) error {
	manual, _ := ctx.Value(schedulerManualRunKey{}).(bool)
	if s.organizer == nil || (!manual && !s.autoOrganizeSourceEnabled(ctx)) {
		return nil
	}
	taskName := "自动整理重命名刮削入库"
	if manual {
		taskName = "手动触发自动整理重命名刮削入库"
	}
	resWrap, err := s.ensureOrganizePipeline().Run(ctx, OrganizePipelineRequest{
		Scope:    OrganizeScopeDirectory,
		Trigger:  OrganizeTriggerScheduled,
		TaskName: taskName,
	})
	if err != nil {
		return err
	}
	res := resWrap.Result
	if res == nil {
		res = &OrganizeResult{}
	}
	if s.log != nil && res != nil {
		s.log.Info("scheduled source organize finished",
			zap.String("source", res.SourcePath),
			zap.String("dest", res.DestPath),
			zap.Int("organized", res.Organized),
			zap.Int("replaced", res.Replaced),
			zap.Int("skipped", res.Skipped),
			zap.Int("scrapes", len(res.Scrapes)),
			zap.Int("errors", len(res.Errors)),
		)
	}
	return nil
}

func (s *SchedulerService) ensureOrganizePipeline() *OrganizePipelineService {
	if s.organizePipeline != nil {
		return s.organizePipeline
	}
	return NewOrganizePipelineService(s.log, s.repo, s.organizer, s.scanner, s.tasks)
}

func (s *SchedulerService) startScheduledOrganizeTask(ctx context.Context, manual bool) *TaskHandle {
	if s == nil || s.tasks == nil {
		return nil
	}
	name := "自动整理重命名入库"
	message := "正在执行计划自动整理/重命名/入库"
	if manual {
		name = "手动触发自动整理重命名入库"
		message = "正在执行手动触发的自动整理/重命名/入库"
	}
	return s.tasks.Start(TaskKindOrganize, name, TaskUpdate{
		Stage:      "organize",
		SourcePath: s.organizer.defaultSourceRoot(ctx, ""),
		DestPath:   s.organizer.defaultDestRoot(ctx, ""),
		Message:    message,
	})
}

func (s *SchedulerService) autoOrganizeSourceEnabled(ctx context.Context) bool {
	if s.repo == nil || s.repo.Setting == nil {
		return false
	}
	v, err := s.repo.Setting.Get(ctx, "organize.auto")
	if err != nil {
		return false
	}
	return parseBoolSetting(v, false)
}

func (s *SchedulerService) organizeSourceInterval(ctx context.Context) time.Duration {
	const fallback = 5 * time.Minute
	if s.repo == nil || s.repo.Setting == nil {
		return fallback
	}
	v, err := s.repo.Setting.Get(ctx, "organize.interval_seconds")
	if err != nil {
		return fallback
	}
	seconds, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || seconds <= 0 {
		return fallback
	}
	if seconds < 60 {
		seconds = 60
	}
	return time.Duration(seconds) * time.Second
}

// ── 云盘整理定时任务 ──────────────────────────────────────────────────────────

// cloudOrganizeLoop 支持 cron 和间隔两种调度模式。
// cron 优先级高于间隔：若 cron 非空则按 cron 调度，否则按 interval 调度。
func (s *SchedulerService) cloudOrganizeLoop(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	last := time.Now().Truncate(time.Minute)
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case now := <-ticker.C:
			if s.cloudOrganize == nil {
				last = now
				continue
			}
			if !s.cloudOrganizeAutoEnabled(ctx) {
				last = now
				continue
			}
			now = now.Truncate(time.Minute)
			// 检查 cron 模式
			cronExpr := s.cloudOrganizeCron(ctx)
			if strings.TrimSpace(cronExpr) != "" {
				// cron 模式：逐分钟回放
				due := make([]time.Time, 0, 2)
				for m := last.Add(time.Minute); !m.After(now); m = m.Add(time.Minute) {
					due = append(due, m)
				}
				last = now
				if len(due) == 0 {
					continue
				}
				matched := false
				for _, m := range due {
					if cronMatches(cronExpr, m) {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
				s.runCloudOrganize(ctx)
			} else {
				// 间隔模式
				last = now
				interval := s.cloudOrganizeInterval(ctx)
				if interval <= 0 {
					continue
				}
				lastRunStr, _ := s.repo.Setting.Get(ctx, "cloud.organize.last_run")
				var lastRun time.Time
				if t, err := time.Parse(time.RFC3339, strings.TrimSpace(lastRunStr)); err == nil {
					lastRun = t
				}
				if now.Sub(lastRun) >= interval {
					s.runCloudOrganize(ctx)
					_ = s.repo.Setting.Set(ctx, "cloud.organize.last_run", now.Format(time.RFC3339))
				}
			}
		}
	}
}

func (s *SchedulerService) runCloudOrganize(ctx context.Context) {
	if s.cloudOrganize == nil {
		return
	}
	config, err := s.buildCloudOrganizeConfig(ctx)
	if err != nil {
		s.log.Warn("cloud organize config error", zap.Error(err))
		return
	}
	s.log.Info("cloud organize scheduled start",
		zap.String("source", config.SourcePath),
		zap.String("target", config.TargetPath))
	if _, err := s.cloudOrganize.Organize(ctx, *config); err != nil {
		s.log.Warn("cloud organize scheduled failed", zap.Error(err))
	}
}

func (s *SchedulerService) buildCloudOrganizeConfig(ctx context.Context) (*CloudOrganizeConfig, error) {
	get := func(key string) string {
		v, _ := s.repo.Setting.Get(ctx, key)
		return strings.TrimSpace(v)
	}
	srcAccount := get("organize.cloud_source_account_id")
	srcPath := get("organize.cloud_source_path")
	dstAccount := get("organize.cloud_target_account_id")
	dstPath := get("organize.cloud_target_path")
	if srcAccount == "" || srcPath == "" || dstAccount == "" || dstPath == "" {
		return nil, fmt.Errorf("cloud organize settings incomplete")
	}
	return &CloudOrganizeConfig{
		SourceAccountID: srcAccount,
		SourceProvider:  get("organize.cloud_source_provider"),
		SourcePath:      srcPath,
		TargetAccountID: dstAccount,
		TargetProvider:  get("organize.cloud_target_provider"),
		TargetPath:      dstPath,
		VideoExt:        get("organize.cloud_video_ext"),
		OverwriteMode:   get("organize.cloud_overwrite_mode"),
	}, nil
}

func (s *SchedulerService) cloudOrganizeAutoEnabled(ctx context.Context) bool {
	if s.repo == nil || s.repo.Setting == nil {
		return false
	}
	v, err := s.repo.Setting.Get(ctx, "organize.cloud_auto")
	if err != nil {
		return false
	}
	return parseBoolSetting(v, false)
}

func (s *SchedulerService) cloudOrganizeCron(ctx context.Context) string {
	if s.repo == nil || s.repo.Setting == nil {
		return ""
	}
	v, _ := s.repo.Setting.Get(ctx, "organize.cloud_cron")
	return strings.TrimSpace(v)
}

func (s *SchedulerService) cloudOrganizeInterval(ctx context.Context) time.Duration {
	const fallback = 30 * time.Minute
	if s.repo == nil || s.repo.Setting == nil {
		return fallback
	}
	v, err := s.repo.Setting.Get(ctx, "organize.cloud_interval_seconds")
	if err != nil {
		return fallback
	}
	seconds, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || seconds <= 0 {
		return fallback
	}
	if seconds < 60 {
		seconds = 60
	}
	return time.Duration(seconds) * time.Second
}

// jobCleanTranscodeCache deletes HLS artefacts older than 24h.
func (s *SchedulerService) jobCleanTranscodeCache(ctx context.Context) error {
	if s.cacheDir == "" {
		return nil
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	return walkAndPrune(s.cacheDir+"/hls", cutoff)
}

// jobCleanImageCache prunes image proxy cache files when disk usage exceeds the configured limit.
func (s *SchedulerService) jobCleanImageCache(ctx context.Context) error {
	if s.cacheDir == "" {
		return nil
	}
	maxMB := s.imagesMaxSizeMB()
	if maxMB <= 0 {
		return nil
	}
	imagesDir := filepath.Join(s.cacheDir, "images")
	maxSizeBytes := int64(maxMB) * 1024 * 1024
	res, err := PruneImageCache(imagesDir, maxSizeBytes)
	if err != nil {
		if s.log != nil {
			s.log.Warn("scheduled image cache cleanup failed", zap.Error(err))
		}
		return err
	}
	if res.DeletedFiles > 0 && s.log != nil {
		s.log.Info("scheduled image cache cleanup completed",
			zap.Int("deleted_files", res.DeletedFiles),
			zap.Int64("freed_bytes", res.FreedBytes),
			zap.Int64("remaining_bytes", res.RemainingBytes),
		)
	}
	return nil
}
