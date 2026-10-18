<script setup lang="ts">
import { RouterLink, useRouter } from 'vue-router'
import { useAuth } from '../stores/auth'
import { useChats } from '../stores/chats'
import { t } from '../strings'
import Avatar from './Avatar.vue'

const auth = useAuth()
const chats = useChats()
const router = useRouter()

async function logout() {
  chats.stop()
  await auth.logout()
  router.push({ name: 'login' })
}

function toggleTheme() {
  const root = document.documentElement
  const dark = root.dataset.theme === 'dark' || (!root.dataset.theme && matchMedia('(prefers-color-scheme: dark)').matches)
  root.dataset.theme = dark ? 'light' : 'dark'
  try {
    localStorage.setItem('theme', root.dataset.theme)
  } catch {
    /* ignore */
  }
}
</script>

<template>
  <nav class="rail" :aria-label="t.nav.chats">
    <div class="logo" aria-hidden="true">N</div>
    <RouterLink :to="{ name: 'home' }" class="item" :title="t.nav.chats" :aria-label="t.nav.chats" active-class="on">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a8 8 0 0 1-11.6 7.1L4 20l1.1-4.4A8 8 0 1 1 21 12z" /></svg>
    </RouterLink>
    <RouterLink v-if="auth.user?.role === 'admin'" :to="{ name: 'admin' }" class="item" :title="t.nav.admin" :aria-label="t.nav.admin" active-class="on">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="9" cy="8" r="4" /><path d="M2 21a7 7 0 0 1 14 0" /><path d="M17 11a3 3 0 0 1 0 6" /><path d="M22 21a5 5 0 0 0-4-4.9" /></svg>
    </RouterLink>
    <span class="grow"></span>
    <button class="item" type="button" :title="t.nav.theme" :aria-label="t.nav.theme" @click="toggleTheme">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3a9 9 0 1 0 9 9c0-.5 0-1-.1-1.4A5.5 5.5 0 0 1 12 3z" /></svg>
    </button>
    <button class="item" type="button" :title="t.nav.logout" :aria-label="t.nav.logout" @click="logout">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M10 17l5-5-5-5" /><path d="M15 12H3" /><path d="M21 3v18" /></svg>
    </button>
    <Avatar v-if="auth.user" :id="auth.user.id" :name="auth.user.display_name" size="sm" />
  </nav>
</template>

<style scoped>
.rail { width: var(--rail-w); flex: 0 0 var(--rail-w); display: flex; flex-direction: column; align-items: center; gap: 6px; padding: 14px 0; border-right: 1px solid var(--border); background: var(--surface-2); }
.logo { width: 34px; height: 34px; border-radius: 10px; background: var(--accent); color: var(--accent-ink); display: flex; align-items: center; justify-content: center; font-weight: 700; font-size: 17px; margin-bottom: 8px; }
.item { width: 44px; height: 44px; border-radius: 10px; display: flex; align-items: center; justify-content: center; color: var(--muted); border: 1px solid transparent; background: transparent; }
.item:hover { color: var(--ink); }
.item.on { background: var(--surface); border-color: var(--border); color: var(--ink); }
.grow { flex-grow: 1; }
</style>
