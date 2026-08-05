<template>
  <div>
    <div v-if="loading" class="py-8 text-center text-sm text-gray-400">Cargando movimientos...</div>

    <div v-else-if="!transactions.length" class="py-8 text-center text-sm text-gray-400">
      No hay transacciones para mostrar todavía.
    </div>

    <ul v-else class="divide-y divide-gray-100">
      <li v-for="tx in transactions" :key="tx.id" class="flex items-start justify-between gap-3 py-3">
        <div class="flex min-w-0 items-center gap-3">
          <span
            class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-sm"
            :class="isIncoming(tx) ? 'bg-green-50 text-green-600' : 'bg-red-50 text-red-600'"
          >
            {{ isIncoming(tx) ? '↓' : '↑' }}
          </span>
          <div class="min-w-0">
            <p class="text-sm font-medium text-gray-900">{{ typeLabel(tx.type) }}</p>
            <p class="truncate text-xs text-gray-400">
              {{ tx.description || 'Sin descripción' }} · {{ formatDate(tx.created_at) }}
            </p>
            <p class="hidden truncate font-mono text-xs text-gray-400 sm:block">
              {{ tx.from_account_number }} → {{ tx.to_account_number }}
            </p>
          </div>
        </div>
        <p class="shrink-0 text-sm font-semibold" :class="isIncoming(tx) ? 'text-green-600' : 'text-red-600'">
          {{ isIncoming(tx) ? '+' : '-' }}${{ tx.amount }}
        </p>
      </li>
    </ul>
  </div>
</template>

<script setup>
import { useAccountsStore } from '../stores/accounts'

const props = defineProps({
  transactions: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
})

const accounts = useAccountsStore()

const typeLabels = {
  deposit: 'Depósito',
  withdrawal: 'Retiro',
  transfer: 'Transferencia',
  internal_transfer: 'Transferencia entre cuentas propias',
}

function typeLabel(type) {
  return typeLabels[type] || type
}

function isIncoming(tx) {
  const ownNumbers = accounts.accounts.map((a) => a.account_number)
  if (tx.type === 'deposit') return true
  if (tx.type === 'withdrawal') return false
  return ownNumbers.includes(tx.to_account_number)
}

function formatDate(value) {
  if (!value) return ''
  const date = new Date(value)
  return date.toLocaleString('es', { dateStyle: 'medium', timeStyle: 'short' })
}
</script>
