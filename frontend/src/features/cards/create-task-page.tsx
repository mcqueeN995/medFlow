import { useRef, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { ArrowLeft, FileText, Loader2, Sparkles, UploadCloud } from 'lucide-react'
import { toast } from 'sonner'
import { postCardsTasks, postUpload } from '@/api/generated/medFlowAPI'
import { CardDifficulty, PostUploadType } from '@/api/generated'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { rememberTaskTopic } from './task-topic-cache'

const DIFFICULTY_LABELS: Record<CardDifficulty, string> = {
  [CardDifficulty.easy]: 'Лёгкая',
  [CardDifficulty.medium]: 'Средняя',
  [CardDifficulty.hard]: 'Сложная',
}

interface LocationState {
  fileId?: string
  fileName?: string
}

export function CreateCardTaskPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const preselected = (location.state as LocationState | null) ?? {}

  const [fileId, setFileId] = useState(preselected.fileId ?? '')
  const [fileName, setFileName] = useState(preselected.fileName ?? '')
  const [uploading, setUploading] = useState(false)
  const [topic, setTopic] = useState('')
  const [difficulty, setDifficulty] = useState<CardDifficulty>(CardDifficulty.medium)
  const [cardsCount, setCardsCount] = useState(10)
  const [submitting, setSubmitting] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  async function handleFile(files: FileList | null) {
    const file = files?.[0]
    if (!file) return
    if (file.type !== 'application/pdf') {
      toast.error('Нужен файл в формате PDF')
      return
    }
    setUploading(true)
    try {
      const res = await postUpload({ file }, { type: PostUploadType.pdf })
      setFileId(res.file_id ?? '')
      setFileName(file.name)
      toast.success('Файл загружен')
    } catch {
      toast.error('Не удалось загрузить файл')
    } finally {
      setUploading(false)
    }
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!fileId) {
      toast.error('Сначала загрузите PDF')
      return
    }
    if (!topic.trim()) {
      toast.error('Укажите тему для карточек')
      return
    }
    setSubmitting(true)
    try {
      const task = await postCardsTasks({
        file_id: fileId,
        topic: topic.trim(),
        difficulty,
        cards_count: cardsCount,
      })
      if (task.id) rememberTaskTopic(task.id, topic.trim())
      toast.success('Задача создана — карточки готовятся')
      navigate(`/cards/tasks/${task.id}`)
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status
      if (status === 429) toast.error('Слишком много активных задач — дождитесь завершения')
      else toast.error('Не удалось создать задачу')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="mx-auto flex max-w-xl flex-col gap-5 p-6">
      <Link to="/cards" className="flex w-fit items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeft className="size-4" /> К карточкам
      </Link>

      <div>
        <h1 className="text-2xl font-bold text-primary">Новая задача на генерацию</h1>
        <p className="text-sm text-muted-foreground">
          ИИ разберёт загруженный PDF и подготовит карточки по указанной теме. Файл удаляется с сервера сразу после обработки.
        </p>
      </div>

      <form onSubmit={onSubmit} className="flex flex-col gap-4 rounded-2xl border border-border bg-card p-5">
        {fileId ? (
          <div className="flex items-center gap-3 rounded-xl border border-border p-3">
            <FileText className="size-5 shrink-0 text-accent" />
            <span className="min-w-0 flex-1 truncate text-sm text-foreground">{fileName}</span>
            <button
              type="button"
              className="text-xs text-muted-foreground underline underline-offset-2 hover:text-foreground"
              onClick={() => {
                setFileId('')
                setFileName('')
              }}
            >
              Заменить
            </button>
          </div>
        ) : (
          <label
            className="flex cursor-pointer flex-col items-center gap-1.5 rounded-xl border-2 border-dashed border-border p-6 text-center transition-colors hover:border-accent"
            onDragOver={(e) => e.preventDefault()}
            onDrop={(e) => {
              e.preventDefault()
              handleFile(e.dataTransfer.files)
            }}
          >
            {uploading ? (
              <Loader2 className="size-6 animate-spin text-accent" />
            ) : (
              <UploadCloud className="size-6 text-accent" />
            )}
            <span className="text-sm font-medium text-foreground">
              {uploading ? 'Загружаем…' : 'Загрузить PDF'}
            </span>
            <input
              ref={inputRef}
              type="file"
              accept="application/pdf"
              className="hidden"
              disabled={uploading}
              onChange={(e) => handleFile(e.target.files)}
            />
          </label>
        )}

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="topic">Тема</Label>
          <Input
            id="topic"
            value={topic}
            onChange={(e) => setTopic(e.target.value)}
            placeholder="Например: Строение сердца"
            className="h-10 rounded-xl"
            maxLength={255}
            required
          />
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div className="flex flex-col gap-1.5">
            <Label>Сложность</Label>
            <Select value={difficulty} onValueChange={(v) => setDifficulty((v as CardDifficulty) ?? CardDifficulty.medium)}>
              <SelectTrigger className="h-10 rounded-xl">
                <SelectValue placeholder="Сложность">{(v: CardDifficulty) => DIFFICULTY_LABELS[v]}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                {Object.values(CardDifficulty).map((d) => (
                  <SelectItem key={d} value={d}>{DIFFICULTY_LABELS[d]}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="count">Кол-во карточек</Label>
            <Input
              id="count"
              type="number"
              min={1}
              max={100}
              value={cardsCount}
              onChange={(e) => setCardsCount(Number(e.target.value))}
              className="h-10 rounded-xl"
            />
          </div>
        </div>

        <Button
          type="submit"
          disabled={submitting || uploading}
          className="h-11 rounded-full bg-linear-to-r from-primary to-accent text-primary-foreground"
        >
          <Sparkles className="size-4" />
          {submitting ? 'Создаём задачу…' : 'Сгенерировать карточки'}
        </Button>
      </form>
    </div>
  )
}
