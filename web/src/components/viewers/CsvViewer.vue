<template>
  <div class="overflow-auto h-full">
    <table v-if="headers.length > 0" class="w-full text-sm text-[var(--text-primary)]">
      <thead class="sticky top-0 z-10">
        <tr>
          <th
            v-for="(header, i) in headers"
            :key="i"
            class="px-3 py-2 text-left text-xs uppercase tracking-wide whitespace-nowrap"
            style="background: var(--bg-surface); border-bottom: 2px solid var(--border-color)"
          >
            {{ header }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="(row, ri) in rows"
          :key="ri"
          class="border-b border-[var(--border-color)]"
          :class="ri % 2 === 0 ? 'bg-transparent' : 'bg-[var(--bg-elevated)]'"
        >
          <td v-for="(cell, ci) in row" :key="ci" class="px-3 py-1.5 whitespace-nowrap max-w-xs truncate">
            {{ cell }}
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="flex items-center justify-center h-full text-[var(--text-secondary)]">
      No data in CSV
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  content: string
}>()

interface ParsedCsv {
  headers: string[]
  rows: string[][]
}

const parsed = computed((): ParsedCsv => {
  try {
    return parseCsv(props.content)
  } catch {
    return { headers: [], rows: [] }
  }
})

const headers = computed(() => parsed.value.headers)
const rows = computed(() => parsed.value.rows)

function parseCsv(text: string): ParsedCsv {
  const trimmed = text.trim()
  if (!trimmed) return { headers: [], rows: [] }

  const allLines: string[] = []
  let currentLine = ''
  let inQuote = false
  for (let i = 0; i < trimmed.length; i++) {
    const ch = trimmed[i]
    if (ch === '"') {
      if (inQuote && i + 1 < trimmed.length && trimmed[i + 1] === '"') {
        currentLine += '""'
        i++
      } else {
        inQuote = !inQuote
      }
      currentLine += ch
    } else if (ch === '\n' && !inQuote) {
      allLines.push(currentLine)
      currentLine = ''
    } else if (ch === '\r' && !inQuote) {
      continue
    } else {
      currentLine += ch
    }
  }
  if (currentLine.trim()) {
    allLines.push(currentLine)
  }

  if (allLines.length === 0) return { headers: [], rows: [] }
  const headers = parseCsvLine(allLines[0])
  const rows = allLines.slice(1).map(parseCsvLine)
  return { headers, rows }
}

function parseCsvLine(line: string): string[] {
  const result: string[] = []
  let current = ''
  let quoted = false
  for (let i = 0; i < line.length; i++) {
    const ch = line[i]
    if (quoted) {
      if (ch === '"') {
        if (i + 1 < line.length && line[i + 1] === '"') {
          current += '"'
          i++
        } else {
          quoted = false
        }
      } else {
        current += ch
      }
    } else {
      if (ch === '"') {
        quoted = true
      } else if (ch === ',') {
        result.push(current.trim())
        current = ''
      } else {
        current += ch
      }
    }
  }
  result.push(current.trim())
  return result
}
</script>
