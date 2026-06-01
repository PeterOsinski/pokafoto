import { ref, computed, watch, type Ref, type ComputedRef } from 'vue'

const LS_PREFIX = 'drive:'

function readNumber(key: string, fallback: number): number {
  try {
    const raw = localStorage.getItem(key)
    if (raw === null) return fallback
    const val = parseInt(raw, 10)
    return Number.isFinite(val) ? val : fallback
  } catch {
    return fallback
  }
}

function readString(key: string, fallback: string): string {
  try {
    const raw = localStorage.getItem(key)
    return raw ? raw : fallback
  } catch {
    return fallback
  }
}

function writeString(key: string, value: string) {
  try {
    localStorage.setItem(key, value)
  } catch {
    // quota exceeded or disabled
  }
}

let _layout: Ref<string> | null = null
let _sortBy: Ref<string> | null = null
let _thumbLevel: Ref<number> | null = null
let _highResDownload: Ref<boolean> | null = null
let _previewMode: Ref<string> | null = null
let _sidebarWidth: Ref<number> | null = null
let _thumbSizePx: ComputedRef<number> | null = null

export function useLocalSettings() {
  if (!_layout) {
    _layout = ref(readString(`${LS_PREFIX}layout`, 'tiles'))
    _sortBy = ref(readString(`${LS_PREFIX}sort`, 'taken_at'))
    _thumbLevel = ref(readNumber(`${LS_PREFIX}thumbLevel`, 5))
    _highResDownload = ref(readString(`${LS_PREFIX}highResDownload`, 'false') === 'true')
    _previewMode = ref(readString(`${LS_PREFIX}previewMode`, 'lightbox'))
    _sidebarWidth = ref(readNumber(`${LS_PREFIX}sidebarWidth`, 400))

    _thumbSizePx = computed(() => {
      const t = _thumbLevel!.value
      if (t <= 0) return 40
      if (t >= 9) return 400
      return Math.round(40 + (t / 9) * 360)
    })

    watch(_layout, (v) => writeString(`${LS_PREFIX}layout`, v), { flush: 'sync' })
    watch(_sortBy, (v) => writeString(`${LS_PREFIX}sort`, v), { flush: 'sync' })
    watch(_thumbLevel, (v) => writeString(`${LS_PREFIX}thumbLevel`, String(v)), { flush: 'sync' })
    watch(_highResDownload!, (v) => writeString(`${LS_PREFIX}highResDownload`, v ? 'true' : 'false'), { flush: 'sync' })
    watch(_previewMode!, (v) => writeString(`${LS_PREFIX}previewMode`, v), { flush: 'sync' })
    watch(_sidebarWidth!, (v) => writeString(`${LS_PREFIX}sidebarWidth`, String(v)), { flush: 'sync' })
  }

  return {
    layout: _layout as Ref<string>,
    sortBy: _sortBy as Ref<string>,
    thumbLevel: _thumbLevel as Ref<number>,
    thumbSizePx: _thumbSizePx as ComputedRef<number>,
    highResDownload: _highResDownload as Ref<boolean>,
    previewMode: _previewMode as Ref<string>,
    sidebarWidth: _sidebarWidth as Ref<number>,
  }
}

export function _resetSingleton() {
  _layout = null
  _sortBy = null
  _thumbLevel = null
  _highResDownload = null
  _previewMode = null
  _sidebarWidth = null
  _thumbSizePx = null
}
