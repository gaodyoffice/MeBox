import type { Setting } from '../types'

export type AutoOrganizeConfig = {
  enabled: string
  scrapeAfter: string
  smartClassify: string
  autoAddLibrary: string
  sourceDir: string
  targetDir: string
  transferMode: string
  intervalSeconds: string
  movieFormat: string
  tvFormat: string
  animeFormat: string
  scrapeAutoOnScan: string
  scrapeProviders: string
  scrapeLanguage: string
  scrapeDelayMinMs: string
  scrapeDelayMaxMs: string
  // 云盘整理配置
  cloudSourceAccountId: string
  cloudSourceProvider: string
  cloudSourcePath: string
  cloudTargetAccountId: string
  cloudTargetProvider: string
  cloudTargetPath: string
  cloudVideoExt: string
  cloudOverwriteMode: string
  // 云盘整理定时配置
  cloudAuto: string
  cloudCron: string
  cloudIntervalSeconds: string
}

export const AUTO_ORGANIZE_DEFAULTS: AutoOrganizeConfig = {
  enabled: 'false',
  scrapeAfter: 'true',
  smartClassify: 'true',
  autoAddLibrary: 'true',
  sourceDir: '',
  targetDir: '',
  transferMode: 'hardlink',
  intervalSeconds: '300',
  movieFormat: '{title} ({year})/{title} ({year})',
  tvFormat: '{title} ({year})/Season {season:02}/{title} S{season:02}E{episode:02}',
  animeFormat: '{title}/Season {season:02}/{title} S{season:02}E{episode:02}',
  scrapeAutoOnScan: 'false',
  scrapeProviders: 'tmdb,douban,bangumi,thetvdb,fanart',
  scrapeLanguage: 'zh-CN',
  scrapeDelayMinMs: '250',
  scrapeDelayMaxMs: '500',
  // 云盘整理默认配置
  cloudSourceAccountId: '',
  cloudSourceProvider: 'openlist',
  cloudSourcePath: '',
  cloudTargetAccountId: '',
  cloudTargetProvider: 'openlist',
  cloudTargetPath: '',
  cloudVideoExt: 'mkv,mp4,avi,rmvb,rm,mov,ts,wmv,flv,m4v,iso,mpg,mpeg,webm',
  cloudOverwriteMode: 'size',
  // 云盘整理定时默认配置
  cloudAuto: 'false',
  cloudCron: '',
  cloudIntervalSeconds: '1800',
}

export const AUTO_ORGANIZE_KEYS: Record<keyof AutoOrganizeConfig, string> = {
  enabled: 'organize.auto',
  scrapeAfter: 'organize.scrape_after',
  smartClassify: 'organizer.smart_classify',
  autoAddLibrary: 'organize.auto_add_library',
  sourceDir: 'organize.source_dir',
  targetDir: 'organize.target_dir',
  transferMode: 'organize.transfer_mode',
  intervalSeconds: 'organize.interval_seconds',
  movieFormat: 'organize.movie_format',
  tvFormat: 'organize.tv_format',
  animeFormat: 'organize.anime_format',
  scrapeAutoOnScan: 'scrape.auto_on_scan',
  scrapeProviders: 'scrape.providers',
  scrapeLanguage: 'scrape.language',
  scrapeDelayMinMs: 'scrape.delay_min_ms',
  scrapeDelayMaxMs: 'scrape.delay_max_ms',
  // 云盘整理配置键
  cloudSourceAccountId: 'organize.cloud_source_account_id',
  cloudSourceProvider: 'organize.cloud_source_provider',
  cloudSourcePath: 'organize.cloud_source_path',
  cloudTargetAccountId: 'organize.cloud_target_account_id',
  cloudTargetProvider: 'organize.cloud_target_provider',
  cloudTargetPath: 'organize.cloud_target_path',
  cloudVideoExt: 'organize.cloud_video_ext',
  cloudOverwriteMode: 'organize.cloud_overwrite_mode',
  // 云盘整理定时配置键
  cloudAuto: 'organize.cloud_auto',
  cloudCron: 'organize.cloud_cron',
  cloudIntervalSeconds: 'organize.cloud_interval_seconds',
}

export type AutoOrganizeTab = 'basic' | 'naming' | 'scrape' | 'cloud'

