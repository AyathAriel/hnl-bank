<template>
  <div class="min-h-screen bg-gray-50">
    <NavBar />

    <main class="mx-auto max-w-4xl px-4 py-8">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 class="text-2xl font-bold text-gray-900">Historial de transacciones</h1>
          <p class="mt-1 text-sm text-gray-500">Todos tus movimientos, más recientes primero.</p>
        </div>

        <div class="flex items-center gap-2">
          <select
            v-model="accountFilter"
            class="rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
            @change="reload"
          >
            <option value="">Todas mis cuentas</option>
            <option v-for="acc in accounts.accounts" :key="acc.account_number" :value="acc.account_number">
              {{ acc.account_number }}
            </option>
          </select>
          <button
            class="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-600 hover:bg-gray-100 disabled:opacity-50"
            :disabled="exporting"
            @click="handleExport"
          >
            {{ exporting ? 'Exportando...' : '⬇ Exportar CSV' }}
          </button>
        </div>
      </div>

      <div class="mt-6 rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
        <TransactionList :transactions="tx.transactions" :loading="tx.loading" />

        <div v-if="totalPages > 1" class="mt-4 flex items-center justify-between border-t border-gray-100 pt-4">
          <button
            class="rounded-md px-3 py-1.5 text-sm font-medium text-gray-600 hover:bg-gray-100 disabled:opacity-40"
            :disabled="page <= 1"
            @click="goToPage(page - 1)"
          >
            ← Anterior
          </button>
          <span class="text-sm text-gray-500">Página {{ page }} de {{ totalPages }}</span>
          <button
            class="rounded-md px-3 py-1.5 text-sm font-medium text-gray-600 hover:bg-gray-100 disabled:opacity-40"
            :disabled="page >= totalPages"
            @click="goToPage(page + 1)"
          >
            Siguiente →
          </button>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import NavBar from '../components/NavBar.vue'
import TransactionList from '../components/TransactionList.vue'
import { useAccountsStore } from '../stores/accounts'
import { useTransactionsStore } from '../stores/transactions'

const accounts = useAccountsStore()
const tx = useTransactionsStore()

const accountFilter = ref('')
const page = ref(1)
const pageSize = 15
const exporting = ref(false)

async function handleExport() {
  exporting.value = true
  try {
    await tx.exportCsv(accountFilter.value)
  } finally {
    exporting.value = false
  }
}

const totalPages = computed(() => Math.max(1, Math.ceil((tx.pagination.total || 0) / pageSize)))

function load() {
  tx.fetchHistory({ account: accountFilter.value, page: page.value, pageSize })
}

function reload() {
  page.value = 1
  load()
}

function goToPage(newPage) {
  page.value = newPage
  load()
}

onMounted(() => {
  if (!accounts.accounts.length) accounts.fetchAccounts()
  load()
})
</script>
