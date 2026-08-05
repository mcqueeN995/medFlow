import { useState } from 'react'
import { ArrowLeft, Link as LinkIcon, Send } from 'lucide-react'
import { Link, useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { postThreads } from '@/api/generated/medFlowAPI'
import { ThreadTag } from '@/api/generated'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'
import { THREAD_TAG_LABELS } from '@/lib/forum'

const MAX_TAGS = 5

export function CreateThreadPage() {
  const navigate = useNavigate()
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [tags, setTags] = useState<ThreadTag[]>([])
  const [submitting, setSubmitting] = useState(false)

  function toggleTag(tag: ThreadTag) {
    setTags((prev) => {
      if (prev.includes(tag)) return prev.filter((t) => t !== tag)
      if (prev.length >= MAX_TAGS) {
        toast.error(`Можно выбрать не больше ${MAX_TAGS} тегов`)
        return prev
      }
      return [...prev, tag]
    })
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!title.trim() || !content.trim()) {
      toast.error('Заполните заголовок и текст треда')
      return
    }
    setSubmitting(true)
    try {
      const thread = await postThreads({ title: title.trim(), content: content.trim(), tags })
      toast.success('Тред создан')
      navigate(`/forum/${thread.id}`)
    } catch {
      toast.error('Не удалось создать тред')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="mx-auto flex max-w-xl flex-col gap-5 p-6">
      <Link to="/forum" className="flex w-fit items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeft className="size-4" /> К форуму
      </Link>

      <div>
        <h1 className="text-2xl font-bold text-primary">Новый тред</h1>
        <p className="text-sm text-muted-foreground">Задайте вопрос, поделитесь новостью или предложите обмен</p>
      </div>

      <form onSubmit={onSubmit} className="flex flex-col gap-4 rounded-2xl border border-border bg-card p-5">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="title">Заголовок</Label>
          <Input
            id="title"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="О чём хотите спросить или рассказать?"
            className="h-10 rounded-xl"
            maxLength={500}
            required
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="content">Текст</Label>
          <Textarea
            id="content"
            value={content}
            onChange={(e) => setContent(e.target.value)}
            placeholder="Подробности…"
            className="min-h-32 rounded-xl"
            maxLength={50000}
            required
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label>Теги (до {MAX_TAGS})</Label>
          <div className="flex flex-wrap gap-1.5">
            {Object.values(ThreadTag).map((tag) => (
              <button
                key={tag}
                type="button"
                onClick={() => toggleTag(tag)}
                className={cn(
                  'rounded-full px-3 py-1.5 text-sm font-medium transition-colors',
                  tags.includes(tag)
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-secondary text-secondary-foreground hover:bg-muted',
                )}
              >
                {THREAD_TAG_LABELS[tag]}
              </button>
            ))}
          </div>
        </div>

        <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
          <LinkIcon className="size-3.5" /> Прикрепление файлов к треду скоро появится
        </p>

        <Button
          type="submit"
          disabled={submitting}
          className="h-11 rounded-full bg-linear-to-r from-primary to-accent text-primary-foreground"
        >
          <Send className="size-4" />
          {submitting ? 'Публикуем…' : 'Опубликовать'}
        </Button>
      </form>
    </div>
  )
}
