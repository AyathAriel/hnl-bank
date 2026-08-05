<template>
  <form class="space-y-4" @submit.prevent="handleSubmit">
    <Toast v-if="tx.actionError" :message="tx.actionError" variant="error" />
    <Toast v-if="tx.actionSuccess" :message="tx.actionSuccess" variant="success" />

    <div v-if="type === 'transfer'">
      <label class="block text-sm font-medium text-gray-700">Cuenta origen</label>
      <select
        v-model="fromAccount"
        required
        class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
      >
        <option v-for="acc in accounts.accounts" :key="acc.account_number" :value="acc.account_number">
          {{ acc.account_number }} (${{ acc.balance }})
        </option>
      </select>
    </div>
    <div v-else>
      <label class="block text-sm font-medium text-gray-700">Cuenta</label>
      <select
        v-model="fromAccount"
        required
        class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
      >
        <option v-for="acc in accounts.accounts" :key="acc.account_number" :value="acc.account_number">
          {{ acc.account_number }} (${{ acc.balance }})
        </option>
      </select>
    </div>

    <div v-if="type === 'transfer'">
      <label class="block text-sm font-medium text-gray-700">Cuenta destino</label>
      <input
        v-model="toAccount"
        type="text"
        required
        placeholder="Número de cuenta destino"
        class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
      />
    </div>

    <div>
      <label class="block text-sm font-medium text-gray-700">Monto (USD)</label>
      <input
        v-model="amount"
        type="number"
        min="0.01"
        step="0.01"
        required
        placeholder="0.00"
        class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
      />
    </div>

    <div>
      <label class="block text-sm font-medium text-gray-700">Descripción (opcional)</label>
      <input
        v-model="description"
        type="text"
        maxlength="280"
        class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
      />
    </div>

    <button
      type="submit"
      :disabled="tx.actionLoading"
      class="w-full rounded-lg px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
      :class="buttonClass"
    >
      {{ tx.actionLoading ? 'Procesando...' : actionLabel }}
    </button>
  </form>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useAccountsStore } from '../stores/accounts'
import { useTransactionsStore } from '../stores/transactions'
import Toast from './Toast.vue'

const props = defineProps({
  type: { type: String, required: true }, // 'deposit' | 'withdraw' | 'transfer'
})

const emit = defineEmits(['completed'])

const accounts = useAccountsStore()
const tx = useTransactionsStore()

const fromAccount = ref('')
const toAccount = ref('')
const amount = ref('')
const description = ref('')

onMounted(() => {
  tx.actionError = null
  tx.actionSuccess = null
})

watch(
  () => accounts.accounts,
  (list) => {
    if (list.length && !fromAccount.value) fromAccount.value = list[0].account_number
  },
  { immediate: true }
)

const labels = { deposit: 'Depositar', withdraw: 'Retirar', transfer: 'Transferir' }
const actionLabel = computed(() => labels[props.type])

const buttonClass = computed(() => {
  if (props.type === 'withdraw') return 'bg-red-600 hover:bg-red-700'
  if (props.type === 'transfer') return 'bg-brand-600 hover:bg-brand-700'
  return 'bg-green-600 hover:bg-green-700'
})

async function handleSubmit() {
  let result = null
  if (props.type === 'deposit') {
    result = await tx.deposit(fromAccount.value, amount.value, description.value)
  } else if (props.type === 'withdraw') {
    result = await tx.withdraw(fromAccount.value, amount.value, description.value)
  } else {
    result = await tx.transfer(fromAccount.value, toAccount.value, amount.value, description.value)
  }

  if (result) {
    amount.value = ''
    description.value = ''
    toAccount.value = ''
    await accounts.fetchAccounts()
    emit('completed')
  }
}
</script>
