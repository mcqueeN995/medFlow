import { useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowLeft, FileText, Layers, Loader2, UploadCloud } from 'lucide-react'
import { toast } from 'sonner'
import { postUpload } from '@/api/generated/medFlowAPI'
import { PostUploadType } from '@/api/generated'
import type { UploadResponse } from '@/api/generated'
import { buttonVariants } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface UploadedFile extends UploadResponse {
  name: string
}

export function LibraryUploadPage() {
  const [uploads, setUploads] = useState<UploadedFile[]>([])
  const [uploading, setUploading] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  async function handleFiles(files: FileList | null) {
    const file = files?.[0]
    if (!file) return
    if (file.type !== 'application/pdf') {
      toast.error('Нужен файл в формате PDF')
      return
    }
    setUploading(true)
    try {
      const res = await postUpload({ file }, { type: PostUploadType.pdf })
      setUploads((prev) => [{ ...res, name: file.name }, ...prev])
      toast.success('Файл загружен')
    } catch {
      toast.error('Не удалось загрузить файл')
    } finally {
      setUploading(false)
      if (inputRef.current) inputRef.current.value = ''
    }
  }

  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-5 p-6">
      <Link to="/library" className="flex w-fit items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeft className="size-4" /> К каталогу
      </Link>

      <div>
        <h1 className="text-2xl font-bold text-primary">Мои материалы для ИИ</h1>
        <p className="text-sm text-muted-foreground">
          Загрузите свою легальную копию главы или конспекта — после генерации карточек файл удаляется с сервера.
        </p>
      </div>

      <label
        className="flex cursor-pointer flex-col items-center gap-2 rounded-2xl border-2 border-dashed border-border bg-card p-10 text-center transition-colors hover:border-accent"
        onDragOver={(e) => e.preventDefault()}
        onDrop={(e) => {
          e.preventDefault()
          handleFiles(e.dataTransfer.files)
        }}
      >
        {uploading ? (
          <Loader2 className="size-8 animate-spin text-accent" />
        ) : (
          <UploadCloud className="size-8 text-accent" />
        )}
        <span className="font-medium text-foreground">
          {uploading ? 'Загружаем…' : 'Перетащите PDF сюда или нажмите, чтобы выбрать'}
        </span>
        <span className="text-xs text-muted-foreground">Только PDF, временное хранение 24 часа</span>
        <input
          ref={inputRef}
          type="file"
          accept="application/pdf"
          className="hidden"
          disabled={uploading}
          onChange={(e) => handleFiles(e.target.files)}
        />
      </label>

      {uploads.length > 0 && (
        <div className="flex flex-col gap-2">
          <h2 className="text-sm font-semibold text-foreground">Загружено в этой сессии</h2>
          {uploads.map((u) => (
            <div key={u.file_id} className="flex items-center gap-3 rounded-xl border border-border bg-card p-3">
              <FileText className="size-5 shrink-0 text-accent" />
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium text-foreground">{u.name}</p>
                <p className="text-xs text-muted-foreground">
                  {u.size_bytes ? `${Math.round(u.size_bytes / 1024)} КБ · ` : ''}
                  доступен до {u.expires_at ? new Date(u.expires_at).toLocaleString('ru-RU') : '—'}
                </p>
              </div>
              <Link
                to="/cards/create"
                state={{ fileId: u.file_id, fileName: u.name }}
                className={cn(buttonVariants({ variant: 'outline', size: 'sm' }), 'shrink-0 rounded-full')}
              >
                <Layers className="size-3.5" /> Создать карточки
              </Link>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
