<template>
  <div class="min-h-screen bg-gray-50">
    <NavBar />

    <main class="mx-auto max-w-lg px-4 py-8">
      <h1 class="text-2xl font-bold text-gray-900">Seguridad</h1>
      <p class="mt-1 text-sm text-gray-500">Autenticación de dos factores (2FA) para tu cuenta.</p>

      <div class="mt-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <Toast v-if="twofa.error" :message="twofa.error" variant="error" class="mb-4" />
        <Toast v-if="twofa.success" :message="twofa.success" variant="success" class="mb-4" />

        <div v-if="loadingStatus" class="py-6 text-center text-sm text-gray-400">Cargando...</div>

        <!-- 2FA ya activado: solo mostrar opción de desactivar -->
        <div v-else-if="twofa.enabled && !settingUp" class="space-y-4">
          <div class="flex items-center gap-2 rounded-lg bg-green-50 px-4 py-3 text-sm text-green-700">
            <span>✅</span> Tienes 2FA activado en tu cuenta.
          </div>

          <form class="space-y-3" @submit.prevent="handleDisable">
            <label class="block text-sm font-medium text-gray-700">Contraseña actual (para desactivar)</label>
            <input
              v-model="disablePassword"
              type="password"
              required
              class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
              placeholder="••••••••"
            />
            <button
              type="submit"
              :disabled="twofa.loading"
              class="w-full rounded-lg bg-red-600 px-4 py-2 text-sm font-semibold text-white hover:bg-red-700 disabled:opacity-60"
            >
              {{ twofa.loading ? 'Desactivando...' : 'Desactivar 2FA' }}
            </button>
          </form>
        </div>

        <!-- Aún no activado y no se inició el setup -->
        <div v-else-if="!settingUp" class="space-y-4 text-center">
          <p class="text-sm text-gray-500">
            2FA agrega una capa extra de seguridad: además de tu contraseña, se pedirá un código de
            tu app autenticadora al iniciar sesión.
          </p>
          <button
            :disabled="twofa.loading"
            class="w-full rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-60"
            @click="handleStartSetup"
          >
            {{ twofa.loading ? 'Generando...' : 'Activar 2FA' }}
          </button>
        </div>

        <!-- Setup en curso: mostrar QR + confirmar código -->
        <div v-else class="space-y-4">
          <p class="text-sm text-gray-500">
            Escanea este código QR con tu app autenticadora (Google Authenticator, Authy, Microsoft
            Authenticator, etc.):
          </p>
          <div class="flex justify-center">
            <img :src="twofa.setupQRCodeDataURL" alt="Código QR de 2FA" class="h-48 w-48 rounded-lg border border-gray-200" />
          </div>
          <p class="text-center text-xs text-gray-400">
            ¿No puedes escanear? Ingresa este código manualmente:
            <br />
            <span class="font-mono text-gray-600">{{ twofa.setupSecret }}</span>
          </p>

          <form class="space-y-3" @submit.prevent="handleConfirm">
            <label class="block text-sm font-medium text-gray-700">Código de 6 dígitos generado por la app</label>
            <input
              v-model="confirmCode"
              type="text"
              inputmode="numeric"
              maxlength="6"
              required
              class="w-full rounded-lg border border-gray-300 px-3 py-2 text-center text-lg tracking-[0.5em] font-mono focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
              placeholder="000000"
            />
            <button
              type="submit"
              :disabled="twofa.loading || confirmCode.length !== 6"
              class="w-full rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-60"
            >
              {{ twofa.loading ? 'Confirmando...' : 'Confirmar y activar' }}
            </button>
            <button type="button" class="w-full text-center text-sm text-gray-500 hover:underline" @click="settingUp = false">
              Cancelar
            </button>
          </form>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import NavBar from '../components/NavBar.vue'
import Toast from '../components/Toast.vue'
import { useTwoFactorStore } from '../stores/twofa'

const twofa = useTwoFactorStore()
const loadingStatus = ref(true)
const settingUp = ref(false)
const confirmCode = ref('')
const disablePassword = ref('')

onMounted(async () => {
  await twofa.fetchStatus()
  loadingStatus.value = false
})

async function handleStartSetup() {
  const ok = await twofa.startSetup()
  if (ok) settingUp.value = true
}

async function handleConfirm() {
  const ok = await twofa.confirmEnable(confirmCode.value)
  if (ok) {
    settingUp.value = false
    confirmCode.value = ''
  }
}

async function handleDisable() {
  const ok = await twofa.disable(disablePassword.value)
  if (ok) disablePassword.value = ''
}
</script>
