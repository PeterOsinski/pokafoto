import { vi } from 'vitest'

class MockIntersectionObserver {
  root: Element | null = null
  rootMargin = ''
  thresholds: number[] = []

  constructor(callback: IntersectionObserverCallback) {
    const entries: IntersectionObserverEntry[] = [{
      isIntersecting: true,
      boundingClientRect: {} as DOMRectReadOnly,
      intersectionRatio: 1,
      intersectionRect: {} as DOMRectReadOnly,
      rootBounds: null,
      target: {} as Element,
      time: Date.now(),
    }]
    setTimeout(() => callback(entries, this), 0)
  }
  observe = vi.fn()
  unobserve = vi.fn()
  disconnect = vi.fn()
  takeRecords = vi.fn(() => [])
}
;(globalThis as any).IntersectionObserver = MockIntersectionObserver

class MockResizeObserver {
  observe = vi.fn()
  unobserve = vi.fn()
  disconnect = vi.fn()
}
;(globalThis as any).ResizeObserver = MockResizeObserver
