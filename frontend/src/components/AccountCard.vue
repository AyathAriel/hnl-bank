<template>
  <div class="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
    <div class="flex items-start justify-between">
      <div>
        <p class="text-xs font-medium uppercase tracking-wide text-gray-400">{{ typeLabel }}</p>
        <p class="mt-1 font-mono text-sm text-gray-500">{{ account.account_number }}</p>
      </div>
      <span class="rounded-full bg-brand-50 px-2 py-1 text-xs font-medium text-brand-700">{{ account.currency }}</span>
    </div>
    <p class="mt-4 text-3xl font-bold text-gray-900">${{ account.balance }}</p>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  account: { type: Object, required: true },
})

const typeLabels = {
  checking: 'Cuenta corriente',
  savings: 'Cuenta de ahorros',
  investment: 'Cuenta de inversión',
}

const typeLabel = computed(() => typeLabels[props.account.account_type] || props.account.account_type)
</script>
