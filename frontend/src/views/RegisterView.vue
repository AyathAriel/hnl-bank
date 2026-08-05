<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 px-4">
    <div class="w-full max-w-sm">
      <div class="text-center mb-8">
        <span class="inline-flex h-12 w-12 items-center justify-center rounded-xl bg-brand-600 text-white text-xl font-bold">H</span>
        <h1 class="mt-3 text-2xl font-bold text-gray-900">Crea tu cuenta</h1>
        <p class="text-sm text-gray-500">Se creará automáticamente una cuenta corriente</p>
      </div>

      <form class="space-y-4 rounded-xl border border-gray-200 bg-white p-6 shadow-sm" @submit.prevent="handleSubmit">
        <Toast v-if="auth.error" :message="auth.error" variant="error" />

        <div>
          <label class="block text-sm font-medium text-gray-700">Nombre completo</label>
          <input
            v-model="fullName"
            type="text"
            required
            minlength="2"
            class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
            placeholder="Ana Pérez"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700">Correo electrónico</label>
          <input
            v-model="email"
            type="email"
            required
            class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
            placeholder="tucorreo@email.com"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700">Contraseña</label>
          <input
            v-model="password"
            type="password"
            required
            minlength="8"
            class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
            placeholder="Mínimo 8 caracteres"
          />
        </div>

        <button
          type="submit"
          :disabled="auth.loading"
          class="w-full rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-60"
        >
          {{ auth.loading ? 'Creando cuenta...' : 'Crear cuenta' }}
        </button>

        <p class="text-center text-sm text-gray-500">
          ¿Ya tienes cuenta?
          <RouterLink to="/login" class="font-medium text-brand-700 hover:underline">Inicia sesión</RouterLink>
        </p>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import Toast from '../components/Toast.vue'

const fullName = ref('')
const email = ref('')
const password = ref('')
const auth = useAuthStore()
const router = useRouter()

async function handleSubmit() {
  const ok = await auth.register(email.value, password.value, fullName.value)
  if (ok) router.push({ name: 'dashboard' })
}
</script>