export function mergeAutoOrganizeSettings(rows: Setting[]): AutoOrganizeConfig {
  const idx = settingIndex(rows)
  return {
    enabled: idx[AUTO_ORGANIZE_KEYS.enabled] ?? AUTO_ORGANIZE_DEFAULTS.enabled,
    scrapeAfter: idx[AUTO_ORGANIZE_KEYS.scrapeAfter] ?? AUTO_ORGANIZE_DEFAULTS.scrapeAfter,
    smartClassify: idx[AUTO_ORGANIZE_KEYS.smartClassify] ?? AUTO_ORGANIZE_DEFAULTS.smartClassify,
    autoAddLibrary: idx[AUTO_ORGANIZE_KEYS.autoAddLibrary] ?? AUTO_ORGANIZE_DEFAULTS.autoAddLibrary,
    sourceDir: idx[AUTO_ORGANIZE_KEYS.sourceDir] ?? AUTO_ORGANIZE_DEFAULTS.sourceDir,
    targetDir: idx[AUTO_ORGANIZE_KEYS.targetDir] ?? AUTO_ORGANIZE_DEFAULTS.targetDir,
    transferMode: idx[AUTO_ORGANIZE_KEYS.transferMode] ?? AUTO_ORGANIZE_DEFAULTS.transferMode,
    intervalSeconds: idx[AUTO_ORGANIZE_KEYS.intervalSeconds] ?? AUTO_ORGANIZE_DEFAULTS.intervalSeconds,
    movieFormat: idx[AUTO_ORGANIZE_KEYS.movieFormat] ?? AUTO_ORGANIZE_DEFAULTS.movieFormat,
    tvFormat: idx[AUTO_ORGANIZE_KEYS.tvFormat] ?? AUTO_ORGANIZE_DEFAULTS.tvFormat,
    animeFormat: idx[AUTO_ORGANIZE_KEYS.animeFormat] ?? AUTO_ORGANIZE_DEFAULTS.animeFormat,
    scrapeAutoOnScan: idx[AUTO_ORGANIZE_KEYS.scrapeAutoOnScan] ?? AUTO_ORGANIZE_DEFAULTS.scrapeAutoOnScan,
    scrapeProviders: idx[AUTO_ORGANIZE_KEYS.scrapeProviders] ?? AUTO_ORGANIZE_DEFAULTS.scrapeProviders,
    scrapeLanguage: idx[AUTO_ORGANIZE_KEYS.scrapeLanguage] ?? AUTO_ORGANIZE_DEFAULTS.scrapeLanguage,
    scrapeDelayMinMs: idx[AUTO_ORGANIZE_KEYS.scrapeDelayMinMs] ?? AUTO_ORGANIZE_DEFAULTS.scrapeDelayMinMs,
    scrapeDelayMaxMs: idx[AUTO_ORGANIZE_KEYS.scrapeDelayMaxMs] ?? AUTO_ORGANIZE_DEFAULTS.scrapeDelayMaxMs,
    // 云盘整理配置
    cloudSourceAccountId: idx[AUTO_ORGANIZE_KEYS.cloudSourceAccountId] ?? AUTO_ORGANIZE_DEFAULTS.cloudSourceAccountId,
    cloudSourceProvider: idx[AUTO_ORGANIZE_KEYS.cloudSourceProvider] ?? AUTO_ORGANIZE_DEFAULTS.cloudSourceProvider,
    cloudSourcePath: idx[AUTO_ORGANIZE_KEYS.cloudSourcePath] ?? AUTO_ORGANIZE_DEFAULTS.cloudSourcePath,
    cloudTargetAccountId: idx[AUTO_ORGANIZE_KEYS.cloudTargetAccountId] ?? AUTO_ORGANIZE_DEFAULTS.cloudTargetAccountId,
    cloudTargetProvider: idx[AUTO_ORGANIZE_KEYS.cloudTargetProvider] ?? AUTO_ORGANIZE_DEFAULTS.cloudTargetProvider,
    cloudTargetPath: idx[AUTO_ORGANIZE_KEYS.cloudTargetPath] ?? AUTO_ORGANIZE_DEFAULTS.cloudTargetPath,
    cloudVideoExt: idx[AUTO_ORGANIZE_KEYS.cloudVideoExt] ?? AUTO_ORGANIZE_DEFAULTS.cloudVideoExt,
    cloudOverwriteMode: idx[AUTO_ORGANIZE_KEYS.cloudOverwriteMode] ?? AUTO_ORGANIZE_DEFAULTS.cloudOverwriteMode,
    // 云盘整理定时配置
    cloudAuto: idx[AUTO_ORGANIZE_KEYS.cloudAuto] ?? AUTO_ORGANIZE_DEFAULTS.cloudAuto,
    cloudCron: idx[AUTO_ORGANIZE_KEYS.cloudCron] ?? AUTO_ORGANIZE_DEFAULTS.cloudCron,
    cloudIntervalSeconds: idx[AUTO_ORGANIZE_KEYS.cloudIntervalSeconds] ?? AUTO_ORGANIZE_DEFAULTS.cloudIntervalSeconds,
  }
}

export function settingOn(value: string): boolean {
  return ['1', 'true', 'yes', 'on', 'enabled', '启用', '开启'].includes(value.trim().toLowerCase())
}

function settingIndex(rows: Setting[]): Record<string, string> {
  const out: Record<string, string> = {}
  for (const row of rows) out[row.key] = row.value
  return out
}
