<template>
  <div class="relative">
    <video
      ref="videoEl"
      controls
      :poster="poster"
      class="max-h-full max-w-full object-contain"
      preload="auto"
      @error="onError"
    >
      <source :src="currentSrc" type="video/mp4" />
    </video>
    <div v-if="loading" class="absolute inset-0 flex items-center justify-center bg-black/50 text-white text-sm">
      Loading...
    </div>
    <div v-if="error" class="absolute inset-0 flex flex-col items-center justify-center gap-2 bg-black/50 text-white text-sm">
      <span>Failed to load video</span>
      <button @click="retry" class="px-2 py-1 bg-white/10 rounded hover:bg-white/20">Retry</button>
    </div>
    <div v-if="hasProxy" class="absolute top-2 right-2 z-10">
      <button
        @click="toggleQuality"
        class="px-2 py-1 text-xs rounded bg-black/60 text-white hover:bg-black/80"
      >
        {{ quality === 'proxy' ? '720p' : 'Original' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'

const props = defineProps<{
  src: string
  proxySrc?: string
  poster?: string
}>()

const videoEl = ref<HTMLVideoElement | null>(null)
const quality = ref<'proxy' | 'original'>('proxy')
const loading = ref(false)
const error = ref(false)

const hasProxy = computed(() => !!props.proxySrc)

const currentSrc = computed(() => {
  if (quality.value === 'proxy' && props.proxySrc) return props.proxySrc
  return props.src
})

function toggleQuality() {
  const wasTime = videoEl.value?.currentTime || 0
  loading.value = true
  error.value = false
  quality.value = quality.value === 'proxy' ? 'original' : 'proxy'
  setTimeout(() => {
    if (videoEl.value) {
      videoEl.value.currentTime = wasTime
      videoEl.value.play().catch(() => {})
    }
  }, 100)
}

function onError() {
  error.value = true
  loading.value = false
}

function retry() {
  error.value = false
  loading.value = false
  if (videoEl.value) {
    videoEl.value.load()
  }
}

watch(() => props.src, () => {
  quality.value = 'proxy'
  loading.value = false
  error.value = false
  if (videoEl.value) {
    videoEl.value.pause()
    videoEl.value.currentTime = 0
  }
})

watch(currentSrc, () => {
  if (videoEl.value) {
    videoEl.value.addEventListener('canplay', () => { loading.value = false }, { once: true })
  }
})
</script>
