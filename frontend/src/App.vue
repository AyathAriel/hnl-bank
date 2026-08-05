<template>
  <RouterView />
  <NotificationToasts />
</template>

<script setup>
import { watch } from 'vue'
import { RouterView } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { useNotificationsStore } from './stores/notifications'
import NotificationToasts from './components/NotificationToasts.vue'

const auth = useAuthStore()
const notifications = useNotificationsStore()

// Conecta/desconecta el WebSocket de notificaciones según el estado de sesión,
// para que funcione tanto al cargar ya autenticado como al hacer login/logout.
watch(
  () => auth.token,
  (token) => {
    if (token) {
      notifications.connect(token)
    } else {
      notifications.disconnect()
    }
  },
  { immediate: true }
)
</script>
