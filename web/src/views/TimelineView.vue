<template>
  <div>
    <h2 class="text-xl font-bold mb-6 text-[var(--text-primary)]">Timeline</h2>
    <div v-if="groups.length === 0 && !loading" class="text-center py-12 text-[var(--text-secondary)]">
      No photos with dates found.
    </div>
    <div v-else class="space-y-8">
      <div v-for="group in groups" :key="group.period" class="relative pl-8">
        <div class="absolute left-0 top-1 w-3 h-3 rounded-full" style="background: var(--accent)"></div>
        <div class="absolute left-[5px] top-4 bottom-0 w-[2px]" style="background: var(--border-color)"></div>
        <h3 class="text-lg font-semibold text-[var(--text-primary)] mb-2">
          {{ group.label }}
          <span class="text-sm font-normal text-[var(--text-secondary)]">({{ group.count }} photos)</span>
        </h3>
        <router-link
          :to="{ path: '/', query: { date_from: group.startDate, date_to: group.endDate } }"
          class="text-sm text-[var(--accent)] hover:underline"
        >
          View in gallery
        </router-link>
      </div>
    </div>
    <div v-if="loading" class="text-center py-8 text-[var(--text-secondary)]">
      Loading...
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '../api/client'

interface TimelineGroup {
  period: string
  label: string
  count: number
  startDate: string
  endDate: string
}

const groups = ref<TimelineGroup[]>([])
const loading = ref(false)

async function loadTimeline() {
  loading.value = true
  try {
    const res = await api.get('/timeline')
    groups.value = res.data.groups || []
  } catch (e) {
    console.error('Failed to load timeline', e)
  } finally {
    loading.value = false
  }
}

onMounted(loadTimeline)
</script>
