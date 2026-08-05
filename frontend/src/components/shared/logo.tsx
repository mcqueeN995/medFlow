import { cn } from '@/lib/utils'

export function Logo({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.4"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={cn('text-primary', className)}
      aria-hidden="true"
    >
      <path d="M12 6.5c-2.2-1.7-5.3-2.2-8.5-1.2v12.4c3.2-1 6.3-.5 8.5 1.2 2.2-1.7 5.3-2.2 8.5-1.2V5.3c-3.2-1-6.3-.5-8.5 1.2Z" />
      <path d="M12 6.5v12.4" />
    </svg>
  )
}
