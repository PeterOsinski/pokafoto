import { describe, it, expect, beforeEach } from 'vitest'
import { useLocalSettings, _resetSingleton } from './useLocalSettings'

describe('useLocalSettings', () => {
  beforeEach(() => {
    localStorage.clear()
    _resetSingleton()
  })

  describe('defaults', () => {
    it('returns tiles layout by default', () => {
      const s = useLocalSettings()
      expect(s.layout.value).toBe('tiles')
    })

    it('returns taken_at sort by default', () => {
      const s = useLocalSettings()
      expect(s.sortBy.value).toBe('taken_at')
    })

    it('returns thumbLevel 5 by default', () => {
      const s = useLocalSettings()
      expect(s.thumbLevel.value).toBe(5)
    })
  })

  describe('persistence', () => {
    it('persists layout to localStorage on change', () => {
      const s = useLocalSettings()
      s.layout.value = 'list'
      expect(localStorage.getItem('drive:layout')).toBe('list')
    })

    it('persists sortBy to localStorage on change', () => {
      const s = useLocalSettings()
      s.sortBy.value = 'created_at'
      expect(localStorage.getItem('drive:sort')).toBe('created_at')
    })

    it('persists thumbLevel to localStorage on change', () => {
      const s = useLocalSettings()
      s.thumbLevel.value = 8
      expect(localStorage.getItem('drive:thumbLevel')).toBe('8')
    })
  })

  describe('restore', () => {
    it('reads layout from localStorage', () => {
      localStorage.setItem('drive:layout', 'grouped')
      const s = useLocalSettings()
      expect(s.layout.value).toBe('grouped')
    })

    it('reads sortBy from localStorage', () => {
      localStorage.setItem('drive:sort', 'filename')
      const s = useLocalSettings()
      expect(s.sortBy.value).toBe('filename')
    })

    it('reads thumbLevel from localStorage', () => {
      localStorage.setItem('drive:thumbLevel', '3')
      const s = useLocalSettings()
      expect(s.thumbLevel.value).toBe(3)
    })

    it('falls back to defaults for invalid localStorage values', () => {
      localStorage.setItem('drive:thumbLevel', 'not-a-number')
      localStorage.setItem('drive:layout', '')
      const s = useLocalSettings()
      expect(s.thumbLevel.value).toBe(5)
      expect(s.layout.value).toBe('tiles')
    })
  })

  describe('thumbSizePx', () => {
    it('returns 40 at level 0', () => {
      const s = useLocalSettings()
      s.thumbLevel.value = 0
      expect(s.thumbSizePx.value).toBe(40)
    })

    it('returns 400 at level 9', () => {
      const s = useLocalSettings()
      s.thumbLevel.value = 9
      expect(s.thumbSizePx.value).toBe(400)
    })

    it('returns 240 at default level 5', () => {
      const s = useLocalSettings()
      s.thumbLevel.value = 5
      expect(s.thumbSizePx.value).toBe(240)
    })

    it('clamps below 0 to 40', () => {
      const s = useLocalSettings()
      s.thumbLevel.value = -2
      expect(s.thumbSizePx.value).toBe(40)
    })

    it('clamps above 9 to 400', () => {
      const s = useLocalSettings()
      s.thumbLevel.value = 15
      expect(s.thumbSizePx.value).toBe(400)
    })

    it('increases monotonically with thumbLevel', () => {
      const s = useLocalSettings()
      const values: number[] = []
      for (let i = 0; i <= 9; i++) {
        s.thumbLevel.value = i
        values.push(s.thumbSizePx.value)
      }
      for (let i = 1; i < values.length; i++) {
        expect(values[i]).toBeGreaterThan(values[i - 1])
      }
    })
  })
})
