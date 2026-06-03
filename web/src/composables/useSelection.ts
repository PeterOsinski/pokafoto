import { ref } from 'vue'

export function useSelection() {
  const selectedIds = ref(new Set<string>())
  const lastClickedIndex = ref(-1)

  function toggle(id: string) {
    const next = new Set(selectedIds.value)
    if (next.has(id)) {
      next.delete(id)
    } else {
      next.add(id)
    }
    selectedIds.value = next
  }

  function clear() {
    selectedIds.value = new Set()
  }

  function selectAll(ids: string[]) {
    selectedIds.value = new Set(ids)
  }

  function isShiftHeld(): boolean {
    if (typeof window === 'undefined') return false
    return !!(window as any).__shiftHeld
  }

  let shiftDownHandler: ((e: KeyboardEvent) => void) | null = null
  let shiftUpHandler: ((e: KeyboardEvent) => void) | null = null
  let deleteHandler: ((e: KeyboardEvent) => void) | null = null

  function setupKeyboard(onDeleteRequested: () => void) {
    if (typeof window === 'undefined') return
    shiftDownHandler = (e: KeyboardEvent) => { (window as any).__shiftHeld = e.shiftKey }
    shiftUpHandler = (e: KeyboardEvent) => { (window as any).__shiftHeld = e.shiftKey }
    deleteHandler = (e: KeyboardEvent) => {
      if (e.key === 'Delete' && selectedIds.value.size > 0 && document.activeElement?.tagName !== 'INPUT') {
        onDeleteRequested()
      }
    }
    window.addEventListener('keydown', shiftDownHandler)
    window.addEventListener('keyup', shiftUpHandler)
    window.addEventListener('keydown', deleteHandler)
  }

  function teardownKeyboard() {
    if (typeof window === 'undefined') return
    if (shiftDownHandler) window.removeEventListener('keydown', shiftDownHandler)
    if (shiftUpHandler) window.removeEventListener('keyup', shiftUpHandler)
    if (deleteHandler) window.removeEventListener('keydown', deleteHandler)
  }

  return { selectedIds, lastClickedIndex, toggle, clear, selectAll, isShiftHeld, setupKeyboard, teardownKeyboard }
}
