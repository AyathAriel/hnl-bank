<template>
  <nav class="bg-white border-b border-gray-200">
    <div class="max-w-6xl mx-auto px-4 flex items-center justify-between h-16">
      <RouterLink to="/" class="flex items-center gap-2 font-bold text-brand-700 text-lg">
        <span class="inline-flex h-8 w-8 items-center justify-center rounded-lg bg-brand-600 text-white">H</span>
        HNL Bank
      </RouterLink>

      <div class="hidden sm:flex items-center gap-1">
        <RouterLink
          v-for="link in links"
          :key="link.to"
          :to="link.to"
          class="px-3 py-2 rounded-md text-sm font-medium text-gray-600 hover:bg-gray-100"
          active-class="!text-brand-700 !bg-brand-50"
        >
          {{ link.label }}
        </RouterLink>
      </div>

      <div class="flex items-center gap-3">
        <span class="hidden sm:block text-sm text-gray-500">{{ auth.user?.full_name }}</span>
        <button
          class="text-sm font-medium text-gray-600 hover:text-red-600 px-3 py-2 rounded-md hover:bg-red-50"
          @click="handleLogout"
        >
          Cerrar sesión
        </button>
      </div>
    </div>

    <div class="sm:hidden flex items-center justify-around border-t border-gray-100">
      <RouterLink
        v-for="link in links"
        :key="link.to"
        :to="link.to"
        class="flex-1 text-center py-2 text-xs font-medium text-gray-600"
        active-class="!text-brand-700"
      >
        {{ link.label }}
      </RouterLink>
    </div>
  </nav>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const links = [
  { to: '/', label: 'Dashboard' },
  { to: '/transactions', label: 'Transacciones' },
  { to: '/history', label: 'Historial' },
]

const auth = useAuthStore()
const router = useRouter()

async function handleLogout() {
  await auth.logout()
  router.push({ name: 'login' })
}
</script>
