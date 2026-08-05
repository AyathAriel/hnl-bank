<template>
  <div class="flex h-full flex-col rounded-xl border border-gray-200 bg-white shadow-sm">
    <div class="flex items-center gap-2 border-b border-gray-100 px-4 py-3">
      <span class="inline-flex h-8 w-8 items-center justify-center rounded-full bg-brand-100 text-brand-700">🤖</span>
      <div>
        <p class="text-sm font-semibold text-gray-900">Asistente HNL</p>
        <p class="text-xs text-gray-400">Pregunta por tu saldo, historial o pide una operación</p>
      </div>
    </div>

    <div ref="scrollArea" class="flex-1 space-y-3 overflow-y-auto px-4 py-3">
      <div v-for="(m, i) in chat.messages" :key="i" class="flex" :class="m.role === 'user' ? 'justify-end' : 'justify-start'">
        <div
          class="max-w-[85%] whitespace-pre-wrap rounded-2xl px-3 py-2 text-sm"
          :class="m.role === 'user' ? 'bg-brand-600 text-white' : 'bg-gray-100 text-gray-800'"
        >
          {{ m.text }}
        </div>
      </div>
      <div v-if="chat.loading" class="flex justify-start">
        <div class="rounded-2xl bg-gray-100 px-3 py-2 text-sm text-gray-400">Escribiendo...</div>
      </div>
    </div>

    <form class="flex items-center gap-2 border-t border-gray-100 p-3" @submit.prevent="handleSend">
      <input
        v-model="input"
        type="text"
        placeholder="Ej: ¿cuánto dinero tengo?"
        class="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
      />
      <button
        type="submit"
        :disabled="chat.loading || !input.trim()"
        class="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50"
      >
        Enviar
      </button>
    </form>
  </div>
</template>

<script setup>
import { nextTick, ref, watch } from 'vue'
import { useChatStore } from '../stores/chat'

const chat = useChatStore()
const input = ref('')
const scrollArea = ref(null)

async function handleSend() {
  const text = input.value.trim()
  if (!text) return
  input.value = ''
  await chat.sendMessage(text)
}

watch(
  () => chat.messages.length,
  async () => {
    await nextTick()
    if (scrollArea.value) {
      scrollArea.value.scrollTop = scrollArea.value.scrollHeight
    }
  }
)
</script>
