# Changelog

## v0.1.0 (2026-09-06)

### 修复
- OpenList 同步目录元数据下载 401 错误（nfo/jpg/png/srt 等）
  - 根因：非视频文件走 WebDAV 下载，API token 不被 WebDAV 端点接受
  - 修复：所有文件优先走 API /api/fs/get 获取直链
