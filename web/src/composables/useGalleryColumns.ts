import { ref, type Ref, onMounted, onUnmounted } from 'vue'

const GAP = 8

export function useGalleryColumns(containerRef: Ref<HTMLElement | null>, thumbSizePx: Ref<number>) {
  const columns = ref(4)
  let observer: ResizeObserver | null = null

  function updateColumns() {
    if (containerRef.value) {
      const w = containerRef.value.clientWidth
      columns.value = Math.max(1, Math.floor(w / (thumbSizePx.value + GAP)))
    }
  }

  onMounted(() => {
    if (containerRef.value) {
      updateColumns()
      observer = new ResizeObserver(updateColumns)
      observer.observe(containerRef.value)
    }
  })

  onUnmounted(() => {
    observer?.disconnect()
  })

  return { columns }
}
