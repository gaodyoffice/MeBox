import { useState, useEffect } from 'react'
import {
  type AutoOrganizeConfig,
  settingOn,
} from './autoOrganizeModel'
import { strmAPI } from '../api/strm'
import { StrmDirBrowserDialog } from './StrmDialogs'

type ConfigChangeHandler = (key: keyof AutoOrganizeConfig, value: string) => void

type AutoOrganizeTabProps = {
  config: AutoOrganizeConfig
  onConfigChange: ConfigChangeHandler
}

export function AutoOrganizeBasicTab({
  config,
  currentDir,
  onConfigChange,
}: AutoOrganizeTabProps & {
  currentDir: string
}) {
  return (
    <>
      <div className="grid gap-3 lg:grid-cols-[1fr_1fr_150px_140px]">
        <label className="space-y-1">
          <span className="text-xs text-ink-50">整理源目录（待整理 / 下载目录）</span>
          <div className="flex gap-2">
            <input
              className="input-base w-full"
              placeholder="例如 F:\\downloads 或 /downloads"
              value={config.sourceDir}
              onChange={(event) => onConfigChange('sourceDir', event.target.value)}
            />
            <button
              type="button"
              className="rounded-xl border border-gray-200 px-3 text-xs text-ink-100 hover:border-primary-400/40"
              disabled={!currentDir}
              onClick={() => onConfigChange('sourceDir', currentDir)}
            >
              当前
            </button>
          </div>
        </label>
        <label className="space-y-1">
          <span className="text-xs text-ink-50">整理目的地目录（媒体库根目录）</span>
          <div className="flex gap-2">
            <input
              className="input-base w-full"
              placeholder="例如 F:\\media 或 /media"
              value={config.targetDir}
              onChange={(event) => onConfigChange('targetDir', event.target.value)}
            />
            <button
              type="button"
              className="rounded-xl border border-gray-200 px-3 text-xs text-ink-100 hover:border-primary-400/40"
              disabled={!currentDir}
              onClick={() => onConfigChange('targetDir', currentDir)}
            >
              当前
            </button>
          </div>
        </label>
        <label className="space-y-1">
          <span className="text-xs text-ink-50">默认整理方式</span>
          <select
            className="input-base w-full"
            value={config.transferMode}
            onChange={(event) => onConfigChange('transferMode', event.target.value)}
          >
            <option value="hardlink">硬链接</option>
            <option value="move">移动</option>
            <option value="copy">复制</option>
            <option value="symlink">软链接</option>
          </select>
        </label>
        <label className="space-y-1">
          <span className="text-xs text-ink-50">检查间隔（秒）</span>
          <input
            type="number"
            min={60}
            className="input-base w-full"
            value={config.intervalSeconds}
            onChange={(event) => onConfigChange('intervalSeconds', event.target.value)}
          />
        </label>
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <BooleanSetting config={config} settingKey="enabled" label="整理源目录定时自动整理" onConfigChange={onConfigChange} />
        <BooleanSetting config={config} settingKey="smartClassify" label="智能分类到子库" onConfigChange={onConfigChange} />
        <BooleanSetting config={config} settingKey="autoAddLibrary" label="自动注册目的地媒体库" onConfigChange={onConfigChange} />
      </div>
    </>
  )
}

export function AutoOrganizeNamingTab({ config, onConfigChange }: AutoOrganizeTabProps) {
  return (
    <div className="grid gap-3">
      <TextSetting config={config} settingKey="movieFormat" label="电影命名格式" className="input-base w-full font-mono text-xs" onConfigChange={onConfigChange} />
      <TextSetting config={config} settingKey="tvFormat" label="剧集命名格式" className="input-base w-full font-mono text-xs" onConfigChange={onConfigChange} />
      <TextSetting config={config} settingKey="animeFormat" label="动漫命名格式" className="input-base w-full font-mono text-xs" onConfigChange={onConfigChange} />
      <p className="text-xs text-sand-500">
        可用占位符：{'{title}'} {'{year}'} {'{season}'} {'{season:02}'} {'{episode}'} {'{episode:02}'} {'{category}'}。扩展名会自动补齐。
      </p>
    </div>
  )
}

