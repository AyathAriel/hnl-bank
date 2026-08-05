import { defineStore } from 'pinia'
import apiClient, { apiErrorMessage } from '../api/client'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('hnl_token') || null,
    user: JSON.parse(localStorage.getItem('hnl_user') || 'null'),
    loading: false,
    error: null,
    pendingToken: null, // token intermedio mientras se espera el código 2FA
  }),

  getters: {
    isAuthenticated: (state) => !!state.token,
  },

  actions: {
    persist(token, user) {
      this.token = token
      this.user = user
      this.pendingToken = null
      localStorage.setItem('hnl_token', token)
      localStorage.setItem('hnl_user', JSON.stringify(user))
    },

    async register(email, password, fullName) {
      this.loading = true
      this.error = null
      try {
        const { data } = await apiClient.post('/api/auth/register', {
          email,
          password,
          full_name: fullName,
        })
        this.persist(data.token, data.user)
        return true
      } catch (err) {
        this.error = apiErrorMessage(err, 'No se pudo completar el registro.')
        return false
      } finally {
        this.loading = false
      }
    },

    // Devuelve 'ok' (sesión iniciada), 'totp' (falta el segundo factor) o
    // 'error' (credenciales inválidas u otro problema, ver this.error).
    async login(email, password) {
      this.loading = true
      this.error = null
      try {
        const { data } = await apiClient.post('/api/auth/login', { email, password })
        if (data.requires_totp) {
          this.pendingToken = data.pending_token
          return 'totp'
        }
        this.persist(data.token, data.user)
        return 'ok'
      } catch (err) {
        this.error = apiErrorMessage(err, 'Credenciales inválidas.')
        return 'error'
      } finally {
        this.loading = false
      }
    },

    async verifyTwoFactor(code) {
      this.loading = true
      this.error = null
      try {
        const { data } = await apiClient.post('/api/auth/2fa/verify', {
          pending_token: this.pendingToken,
          code,
        })
        this.persist(data.token, data.user)
        return true
      } catch (err) {
        this.error = apiErrorMessage(err, 'Código incorrecto.')
        return false
      } finally {
        this.loading = false
      }
    },

    async logout() {
      try {
        await apiClient.post('/api/auth/logout')
      } catch {
        // aunque falle en el servidor, igual limpiamos la sesión local
      }
      this.token = null
      this.user = null
      this.pendingToken = null
      localStorage.removeItem('hnl_token')
      localStorage.removeItem('hnl_user')
    },
  },
})
