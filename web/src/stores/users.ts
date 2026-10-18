import { defineStore } from 'pinia'
import { ref } from 'vue'
import { get } from '../api/http'
import type { User } from '../types'

/** Directory cache: id → user, for names and avatars everywhere. */
export const useUsers = defineStore('users', () => {
  const byId = ref<Record<number, User>>({})
  const loaded = ref(false)

  async function load() {
    const list = await get<User[]>('/api/users')
    const map: Record<number, User> = {}
    for (const u of list) map[u.id] = u
    byId.value = map
    loaded.value = true
    return list
  }

  function get_(id: number): User | undefined {
    return byId.value[id]
  }

  function name(id: number): string {
    return byId.value[id]?.display_name ?? `#${id}`
  }

  function isBot(id: number): boolean {
    return byId.value[id]?.kind === 'bot'
  }

  return { byId, loaded, load, get: get_, name, isBot }
})
