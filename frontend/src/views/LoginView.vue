<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 px-4">
    <div class="w-full max-w-sm">
      <div class="text-center mb-8">
        <span class="inline-flex h-12 w-12 items-center justify-center rounded-xl bg-brand-600 text-white text-xl font-bold">H</span>
        <h1 class="mt-3 text-2xl font-bold text-gray-900">HNL Bank</h1>
        <p class="text-sm text-gray-500">{{ step === 'credentials' ? 'Inicia sesión en tu cuenta' : 'Verificación en dos pasos' }}</p>
      </div>

      <form
        v-if="step === 'credentials'"
        class="space-y-4 rounded-xl border border-gray-200 bg-white p-6 shadow-sm"
        @submit.prevent="handleSubmit"
      >
        <Toast v-if="auth.error" :message="auth.error" variant="error" />

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
            class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
            placeholder="••••••••"
          />
        </div>

        <button
          type="submit"
          :disabled="auth.loading"
          class="w-full rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-60"
        >
          {{ auth.loading ? 'Ingresando...' : 'Ingresar' }}
        </button>

        <p class="text-center text-sm text-gray-500">
          ¿No tienes cuenta?
          <RouterLink to="/register" class="font-medium text-brand-700 hover:underline">Regístrate</RouterLink>
        </p>
      </form>

      <form v-else class="space-y-4 rounded-xl border border-gray-200 bg-white p-6 shadow-sm" @submit.prevent="handleVerify">
        <Toast v-if="auth.error" :message="auth.error" variant="error" />

        <p class="text-sm text-gray-500">
          Ingresa el código de 6 dígitos de tu app autenticadora (Google Authenticator, Authy, etc.).
        </p>

        <div>
          <label class="block text-sm font-medium text-gray-700">Código de verificación</label>
          <input
            v-model="code"
            type="text"
            inputmode="numeric"
            pattern="[0-9]{6}"
            maxlength="6"
            required
            autofocus
            class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-center text-lg tracking-[0.5em] font-mono focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
            placeholder="000000"
          />
        </div>

        <button
          type="submit"
          :disabled="auth.loading || code.length !== 6"
          class="w-full rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-60"
        >
          {{ auth.loading ? 'Verificando...' : 'Verificar' }}
        </button>

        <button type="button" class="w-full text-center text-sm text-gray-500 hover:underline" @click="step = 'credentials'">
          ← Volver
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import Toast from '../components/Toast.vue'

const email = ref('')
const password = ref('')
const code = ref('')
const step = ref('credentials') // 'credentials' | 'totp'
const auth = useAuthStore()
const router = useRouter()

async function handleSubmit() {
  const result = await auth.login(email.value, password.value)
  if (result === 'ok') router.push({ name: 'dashboard' })
  else if (result === 'totp') step.value = 'totp'
}

async function handleVerify() {
  const ok = await auth.verifyTwoFactor(code.value)
  if (ok) router.push({ name: 'dashboard' })
}
</script>
