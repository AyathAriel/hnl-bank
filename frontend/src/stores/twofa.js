import { defineStore } from 'pinia'
import apiClient, { apiErrorMessage } from '../api/client'

export const useTwoFactorStore = defineStore('twofa', {
  state: () => ({
    enabled: false,
    loading: false,
    error: null,
    success: null,
    setupSecret: '',
    setupQRCodeDataURL: '',
  }),

  actions: {
    async fetchStatus() {
      try {
        const { data } = await apiClient.get('/api/auth/2fa/status')
        this.enabled = data.enabled
      } catch {
        // si falla la consulta, se asume desactivado; no es crítico
      }
    },

    async startSetup() {
      this.loading = true
      this.error = null
      this.success = null
      try {
        const { data } = await apiClient.post('/api/auth/2fa/setup')
        this.setupSecret = data.secret
        this.setupQRCodeDataURL = data.qr_code_data_url
        return true
      } catch (err) {
        this.error = apiErrorMessage(err, 'No se pudo iniciar la configuración de 2FA.')
        return false
      } finally {
        this.loading = false
      }
    },

    async confirmEnable(code) {
      this.loading = true
      this.error = null
      this.success = null
      try {
        await apiClient.post('/api/auth/2fa/enable', { code })
        this.enabled = true
        this.setupSecret = ''
        this.setupQRCodeDataURL = ''
        this.success = '2FA activado correctamente.'
        return true
      } catch (err) {
        this.error = apiErrorMessage(err, 'Código incorrecto.')
        return false
      } finally {
        this.loading = false
      }
    },

    async disable(password) {
      this.loading = true
      this.error = null
      this.success = null
      try {
        await apiClient.post('/api/auth/2fa/disable', { password })
        this.enabled = false
        this.success = '2FA desactivado.'
        return true
      } catch (err) {
        this.error = apiErrorMessage(err, 'No se pudo desactivar 2FA.')
        return false
      } finally {
        this.loading = false
      }
    },
  },
})
