import { defineStore } from 'pinia'
import apiClient, { apiErrorMessage } from '../api/client'

export const useChatStore = defineStore('chat', {
  state: () => ({
    conversationId: '',
    messages: [
      {
        role: 'assistant',
        text: 'Hola, soy tu asistente bancario. Puedo ayudarte a consultar tu saldo, ver tu historial o hacer depósitos, retiros y transferencias. ¿En qué te ayudo?',
      },
    ],
    loading: false,
    error: null,
  }),

  actions: {
    async sendMessage(text) {
      this.messages.push({ role: 'user', text })
      this.loading = true
      this.error = null
      try {
        const { data } = await apiClient.post('/api/chat', {
          conversation_id: this.conversationId,
          message: text,
        })
        this.conversationId = data.conversation_id
        this.messages.push({ role: 'assistant', text: data.reply })
      } catch (err) {
        const message = apiErrorMessage(err, 'No se pudo contactar al asistente.')
        this.error = message
        this.messages.push({ role: 'assistant', text: `⚠️ ${message}` })
      } finally {
        this.loading = false
      }
    },
  },
})
