<template>
  <Transition name="fade">
    <div
      v-if="message"
      class="rounded-lg border px-4 py-3 text-sm font-medium"
      :class="variant === 'error'
        ? 'border-red-200 bg-red-50 text-red-700'
        : 'border-green-200 bg-green-50 text-green-700'"
    >
      {{ message }}
    </div>
  </Transition>
</template>

<script setup>
defineProps({
  message: { type: String, default: '' },
  variant: { type: String, default: 'success' }, // 'success' | 'error'
})
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
