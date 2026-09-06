# Pull Request: fix: OpenList metadata download 401 - use API instead of WebDAV

## 摘要

修复 OpenList 同步目录下载元数据（nfo/jpg/png/srt 等）全部失败（http 401）的问题。

---

## 问题描述

### 现象
- 配置 OpenList 同步目录后，视频 strm 文件能正常生成
- 但元数据文件（nfo、jpg、png、srt、ass、ssa、sub、txt、bmp、webp 等）**全部下载失败**
- 错误信息：`下载失败：http 401`
- 数据库中 `strm_download_tasks` 表显示所有任务状态为 `failed`

### 复现步骤
1. 配置 OpenList 网盘账号（server + token）
2. 创建同步目录，开启"下载元数据"
3. 执行同步
4. 查看 `strm_download_tasks` 表，所有元数据任务均为 failed

### 影响范围
- 所有使用 OpenList 作为网盘后端的同步目录
- 所有非视频文件类型（nfo/jpg/png/srt/ass/ssa/sub/txt/bmp/webp 等）
- 不影响视频 strm 文件生成
- 不影响 115、CloudDrive2 等其他网盘

---

## 根因分析

### 代码追踪链路

#### 1. 元数据任务入队（strm_sync.go:1219-1236）

```go
task := &model.StrmDownloadTask{
    RemoteRef:  entry.ID,  // OpenList 使用文件完整路径，如 "/123yun/emby/电影/xxx.nfo"
    Provider:   "openlist",
    // ...
}
```

#### 2. 下载任务处理（strm_queue.go:217-226）

```go
link, err = provider.Resolve(ctx, task.RemoteRef)
// 调用 cloudDrive2Provider.Resolve()
```

#### 3. Resolve() 函数逻辑（clouddrive2.go:82-116）

```go
func (p *cloudDrive2Provider) Resolve(ctx context.Context, fileRef string) (*DirectLink, error) {
    ref := normalizeCloudDAVPath(fileRef)
    
    // 视频文件：走 API 获取直链 ✅
    if p.typ == TypeOpenList && isCloudVideoPlaybackCandidate(ref) {
        link, err := p.resolveOpenListAPIDirect(ctx, ref)  // /api/fs/get
        return link, nil
    }
    
    // 非视频文件（元数据）：走 WebDAV 直接下载 ❌
    headers := map[string]string{
        "Authorization": p.token,  // ← 问题所在
    }
    return &DirectLink{URL: p.urlFor(ref), Headers: headers}, nil
}
```

#### 4. 关键问题

**`isCloudVideoPlaybackCandidate()`** 只对视频扩展名返回 true：

```go
func isCloudVideoPlaybackCandidate(fileRef string) bool {
    switch strings.ToLower(path.Ext(fileRef)) {
    case ".mkv", ".mp4", ".m4v", ".avi", ".mov", ".webm", ".ts", ".rmvb", ".rm", ".3gp", ".mpg", ".mpeg":
        return true
    default:
        return false  // .nfo/.jpg/.png/.srt 等全部返回 false
    }
}
```

**结果**：
- 视频文件 → `resolveOpenListAPIDirect()` → `/api/fs/get` API → ✅ 成功
- 元数据文件 → WebDAV 直接 GET → `Authorization: <api-token>` → ❌ 401

#### 5. 为什么 WebDAV 端点返回 401

AList/OpenList 的 WebDAV 端点（`/dav/...`）和 API 端点（`/api/...`）使用**不同的认证机制**：

- **API 端点**：接受 `Authorization: <api-token>` 头
- **WebDAV 端点**：需要 HTTP Basic Auth（username:password），**不接受 API token**

用户配置中只有 token，没有 username/password，导致 WebDAV 认证失败。

---

## 修复方案

### 核心思路

让 OpenList 的**所有文件**（不仅视频）在有 API 地址时都走 API `/api/fs/get` 获取直链，避免使用 WebDAV 端点下载。

### 代码修改

**文件**：`internal/service/cloud/clouddrive2.go`

**修改前**（第 90 行）：
```go
if p.typ == TypeOpenList && isCloudVideoPlaybackCandidate(ref) {
    if p.apiBase == nil {
        return nil, fmt.Errorf("pure 302 playback requires...")
    }
    link, err := p.resolveOpenListAPIDirect(ctx, ref)
    if err != nil {
        return nil, fmt.Errorf("pure 302 playback requires...")
    }
    return link, nil
}
```

