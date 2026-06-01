<template>
  <div class="relative">
    <div class="flex flex-wrap items-center gap-1.5 min-h-[36px] p-1.5 bg-gray-800 rounded-lg border border-gray-700 focus-within:border-blue-500">
      <span v-for="tag in modelValue" :key="tag" class="inline-flex items-center gap-1 px-2 py-0.5 bg-blue-500/20 text-blue-300 rounded text-sm">
        {{ tag }}
        <button @click="removeTag(tag)" class="text-blue-400 hover:text-blue-200">&times;</button>
      </span>
      <input
        ref="inputRef"
        v-model="inputValue"
        type="text"
        :placeholder="modelValue.length ? '' : placeholder"
        class="flex-1 min-w-[80px] bg-transparent border-none outline-none text-sm px-1 py-0.5"
        @keydown.enter.prevent="addTag"
        @keydown.backspace="handleBackspace"
        @input="onInput"
      />
    </div>
    <div v-if="suggestions.length" class="absolute top-full left-0 right-0 mt-1 bg-gray-800 border border-gray-700 rounded-lg shadow-lg z-50 max-h-48 overflow-y-auto">
      <button
        v-for="s in suggestions"
        :key="s.name"
        class="w-full text-left px-3 py-2 text-sm text-gray-200 hover:bg-gray-700 rounded"
        @click="selectSuggestion(s.name)"
      >
        {{ s.name }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue'
import api from '../api/client'

const props = defineProps<{
  modelValue: string[]
  placeholder?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
}>()

const inputValue = ref('')
const suggestions = ref<{ id: string; name: string }[]>([])
const inputRef = ref<HTMLInputElement>()

async function onInput() {
  if (inputValue.value.trim()) {
    try {
      const res = await api.get('/tags', { params: { q: inputValue.value.trim() } })
      suggestions.value = res.data.tags || []
    } catch {
      suggestions.value = []
    }
  } else {
    suggestions.value = []
  }
}

function addTag() {
  const value = inputValue.value.trim()
  if (!value) return
  if (props.modelValue.includes(value)) {
    inputValue.value = ''
    suggestions.value = []
    return
  }
  emit('update:modelValue', [...props.modelValue, value])
  inputValue.value = ''
  suggestions.value = []
}

function selectSuggestion(name: string) {
  if (props.modelValue.includes(name)) {
    inputValue.value = ''
    suggestions.value = []
    return
  }
  emit('update:modelValue', [...props.modelValue, name])
  inputValue.value = ''
  suggestions.value = []
  nextTick(() => inputRef.value?.focus())
}

function removeTag(tag: string) {
  emit('update:modelValue', props.modelValue.filter(t => t !== tag))
  nextTick(() => inputRef.value?.focus())
}

function handleBackspace() {
  if (inputValue.value === '' && props.modelValue.length > 0) {
    emit('update:modelValue', props.modelValue.slice(0, -1))
  }
}
</script>
