<template>
  <div class="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
    <div class="flex items-center justify-between">
      <h2 class="font-semibold text-gray-900">Evolución del saldo</h2>
      <span class="text-xs text-gray-400">{{ account?.account_number }}</span>
    </div>

    <div v-if="loading" class="py-10 text-center text-sm text-gray-400">Cargando gráfica...</div>

    <div v-else-if="points.length < 2" class="py-10 text-center text-sm text-gray-400">
      Todavía no hay suficientes movimientos para graficar la evolución del saldo.
    </div>

    <div v-else class="relative mt-4" @mouseleave="hoverIndex = null">
      <svg :viewBox="`0 0 ${width} ${height}`" class="w-full" :height="height" @mousemove="handleMove">
        <line
          v-for="(gy, i) in gridLines"
          :key="i"
          :x1="padding"
          :x2="width - padding"
          :y1="gy"
          :y2="gy"
          stroke="#e5e7eb"
          stroke-width="1"
        />

        <path :d="areaPath" fill="url(#balanceGradient)" />
        <path :d="linePath" fill="none" stroke="#2563eb" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />

        <circle
          v-for="(p, i) in scaledPoints"
          :key="i"
          :cx="p.x"
          :cy="p.y"
          r="3"
          fill="#2563eb"
          :opacity="i === scaledPoints.length - 1 ? 1 : 0"
        />

        <g v-if="hoverIndex !== null">
          <line
            :x1="scaledPoints[hoverIndex].x"
            :x2="scaledPoints[hoverIndex].x"
            :y1="padding"
            :y2="height - padding"
            stroke="#9ca3af"
            stroke-width="1"
            stroke-dasharray="3 3"
          />
          <circle :cx="scaledPoints[hoverIndex].x" :cy="scaledPoints[hoverIndex].y" r="4" fill="#1d4ed8" />
        </g>

        <defs>
          <linearGradient id="balanceGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#2563eb" stop-opacity="0.18" />
            <stop offset="100%" stop-color="#2563eb" stop-opacity="0" />
          </linearGradient>
        </defs>
      </svg>

      <div
        v-if="hoverIndex !== null"
        class="pointer-events-none absolute -translate-x-1/2 -translate-y-full rounded-md bg-gray-900 px-2 py-1 text-xs text-white shadow"
        :style="{ left: scaledPoints[hoverIndex].x + 'px', top: scaledPoints[hoverIndex].y - 8 + 'px' }"
      >
        ${{ points[hoverIndex].balance }} · {{ formatDate(points[hoverIndex].timestamp) }}
      </div>

      <div class="mt-2 flex justify-between text-xs text-gray-400">
        <span>{{ formatDate(points[0].timestamp) }}</span>
        <span>{{ formatDate(points[points.length - 1].timestamp) }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import apiClient from '../api/client'

const props = defineProps({
  account: { type: Object, default: null },
})

const points = ref([])
const loading = ref(false)
const hoverIndex = ref(null)

const width = 600
const height = 200
const padding = 24

async function load() {
  if (!props.account?.account_number) return
  loading.value = true
  try {
    const { data } = await apiClient.get(`/api/accounts/${props.account.account_number}/balance-history`)
    points.value = data || []
  } catch {
    points.value = []
  } finally {
    loading.value = false
  }
}

const balances = computed(() => points.value.map((p) => Number(p.balance)))
const minBalance = computed(() => Math.min(...balances.value))
const maxBalance = computed(() => Math.max(...balances.value))

const scaledPoints = computed(() => {
  const n = points.value.length
  if (n === 0) return []
  const range = maxBalance.value - minBalance.value || 1
  return points.value.map((p, i) => {
    const x = padding + (i / (n - 1)) * (width - padding * 2)
    const y = height - padding - ((Number(p.balance) - minBalance.value) / range) * (height - padding * 2)
    return { x, y }
  })
})

const linePath = computed(() =>
  scaledPoints.value.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x} ${p.y}`).join(' ')
)

const areaPath = computed(() => {
  if (!scaledPoints.value.length) return ''
  const first = scaledPoints.value[0]
  const last = scaledPoints.value[scaledPoints.value.length - 1]
  return `${linePath.value} L ${last.x} ${height - padding} L ${first.x} ${height - padding} Z`
})

const gridLines = computed(() => [padding, height / 2, height - padding])

function handleMove(event) {
  if (!scaledPoints.value.length) return
  const rect = event.currentTarget.getBoundingClientRect()
  const relX = ((event.clientX - rect.left) / rect.width) * width
  let closest = 0
  let closestDist = Infinity
  scaledPoints.value.forEach((p, i) => {
    const dist = Math.abs(p.x - relX)
    if (dist < closestDist) {
      closestDist = dist
      closest = i
    }
  })
  hoverIndex.value = closest
}

function formatDate(value) {
  return new Date(value).toLocaleDateString('es', { day: '2-digit', month: 'short' })
}

onMounted(load)
watch(() => props.account?.account_number, load)
</script>
