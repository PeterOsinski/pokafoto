<template>
  <div class="h-full flex flex-col">
    <div ref="mapContainer" class="flex-1 rounded-lg overflow-hidden z-0 relative" style="min-height: 400px">
      <div v-if="isEmpty && !loading" class="absolute inset-0 flex items-center justify-center z-[999]" style="pointer-events: none">
        <p class="text-[var(--text-secondary)] bg-[var(--bg-surface)]/90 px-4 py-2 rounded-lg">No geo-tagged photos. Photos with GPS data will appear here.</p>
      </div>
    </div>
    <div v-if="selectedPoint" class="p-3 mt-2 rounded-md" style="background: var(--bg-surface)">
      <p class="text-sm font-medium text-[var(--text-primary)]">{{ selectedPoint.fileId }}</p>
      <p class="text-xs text-[var(--text-secondary)]">{{ selectedPoint.takenAt }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import api from '../api/client'

interface GeoPoint {
  fileId: string
  latitude: number
  longitude: number
  thumbnailUrl: string
  takenAt: string
}

interface Cluster {
  latitude: number
  longitude: number
  count: number
  thumbnailUrl?: string
}

const mapContainer = ref<HTMLElement | null>(null)
const selectedPoint = ref<GeoPoint | null>(null)
const isEmpty = ref(false)
const loading = ref(true)
let map: L.Map | null = null
let markerLayer: L.LayerGroup | null = null

onMounted(() => {
  if (!mapContainer.value) return

  map = L.map(mapContainer.value).setView([51.505, -0.09], 3)

  L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
    attribution: '&copy; OpenStreetMap contributors',
    maxZoom: 19,
  }).addTo(map)

  markerLayer = L.layerGroup().addTo(map)

  map.on('moveend', loadPoints)
  loadPoints()
})

onUnmounted(() => {
  map?.remove()
  map = null
})

async function loadPoints() {
  if (!map) return

  const bounds = map.getBounds()
  const zoom = map.getZoom()

  try {
    const res = await api.get('/geo/clusters', {
      params: {
        zoom,
        lat_min: bounds.getSouth(),
        lat_max: bounds.getNorth(),
        lon_min: bounds.getWest(),
        lon_max: bounds.getEast(),
      },
    })

    markerLayer?.clearLayers()

    const clusters = res.data.clusters as Cluster[]
    if (clusters.length === 0 && loading.value) {
      isEmpty.value = true
      loading.value = false
    } else if (clusters.length > 0) {
      isEmpty.value = false
      loading.value = false
    }

    for (const c of clusters) {
      const marker = L.circleMarker([c.latitude, c.longitude], {
        radius: Math.min(8 + c.count * 2, 40),
        fillColor: '#3b82f6',
        color: '#1d4ed8',
        weight: 1,
        opacity: 1,
        fillOpacity: 0.7,
      }).addTo(markerLayer!)

      marker.bindPopup(`<b>${c.count} photos</b>`)
    }
  } catch (e) {
    console.error('Failed to load geo clusters', e)
    loading.value = false
  }
}
</script>
