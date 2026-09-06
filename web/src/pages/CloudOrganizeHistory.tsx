import { useState, useEffect } from 'react'
import { Trash2, RefreshCw, CheckCircle2, XCircle, Clock, Loader2 } from 'lucide-react'
import { api } from '../api/client'
import { apiErrorMessage } from './StrmManagePage'
import toast from 'react-hot-toast'

type CloudOrganizeHistoryItem = {
  id: string
  source_account_id: string
  source_provider: string
  source_path: string
  target_account_id: string
  target_provider: string
  target_path: string
  total_files: number
  success_files: number
  failed_files: number
  status: string
  error_message: string
  started_at: string
  completed_at: string | null
}

export function CloudOrganizeHistory() {
  const [history, setHistory] = useState<CloudOrganizeHistoryItem[]>([])
  const [loading, setLoading] = useState(true)

  const loadHistory = async () => {
    setLoading(true)
    try {
      const res = await api.get<CloudOrganizeHistoryItem[]>('/admin/organize/cloud/history')
      setHistory(res.data)
    } catch (err) {
      toast.error(apiErrorMessage(err))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadHistory()
  }, [])

  const handleClearAll = async () => {
    if (!confirm('确定要清空所有整理历史吗？')) return
    try {
      await api.delete('/admin/organize/cloud/history')
      setHistory([])
      toast.success('历史已清空')
    } catch (err) {
      toast.error(apiErrorMessage(err))
    }
  }

  const handleClearOld = async () => {
    try {
      const res = await api.delete('/admin/organize/cloud/history/old')
      toast.success(`已清理 ${res.data.count} 条 30 天前的历史`)
      loadHistory()
    } catch (err) {
      toast.error(apiErrorMessage(err))
    }
  }

  const formatDate = (dateStr: string) => {
    if (!dateStr) return '-'
    return new Date(dateStr).toLocaleString('zh-CN')
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed': return <CheckCircle2 size={14} className="text-green-500" />
      case 'failed': return <XCircle size={14} className="text-red-500" />
      case 'cancelled': return <Clock size={14} className="text-yellow-500" />
      case 'running': return <Loader2 size={14} className="animate-spin text-blue-500" />
      default: return <Clock size={14} className="text-gray-400" />
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-semibold text-ink-600">整理历史</h4>
        <div className="flex gap-2">
          <button
            type="button"
            className="neon-button !border-gray-300 !text-ink-100"
            disabled={loading}
            onClick={loadHistory}
          >
            <RefreshCw size={12} /> 刷新
          </button>
          <button
            type="button"
            className="neon-button !border-gray-300 !text-ink-100"
            disabled={loading || history.length === 0}
            onClick={handleClearOld}
          >
            清理30天前
          </button>
          <button
            type="button"
            className="neon-button !border-red-300 !text-red-500"
            disabled={loading || history.length === 0}
            onClick={handleClearAll}
          >
            <Trash2 size={12} /> 清空
          </button>
        </div>
      </div>

      {loading ? (
        <div className="flex justify-center py-8">
          <Loader2 className="animate-spin text-ink-50" />
        </div>
      ) : history.length === 0 ? (
        <p className="py-8 text-center text-sm text-sand-500">暂无整理历史</p>
      ) : (
        <div className="space-y-2">
          {history.map((item) => (
            <div
              key={item.id}
              className="flex items-center gap-3 rounded-xl border border-gray-200 p-3"
            >
              {getStatusIcon(item.status)}
              <div className="flex-1 min-w-0">
                <p className="truncate text-xs text-ink-600">
                  {item.source_path} → {item.target_path}
                </p>
                <p className="text-xs text-sand-500">
                  {item.source_provider} · {formatDate(item.started_at)}
                </p>
              </div>
              <div className="flex gap-2 text-xs">
                <span className="text-green-600">{item.success_files} 成功</span>
                {item.failed_files > 0 && (
                  <span className="text-red-500">{item.failed_files} 失败</span>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
