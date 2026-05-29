import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

export function useRouteQuery<T extends string>(key: string, defaultValue: T) {
  const route = useRoute()
  const router = useRouter()

  const v = route.query[key]
  const initial = (Array.isArray(v) ? (v[0] as string) : (v as string)) ?? defaultValue
  const _value = ref<string>(initial)

  watch(
    () => route.query[key],
    (newVal) => {
      const val = Array.isArray(newVal) ? (newVal[0] as string) : (newVal as string)
      _value.value = val ?? defaultValue
    },
  )

  return computed({
    get: () => _value.value,
    set: (value: string | null) => {
      _value.value = value ?? defaultValue
      const query = { ...route.query }
      if (value === null || value === '' || value === defaultValue) {
        delete query[key]
      } else {
        query[key] = value
      }
      router.replace({ query, hash: route.hash })
    },
  })
}
