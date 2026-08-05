import { defineStore } from 'pinia'
import apiClient, { apiErrorMessage } from '../api/client'

export const useAccountsStore = defineStore('accounts', {
  state: () => ({
    accounts: [],
    dashboard: null,
    loading: false,
    error: null,
  }),

  actions: {
    async fetchAccounts() {
      this.loading = true
      this.error = null
      try {
        const { data } = await apiClient.get('/api/accounts')
        this.accounts = data || []
      } catch (err) {
        this.error = apiErrorMessage(err, 'No se pudieron cargar las cuentas.')
      } finally {
        this.loading = false
      }
    },

    async fetchDashboard() {
      this.loading = true
      this.error = null
      try {
        const { data } = await apiClient.get('/api/dashboard')
        this.dashboard = data
        this.accounts = data.accounts || []
      } catch (err) {
        this.error = apiErrorMessage(err, 'No se pudo cargar el dashboard.')
      } finally {
        this.loading = false
      }
    },
  },
})
