<template>
  <div class="h-full overflow-y-auto p-3">
    <span class="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wider">Photos by date</span>

    <div class="mt-2">
      <button
        class="flex items-center w-full text-left text-sm py-1 px-2 rounded hover:bg-[var(--bg-elevated)]"
        :class="!dateFrom && !dateTo ? 'text-[var(--accent)]' : 'text-[var(--text-primary)]'"
        @click="clearDate"
      >
        All photos
      </button>
    </div>

    <div v-if="loading" class="text-xs text-[var(--text-secondary)] py-2">
      Loading...
    </div>

    <div v-else-if="error" class="text-xs text-[var(--error)] py-2">
      {{ error }}
    </div>

    <div v-else-if="yearGroups.length === 0" class="text-xs text-[var(--text-secondary)] py-2">
      No photos with dates
    </div>

    <template v-else>
      <div v-for="year in yearGroups" :key="year.year">
        <button
          @click="toggleYear(year.year)"
          class="flex items-center w-full text-left text-sm py-1 px-2 rounded hover:bg-[var(--bg-elevated)]"
          :class="isYearActive(year.year) ? 'text-[var(--accent)]' : 'text-[var(--text-primary)]'"
        >
          <span class="mr-1 text-xs w-4 inline-block text-center">
            {{ expandedYears.has(year.year) ? '&#9660;' : '&#9654;' }}
          </span>
          <span class="flex-1">{{ year.year }}</span>
          <span class="text-xs text-[var(--text-secondary)] ml-1">{{ year.total }}</span>
        </button>
        <template v-if="expandedYears.has(year.year)">
          <div class="ml-3">
            <button
              v-for="month in year.months"
              :key="`${year.year}-${month.month}`"
              @click="selectMonth(year.year, month.month)"
              class="flex items-center w-full text-left text-sm py-1 px-2 rounded hover:bg-[var(--bg-elevated)]"
              :class="isMonthActive(year.year, month.month) ? 'text-[var(--accent)]' : 'text-[var(--text-primary)]'"
            >
              <span class="flex-1">{{ month.label }}</span>
              <span class="text-xs text-[var(--text-secondary)] ml-1">{{ month.count }}</span>
            </button>
          </div>
        </template>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import api from '../api/client'

interface MonthGroup {
  month: number
  label: string
  count: number
}

interface YearGroup {
  year: number
  total: number
  months: MonthGroup[]
}

interface TimelineGroup {
  period: string
  label: string
  count: number
}

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const error = ref('')
const groups = ref<TimelineGroup[]>([])
const expandedYears = ref(new Set<number>())

const dateFrom = computed(() => (route.query.date_from as string) || '')
const dateTo = computed(() => (route.query.date_to as string) || '')

const yearGroups = computed<YearGroup[]>(() => {
  const map = new Map<number, YearGroup>()

  for (const g of groups.value) {
    const parts = g.period.split('-')
    if (parts.length < 2) continue
    const year = parseInt(parts[0], 10)
    const month = parseInt(parts[1], 10)

    if (!map.has(year)) {
      map.set(year, { year, total: 0, months: [] })
    }

    const yg = map.get(year)!
    yg.total += g.count
    yg.months.push({ month, label: g.label, count: g.count })
  }

  return Array.from(map.values()).sort((a, b) => b.year - a.year)
})

function isYearActive(year: number): boolean {
  if (!dateFrom.value && !dateTo.value) return false
  const fromYear = dateFrom.value ? new Date(dateFrom.value).getFullYear() : null
  const toYear = dateTo.value ? new Date(dateTo.value).getFullYear() : null
  return fromYear === year && toYear === year
}

function isMonthActive(year: number, month: number): boolean {
  const targetStart = `${year}-${String(month).padStart(2, '0')}-01`
  return dateFrom.value === targetStart
}

function toggleYear(year: number) {
  if (expandedYears.value.has(year)) {
    expandedYears.value.delete(year)
  } else {
    expandedYears.value.add(year)
  }
  expandedYears.value = new Set(expandedYears.value)
}

function selectMonth(year: number, month: number) {
  const from = `${year}-${String(month).padStart(2, '0')}-01`
  const nextMonth = month === 12 ? 1 : month + 1
  const nextYear = month === 12 ? year + 1 : year
  const to = `${nextYear}-${String(nextMonth).padStart(2, '0')}-01`

  router.replace({
    query: {
      ...route.query,
      date_from: from,
      date_to: to,
    },
  })
}

function clearDate() {
  const { date_from, date_to, ...rest } = route.query
  router.replace({ query: rest })
}

async function fetchTimeline() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get('/timeline', { params: { granularity: 'month' } })
    groups.value = res.data.groups || []
  } catch (e) {
    error.value = 'Failed to load dates'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchTimeline()
})

watch(
  () => [dateFrom.value, dateTo.value],
  ([newFrom, newTo], [oldFrom, oldTo]) => {
    if (newFrom !== oldFrom || newTo !== oldTo) {
      if (newFrom) {
        const d = new Date(newFrom)
        const year = d.getFullYear()
        expandedYears.value.add(year)
        expandedYears.value = new Set(expandedYears.value)
      }
    }
  },
)
</script>
