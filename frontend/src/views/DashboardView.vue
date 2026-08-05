<template>
  <div class="min-h-screen bg-gray-50">
    <NavBar />

    <main class="mx-auto max-w-6xl px-4 py-8">
      <h1 class="text-2xl font-bold text-gray-900">Hola, {{ auth.user?.full_name?.split(' ')[0] }} 👋</h1>
      <p class="mt-1 text-sm text-gray-500">Este es el resumen de tus cuentas.</p>

      <div class="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div class="space-y-6 lg:col-span-2">
          <div v-if="accounts.loading" class="rounded-xl border border-gray-200 bg-white p-8 text-center text-sm text-gray-400">
            Cargando cuentas...
          </div>

          <div v-else-if="!accounts.accounts.length" class="rounded-xl border border-dashed border-gray-300 bg-white p-8 text-center">
            <p class="text-sm text-gray-500">Todavía no tienes cuentas registradas.</p>
          </div>

          <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <AccountCard v-for="acc in accounts.accounts" :key="acc.account_number" :account="acc" />
          </div>

          <BalanceChart v-if="accounts.accounts.length" :account="accounts.accounts[0]" />

          <div class="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
            <div class="flex items-center justify-between">
              <h2 class="font-semibold text-gray-900">Transacciones recientes</h2>
              <RouterLink to="/history" class="text-sm font-medium text-brand-700 hover:underline">Ver todo</RouterLink>
            </div>

            <TransactionList
              class="mt-4"
              :transactions="accounts.dashboard?.recent_transactions || []"
              :loading="accounts.loading"
            />
          </div>
        </div>

        <div class="h-[600px] lg:h-auto">
          <ChatWidget class="h-full min-h-[500px]" />
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted } from 'vue'
import NavBar from '../components/NavBar.vue'
import AccountCard from '../components/AccountCard.vue'
import BalanceChart from '../components/BalanceChart.vue'
import TransactionList from '../components/TransactionList.vue'
import ChatWidget from '../components/ChatWidget.vue'
import { useAccountsStore } from '../stores/accounts'
import { useAuthStore } from '../stores/auth'
import { useNotificationsStore } from '../stores/notifications'

const accounts = useAccountsStore()
const auth = useAuthStore()
const notifications = useNotificationsStore()

onMounted(() => {
  accounts.fetchDashboard()
  // Cuando llega una notificación en tiempo real (depósito/retiro/transferencia),
  // refresca el dashboard para reflejar el saldo actualizado sin recargar la página.
  notifications.onEvent = () => accounts.fetchDashboard()
})

onUnmounted(() => {
  notifications.onEvent = null
})
</script>