export function AutoOrganizeScrapeTab({ config, onConfigChange }: AutoOrganizeTabProps) {
  return (
    <>
      <div className="flex flex-wrap items-center gap-3">
        <BooleanSetting config={config} settingKey="scrapeAfter" label="整理后自动刮削" onConfigChange={onConfigChange} />
        <BooleanSetting config={config} settingKey="scrapeAutoOnScan" label="扫描后自动刮削" onConfigChange={onConfigChange} />
      </div>
      <div className="grid gap-3 lg:grid-cols-[1fr_160px_160px_160px]">
        <TextSetting
          config={config}
          settingKey="scrapeProviders"
          label="刮削源优先级"
          placeholder="tmdb,douban,bangumi,thetvdb,fanart"
          onConfigChange={onConfigChange}
        />
        <TextSetting config={config} settingKey="scrapeLanguage" label="首选语言" onConfigChange={onConfigChange} />
        <NumberSetting config={config} settingKey="scrapeDelayMinMs" label="最小间隔 ms" onConfigChange={onConfigChange} />
        <NumberSetting config={config} settingKey="scrapeDelayMaxMs" label="最大间隔 ms" onConfigChange={onConfigChange} />
      </div>
    </>
  )
}

function BooleanSetting({
  config,
  settingKey,
  label,
  onConfigChange,
}: AutoOrganizeTabProps & {
  settingKey: keyof AutoOrganizeConfig
  label: string
}) {
  return (
    <label className="flex items-center gap-2 rounded-lg border border-gray-200 bg-gray-50 px-2 py-1 text-xs text-ink-100">
      <input
        type="checkbox"
        checked={settingOn(config[settingKey])}
        onChange={(event) => onConfigChange(settingKey, event.target.checked ? 'true' : 'false')}
      />
      {label}
    </label>
  )
}

function TextSetting({
  config,
  settingKey,
  label,
  className = 'input-base w-full',
  placeholder,
  onConfigChange,
}: AutoOrganizeTabProps & {
  settingKey: keyof AutoOrganizeConfig
  label: string
  className?: string
  placeholder?: string
}) {
  return (
    <label className="space-y-1">
      <span className="text-xs text-ink-50">{label}</span>
      <input
        className={className}
        placeholder={placeholder}
        value={config[settingKey]}
        onChange={(event) => onConfigChange(settingKey, event.target.value)}
      />
    </label>
  )
}

function NumberSetting({
  config,
  settingKey,
  label,
  onConfigChange,
}: AutoOrganizeTabProps & {
  settingKey: keyof AutoOrganizeConfig
  label: string
}) {
  return (
    <label className="space-y-1">
      <span className="text-xs text-ink-50">{label}</span>
      <input
        type="number"
        min={0}
        className="input-base w-full"
        value={config[settingKey]}
        onChange={(event) => onConfigChange(settingKey, event.target.value)}
      />
    </label>
  )
}

