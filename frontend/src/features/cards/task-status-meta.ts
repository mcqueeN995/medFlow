import { CardTaskStatus } from '@/api/generated'

export const TASK_STATUS_META: Record<CardTaskStatus, { label: string; className: string }> = {
  [CardTaskStatus.pending]: { label: 'В очереди', className: 'bg-secondary text-secondary-foreground' },
  [CardTaskStatus.processing]: { label: 'Обрабатывается', className: 'bg-accent text-accent-foreground' },
  [CardTaskStatus.done]: { label: 'Готово', className: 'bg-[color-mix(in_oklch,var(--accent)_20%,transparent)] text-accent' },
  [CardTaskStatus.failed]: { label: 'Ошибка', className: 'bg-destructive/10 text-destructive' },
}
