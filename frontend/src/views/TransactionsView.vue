<template>
  <div class="min-h-screen bg-gray-50">
    <NavBar />

    <main class="mx-auto max-w-2xl px-4 py-8">
      <h1 class="text-2xl font-bold text-gray-900">Transacciones</h1>
      <p class="mt-1 text-sm text-gray-500">Deposita, retira o transfiere fondos entre cuentas.</p>

      <div class="mt-6 flex gap-2 rounded-lg bg-gray-100 p-1">
        <button
          v-for="tab in tabs"
          :key="tab.value"
          class="flex-1 rounded-md py-2 text-sm font-medium transition"
          :class="activeTab === tab.value ? 'bg-white text-brand-700 shadow-sm' : 'text-gray-500 hover:text-gray-700'"
          @click="activeTab = tab.value"
        >
          {{ tab.label }}
        </button>
      </div>

      <div class="mt-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <TransactionForm :key="activeTab" :type="activeTab" />
      </div>
    </main>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import NavBar from '../components/NavBar.vue'
import TransactionForm from '../components/TransactionForm.vue'
import { useAccountsStore } from '../stores/accounts'

const tabs = [
  { value: 'deposit', label: 'Depositar' },
  { value: 'withdraw', label: 'Retirar' },
  { value: 'transfer', label: 'Transferir' },
]

const activeTab = ref('deposit')
const accounts = useAccountsStore()

onMounted(() => {
  if (!accounts.accounts.length) accounts.fetchAccounts()
})
</script>