export function AutoOrganizeCloudTab({ config, onConfigChange }: AutoOrganizeTabProps) {
  const [accounts, setAccounts] = useState<Array<{ id: string; name: string; provider: string }>>([])
  const [loadingAccounts, setLoadingAccounts] = useState(true)
  const [browseTarget, setBrowseTarget] = useState<'source' | 'target' | null>(null)

  useEffect(() => {
    strmAPI.listAccounts()
      .then((rows) => setAccounts(rows.filter((a) => a.provider !== 'emby_remote')))
      .catch(() => undefined)
      .finally(() => setLoadingAccounts(false))
  }, [])

  const OVERWRITE_MODES = [
    { value: 'size', label: '按大小判断' },
    { value: 'always', label: '始终覆盖' },
    { value: 'never', label: '永不覆盖' },
    { value: 'latest', label: '按修改时间' },
  ]

  const selectedAccountId = browseTarget === 'source' ? config.cloudSourceAccountId : config.cloudTargetAccountId

  return (
    <div className="space-y-4">
      <p className="text-xs text-sand-500">
        云盘整理用于在同一个云盘内移动文件（同盘移动），不支持跨盘或跨存储类型移动。
      </p>

      <div className="grid gap-3 lg:grid-cols-2">
        <div className="space-y-3 rounded-2xl border border-gray-200 p-4">
          <h4 className="text-sm font-semibold text-ink-600">源目录</h4>
          <label className="space-y-1">
            <span className="text-xs text-ink-50">云盘账号</span>
            <select
              className="input-base w-full"
              value={config.cloudSourceAccountId}
              onChange={(event) => {
                const accountId = event.target.value
                const account = accounts.find((a) => a.id === accountId)
                onConfigChange('cloudSourceAccountId', accountId)
                if (account) onConfigChange('cloudSourceProvider', account.provider)
              }}
            >
              <option value="">{loadingAccounts ? '加载中...' : '请选择账号'}</option>
              {accounts.map((a) => (
                <option key={a.id} value={a.id}>{a.name} ({a.provider})</option>
              ))}
            </select>
          </label>
          <label className="space-y-1">
            <span className="text-xs text-ink-50">源目录路径</span>
            <div className="flex gap-2">
              <input
                className="input-base w-full font-mono text-xs"
                placeholder="例如 /下载目录"
                value={config.cloudSourcePath}
                onChange={(event) => onConfigChange('cloudSourcePath', event.target.value)}
              />
              <button
                type="button"
                className="rounded-xl border border-gray-200 px-3 text-xs text-ink-100 hover:border-primary-400/40"
                disabled={!config.cloudSourceAccountId}
                onClick={() => setBrowseTarget('source')}
              >
                浏览
              </button>
            </div>
          </label>
        </div>

        <div className="space-y-3 rounded-2xl border border-gray-200 p-4">
          <h4 className="text-sm font-semibold text-ink-600">目标目录</h4>
          <label className="space-y-1">
            <span className="text-xs text-ink-50">云盘账号</span>
            <select
              className="input-base w-full"
              value={config.cloudTargetAccountId}
              onChange={(event) => {
                const accountId = event.target.value
                const account = accounts.find((a) => a.id === accountId)
                onConfigChange('cloudTargetAccountId', accountId)
                if (account) onConfigChange('cloudTargetProvider', account.provider)
              }}
            >
              <option value="">{loadingAccounts ? '加载中...' : '请选择账号'}</option>
              {accounts.map((a) => (
                <option key={a.id} value={a.id}>{a.name} ({a.provider})</option>
              ))}
            </select>
          </label>
          <label className="space-y-1">
            <span className="text-xs text-ink-50">目标目录路径</span>
            <div className="flex gap-2">
              <input
                className="input-base w-full font-mono text-xs"
                placeholder="例如 /媒体库/电影"
                value={config.cloudTargetPath}
                onChange={(event) => onConfigChange('cloudTargetPath', event.target.value)}
              />
              <button
                type="button"
                className="rounded-xl border border-gray-200 px-3 text-xs text-ink-100 hover:border-primary-400/40"
                disabled={!config.cloudTargetAccountId}
                onClick={() => setBrowseTarget('target')}
              >
                浏览
              </button>
            </div>
          </label>
        </div>
      </div>

      <div className="grid gap-3 lg:grid-cols-2">
        <label className="space-y-1">
          <span className="text-xs text-ink-50">视频扩展名（逗号分隔）</span>
          <input
            className="input-base w-full font-mono text-xs"
            placeholder="mkv,mp4,avi"
            value={config.cloudVideoExt}
            onChange={(event) => onConfigChange('cloudVideoExt', event.target.value)}
          />
        </label>
        <label className="space-y-1">
          <span className="text-xs text-ink-50">覆盖策略</span>
          <select
            className="input-base w-full"
            value={config.cloudOverwriteMode}
            onChange={(event) => onConfigChange('cloudOverwriteMode', event.target.value)}
          >
            {OVERWRITE_MODES.map((mode) => (
              <option key={mode.value} value={mode.value}>{mode.label}</option>
            ))}
          </select>
        </label>
      </div>

      {/* 定时整理配置 */}
      <div className="space-y-3 rounded-2xl border border-gray-200 p-4">
        <h4 className="text-sm font-semibold text-ink-600">定时整理</h4>
        <div className="flex flex-wrap items-center gap-3">
          <BooleanSetting config={config} settingKey="cloudAuto" label="启用定时云盘整理" onConfigChange={onConfigChange} />
        </div>
        <div className="grid gap-3 lg:grid-cols-2">
          <label className="space-y-1">
            <span className="text-xs text-ink-50">Cron 表达式（优先于间隔）</span>
            <input
              className="input-base w-full font-mono text-xs"
              placeholder="例如 0 */6 * * *（每6小时），留空则按间隔"
              value={config.cloudCron}
              onChange={(event) => onConfigChange('cloudCron', event.target.value)}
            />
            <p className="text-xs text-sand-500">5段格式：分 时 日 月 周</p>
          </label>
          <label className="space-y-1">
            <span className="text-xs text-ink-50">间隔秒数（cron 为空时生效）</span>
            <input
              type="number"
              min={60}
              className="input-base w-full"
              value={config.cloudIntervalSeconds}
              onChange={(event) => onConfigChange('cloudIntervalSeconds', event.target.value)}
            />
            <p className="text-xs text-sand-500">最小 60 秒，默认 1800 秒</p>
          </label>
        </div>
      </div>

      {browseTarget && selectedAccountId && (
        <StrmDirBrowserDialog
          accountId={selectedAccountId}
          initialDir={browseTarget === 'source' ? config.cloudSourcePath : config.cloudTargetPath}
          onSelect={(_id, _name, fullPath) => {
            if (browseTarget === 'source') {
              onConfigChange('cloudSourcePath', fullPath || '')
            } else {
              onConfigChange('cloudTargetPath', fullPath || '')
            }
            setBrowseTarget(null)
          }}
          onClose={() => setBrowseTarget(null)}
        />
      )}
    </div>
  )
}
