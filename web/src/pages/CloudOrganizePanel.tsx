import { useState, useEffect, useCallback } from 'react'
import { Play, X, Loader2, CheckCircle2, XCircle, AlertTriangle } from 'lucide-react'
import { api } from '../api/client'
import { apiErrorMessage } from './StrmManagePage'
import toast from 'react-hot-toast'

type CloudOrganizeProgress = {
  total: number
  completed: number
  failed: number
  current_file: string
  errors: string[]
}

type CloudOrganizePanelProps = {
  config: {
    cloudSourceAccountId: string
    cloudSourceProvider: string
    cloudSourcePath: string
    cloudTargetAccountId: string
    cloudTargetProvider: string
    cloudTargetPath: string
    cloudVideoExt: string
    cloudOverwriteMode: string
  }
}

export function CloudOrganizePanel({ config }: CloudOrganizePanelProps) {
  const [running, setRunning] = useState(false)
  const [progress, setProgress] = useState<CloudOrganizeProgress | null>(null)
  const [cancelled, setCancelled] = useState(false)

  const pollProgress = useCallback(async () => {
    try {
      const res = await api.get<CloudOrganizeProgress>('/admin/organize/cloud/progress')
      setProgress(res.data)
      if (res.data.total > 0 && res.data.completed + res.data.failed >= res.data.total) {
        setRunning(false)
      }
    } catch {
      // 忽略轮询错误
    }
  }, [])

  useEffect(() => {
    if (!running) return
    const timer = setInterval(pollProgress, 1000)
    return () => clearInterval(timer)
  }, [running, pollProgress])

  const handleStart = async () => {
    if (!config.cloudSourceAccountId || !config.cloudTargetAccountId) {
      toast.error('请选择源和目标云盘账号')
      return
    }
    if (!config.cloudSourcePath || !config.cloudTargetPath) {
      toast.error('请输入源和目标目录路径')
      return
    }

    setRunning(true)
    setCancelled(false)
    setProgress(null)

    try {
      await api.post('/admin/organize/cloud/start', {
        source_account_id: config.cloudSourceAccountId,
        source_provider: config.cloudSourceProvider,
        source_path: config.cloudSourcePath,
        target_account_id: config.cloudTargetAccountId,
        target_provider: config.cloudTargetProvider,
        target_path: config.cloudTargetPath,
        video_ext: config.cloudVideoExt,
        overwrite_mode: config.cloudOverwriteMode,
      })
      toast.success('云盘整理已开始')
    } catch (err) {
      toast.error(apiErrorMessage(err))
      setRunning(false)
    }
  }

  const handleCancel = async () => {
    try {
      await api.post('/admin/organize/cloud/cancel')
      setCancelled(true)
      setRunning(false)
      toast.success('已取消整理')
    } catch (err) {
      toast.error(apiErrorMessage(err))
    }
  }

  const percent = progress && progress.total > 0
    ? Math.round(((progress.completed + progress.failed) / progress.total) * 100)
    : 0

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <button
          type="button"
          className="neon-button"
          disabled={running}
          onClick={handleStart}
        >
          {running ? (
            <>
              <Loader2 size={14} className="animate-spin" /> 整理中...
            </>
          ) : (
            <>
              <Play size={14} /> 开始云盘整理
            </>
          )}
        </button>
        {running && (
          <button
            type="button"
            className="neon-button !border-red-300 !text-red-500 hover:!bg-red-50"
            onClick={handleCancel}
          >
            <X size={14} /> 取消
          </button>
        )}
      </div>

      {progress && (
        <div className="space-y-3 rounded-2xl border border-gray-200 p-4">
          <div className="flex items-center justify-between text-sm">
            <span className="text-ink-600">
              进度: {progress.completed + progress.failed} / {progress.total}
            </span>
            <span className="text-sand-500">{percent}%</span>
          </div>
          <div className="h-2 w-full overflow-hidden rounded-full bg-gray-200">
            <div
              className="h-full rounded-full bg-brand-400 transition-all duration-300"
              style={{ width: `${percent}%` }}
            />
          </div>
          {progress.current_file && (
            <p className="truncate text-xs text-sand-500">
              当前: {progress.current_file}
            </p>
          )}
          <div className="flex gap-4 text-xs">
            <span className="flex items-center gap-1 text-green-600">
              <CheckCircle2 size={12} /> 成功: {progress.completed}
            </span>
            <span className="flex items-center gap-1 text-red-500">
              <XCircle size={12} /> 失败: {progress.failed}
            </span>
          </div>
          {progress.errors.length > 0 && (
            <div className="space-y-1 rounded-lg bg-red-50 p-3">
              <p className="flex items-center gap-1 text-xs font-semibold text-red-600">
                <AlertTriangle size={12} /> 错误日志
              </p>
              <div className="max-h-32 space-y-0.5 overflow-y-auto">
                {progress.errors.map((err, i) => (
                  <p key={i} className="text-xs text-red-500">{err}</p>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {cancelled && (
        <p className="text-xs text-sand-500">整理已取消</p>
      )}
    </div>
  )
}
