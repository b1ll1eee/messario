import { defineStore } from 'pinia'
import { ref } from 'vue'
import { get, post, ApiError } from '../api/http'
import type { User } from '../types'

export const useAuth = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const checked = ref(false)

  async function load() {
    try {
      user.value = await get<User>('/api/me')
    } catch (e) {
      if (e instanceof ApiError && (e.status === 401 || e.status === 403)) user.value = null
      else throw e
    } finally {
      checked.value = true
    }
  }

  async function login(username: string, password: string) {
    user.value = await post<User>('/api/auth/login', { username, password })
  }

  async function logout() {
    await post('/api/auth/logout')
    user.value = null
  }

  return { user, checked, load, login, logout }
})