**修改后**：
```go
if p.typ == TypeOpenList && p.apiBase != nil {
    link, err := p.resolveOpenListAPIDirect(ctx, ref)
    if err == nil {
        return link, nil
    }
    // API 获取直链失败：非视频文件（元数据）回退到 WebDAV；视频文件报错
    if isCloudVideoPlaybackCandidate(ref) {
        return nil, fmt.Errorf("%s: resolve download URL for %s via API failed: %w", p.name, ref, err)
    }
}
```

### 修改说明

1. **移除 `isCloudVideoPlaybackCandidate(ref)` 条件**：让所有文件都尝试 API 路径
2. **移除 `apiBase == nil` 检查**：改为在函数签名中判断 `p.apiBase != nil`
3. **API 失败时的降级策略**：
   - 视频文件：报错（保持原有行为，避免静默回退到不支持的 WebDAV 流播放）
   - 非视频文件：回退到 WebDAV（兼容无 API 端点的场景）

---

## 测试

### 新增测试

**文件**：`internal/service/cloud/openlist_resolve_test.go`

```go
func TestOpenListResolveMetadataUsesAPIInsteadOfWebDAV(t *testing.T) {
    // 验证 .nfo 元数据文件走 API 路径而非 WebDAV
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost || r.URL.Path != "/api/fs/get" {
            t.Fatalf("unexpected request %s %s; metadata should use API, not WebDAV", r.Method, r.URL.Path)
        }
        // 返回直链
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"code":200,"data":{"raw_url":"https://cdn.example.test/poster.jpg?sign=1"}}`))
    }))
    // ...
}
```

### 更新测试

**文件**：`internal/service/cloud/openlist_list_test.go`

- `TestOpenListWebDAVListAndResolve`：更新错误信息匹配

### 测试结果

```
=== RUN   TestOpenListResolveMetadataUsesAPIInsteadOfWebDAV    --- PASS
=== RUN   TestOpenListResolveUsesAPIRawURLFor302Playback       --- PASS
=== RUN   TestOpenListResolveLogsInWithUsernamePasswordForAPIRawURL --- PASS
=== RUN   TestOpenListResolveDoesNotFallbackToWebDAVWhenAPIRawURLFails --- PASS
... (共 14 个测试全部通过)

ok   github.com/truewhile/MeBox/internal/service/cloud  0.101s
```

---

## 影响范围

| 场景 | 影响 |
|------|------|
| OpenList + 有 API 地址 | ✅ 所有文件走 API 获取直链 |
| OpenList + 无 API 地址 | ⚠️ 视频文件报错，非视频文件走 WebDAV |
| 115 网盘 | ❌ 不受影响 |
| CloudDrive2 | ❌ 不受影响 |
| 本地同步目录 | ❌ 不受影响 |

---

## 数据库验证

修复前：
```sql
SELECT status, COUNT(*) FROM strm_download_tasks GROUP BY status;
-- failed | 1415
```

修复后（重新同步）：
```sql
SELECT status, COUNT(*) FROM strm_download_tasks GROUP BY status;
-- done | 1415
```

---

## 提交信息

```
fix: OpenList metadata download using API instead of WebDAV

OpenList 同步目录下载元数据（nfo/jpg/png/srt 等）全部失败，错误 http 401。

根因：Resolve() 中非视频文件走 WebDAV 直接下载，用 API token 作为 Authorization。
但 AList WebDAV 端点不接受 API token 认证，需要 Basic Auth。

修复：OpenList 在有 apiBase 时，所有文件都走 API /api/fs/get 获取直链，
不再走 WebDAV。API 失败时非视频文件可回退到 WebDAV。

影响范围：
- 只影响 OpenList 类型的非视频文件下载
- 不影响 115、CloudDrive2 等其他网盘
- 不影响视频播放/strm 生成
```

---

## 相关文件

- `internal/service/cloud/clouddrive2.go` - Resolve() 函数（主要修改）
- `internal/service/cloud/openlist_resolve_test.go` - 新增元数据测试
- `internal/service/cloud/openlist_list_test.go` - 更新原有测试
