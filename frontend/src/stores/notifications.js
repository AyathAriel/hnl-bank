import { defineStore } from 'pinia'

// Bonus: notificaciones en tiempo real vía WebSocket. Se conecta con el JWT
// como query param (el navegador no permite headers personalizados al abrir
// un WebSocket) y muestra un toast efímero por cada evento de transacción.
export const useNotificationsStore = defineStore('notifications', {
  state: () => ({
    socket: null,
    connected: false,
    toasts: [],
    onEvent: null, // callback opcional que se dispara con cada evento recibido
  }),

  actions: {
    connect(token) {
      if (this.socket || !token) return

      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const url = `${protocol}//${window.location.host}/api/ws?token=${encodeURIComponent(token)}`

      try {
        this.socket = new WebSocket(url)
      } catch {
        return
      }

      this.socket.onopen = () => {
        this.connected = true
      }
      this.socket.onclose = () => {
        this.connected = false
        this.socket = null
      }
      this.socket.onerror = () => {
        this.connected = false
      }
      this.socket.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          this.pushToast(data.message)
          if (this.onEvent) this.onEvent(data)
        } catch {
          // ignora mensajes que no sean JSON válido
        }
      }
    },

    disconnect() {
      this.socket?.close()
      this.socket = null
      this.connected = false
    },

    pushToast(message) {
      const id = Date.now() + Math.random()
      this.toasts.push({ id, message })
      setTimeout(() => {
        this.toasts = this.toasts.filter((t) => t.id !== id)
      }, 5000)
    },
  },
})
