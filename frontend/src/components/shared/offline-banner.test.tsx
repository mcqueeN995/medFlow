import { act, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { OfflineBanner } from './offline-banner'

describe('OfflineBanner', () => {
  it('renders nothing while online', () => {
    render(<OfflineBanner />)
    expect(screen.queryByText(/Нет соединения/i)).not.toBeInTheDocument()
  })

  it('shows the banner when the browser goes offline, hides it when back online', () => {
    render(<OfflineBanner />)

    act(() => {
      window.dispatchEvent(new Event('offline'))
    })
    expect(screen.getByText(/Нет соединения/i)).toBeInTheDocument()

    act(() => {
      window.dispatchEvent(new Event('online'))
    })
    expect(screen.queryByText(/Нет соединения/i)).not.toBeInTheDocument()
  })
})
