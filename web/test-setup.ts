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
  private callback: ResizeObserverCallback | null = null
  observe = vi.fn()
  unobserve = vi.fn()
  disconnect = vi.fn()
  constructor(callback: ResizeObserverCallback) {
    this.callback = callback
    const entries: ResizeObserverEntry[] = [{
      target: {} as Element,
      contentRect: { width: 1024, height: 768, x: 0, y: 0, top: 0, right: 1024, bottom: 768, left: 0 } as DOMRectReadOnly,
      borderBoxSize: [],
      contentBoxSize: [],
      devicePixelContentBoxSize: [],
    }]
    setTimeout(() => {
      if (this.callback) this.callback(entries, this)
    }, 0)
  }
}
;(globalThis as any).ResizeObserver = MockResizeObserver
