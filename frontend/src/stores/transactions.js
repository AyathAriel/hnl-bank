import { defineStore } from 'pinia'
import apiClient, { apiErrorMessage } from '../api/client'

export const useTransactionsStore = defineStore('transactions', {
  state: () => ({
    transactions: [],
    pagination: { page: 1, page_size: 20, total: 0 },
    loading: false,
    error: null,
    actionLoading: false,
    actionError: null,
    actionSuccess: null,
  }),

  actions: {
    async fetchHistory({ account = '', page = 1, pageSize = 20 } = {}) {
      this.loading = true
      this.error = null
      try {
        const { data } = await apiClient.get('/api/transactions', {
          params: { account: account || undefined, page, page_size: pageSize },
        })
        this.transactions = data.transactions
        this.pagination = data.pagination
      } catch (err) {
        this.error = apiErrorMessage(err, 'No se pudo cargar el historial.')
      } finally {
        this.loading = false
      }
    },

    async deposit(accountNumber, amount, description) {
      return this.runAction('/api/transactions/deposit', {
        account_number: accountNumber,
        amount: String(amount),
        description,
      })
    },

    async withdraw(accountNumber, amount, description) {
      return this.runAction('/api/transactions/withdraw', {
        account_number: accountNumber,
        amount: String(amount),
        description,
      })
    },

    async transfer(fromAccountNumber, toAccountNumber, amount, description) {
      return this.runAction('/api/transactions/transfer', {
        from_account_number: fromAccountNumber,
        to_account_number: toAccountNumber,
        amount: String(amount),
        description,
      })
    },

    async exportCsv(account = '') {
      const response = await apiClient.get('/api/transactions/export', {
        params: { account: account || undefined },
        responseType: 'blob',
      })
      const url = window.URL.createObjectURL(new Blob([response.data], { type: 'text/csv' }))
      const link = document.createElement('a')
      link.href = url
      link.download = 'historial-hnl-bank.csv'
      document.body.appendChild(link)
      link.click()
      link.remove()
      window.URL.revokeObjectURL(url)
    },

    async runAction(url, payload) {
      this.actionLoading = true
      this.actionError = null
      this.actionSuccess = null
      try {
        const { data } = await apiClient.post(url, payload)
        this.actionSuccess = 'Operación realizada con éxito.'
        return data
      } catch (err) {
        this.actionError = apiErrorMessage(err, 'No se pudo completar la operación.')
        return null
      } finally {
        this.actionLoading = false
      }
    },
  },
})
