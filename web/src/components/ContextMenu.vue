<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="fixed inset-0 z-[100]"
      @click.self="close"
      @contextmenu.prevent="close"
    >
      <div
        ref="menuRef"
        class="absolute rounded-lg shadow-xl py-1 min-w-[140px]"
        style="background: var(--bg-surface); border: 1px solid var(--border-color)"
        :style="{ left: position.x + 'px', top: position.y + 'px' }"
      >
        <button
          v-for="(item, i) in items"
          :key="i"
          @click="execute(item)"
          class="flex items-center gap-2 w-full px-3 py-2 text-sm text-left transition-colors"
          :class="item.danger ? 'text-red-400 hover:bg-red-500/10' : 'text-[var(--text-primary)] hover:bg-[var(--bg-elevated)]'"
        >
          <span v-if="item.icon" class="w-4 text-center" v-html="item.icon"></span>
          <span>{{ item.label }}</span>
        </button>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'

export interface ContextMenuItem {
  label: string
  icon?: string
  danger?: boolean
  action: () => void
}

const props = defineProps<{
  visible: boolean
  position: { x: number; y: number }
  items: ContextMenuItem[]
}>()

const emit = defineEmits<{
  close: []
}>()

const menuRef = ref<HTMLElement | null>(null)

function close() {
  emit('close')
}

function execute(item: ContextMenuItem) {
  item.action()
  close()
}

watch(() => props.visible, async (v) => {
  if (v) {
    await nextTick()
    if (menuRef.value) {
      const rect = menuRef.value.getBoundingClientRect()
      const viewportW = window.innerWidth
      const viewportH = window.innerHeight

      if (rect.right > viewportW) {
        menuRef.value.style.left = (props.position.x - rect.width + 4) + 'px'
      }
      if (rect.bottom > viewportH) {
        menuRef.value.style.top = (props.position.y - rect.height + 4) + 'px'
      }
    }
  }
})
</script>
