<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useAuth } from '../stores/auth'
import { useChats } from '../stores/chats'
import { t } from '../strings'
import Avatar from './Avatar.vue'
import Logo from './Logo.vue'

const auth = useAuth()
const chats = useChats()
const router = useRouter()
const dark = ref(document.documentElement.dataset.theme !== 'light')

async function logout() {
  chats.stop()
  await auth.logout()
  router.push({ name: 'login' })
}

function toggleTheme() {
  dark.value = !dark.value
  document.documentElement.dataset.theme = dark.value ? 'dark' : 'light'
  try {
    localStorage.setItem('theme', dark.value ? 'dark' : 'light')
  } catch {
    /* ignore */
  }
}
</script>

<template>
  <nav class="rail panel" :aria-label="t.nav.chats">
    <div class="logo"><Logo :size="38" /></div>
    <RouterLink :to="{ name: 'home' }" class="item" :title="t.nav.chats" :aria-label="t.nav.chats" active-class="on">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a8 8 0 0 1-11.6 7.1L4 20l1.1-4.4A8 8 0 1 1 21 12z" /></svg>
    </RouterLink>
    <RouterLink v-if="auth.user?.role === 'admin'" :to="{ name: 'admin' }" class="item" :title="t.nav.admin" :aria-label="t.nav.admin" active-class="on">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><circle cx="9" cy="8" r="4" /><path d="M2 21a7 7 0 0 1 14 0" /><path d="M17 11a3 3 0 0 1 0 6" /><path d="M22 21a5 5 0 0 0-4-4.9" /></svg>
    </RouterLink>
    <span class="grow"></span>
    <button class="item" type="button" :title="t.nav.theme" :aria-label="t.nav.theme" @click="toggleTheme">
      <svg v-if="dark" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="4" /><path d="M12 2v2M12 20v2M2 12h2M20 12h2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" /></svg>
      <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3a9 9 0 1 0 9 9c0-.5 0-1-.1-1.4A5.5 5.5 0 0 1 12 3z" /></svg>
    </button>
    <button class="item" type="button" :title="t.nav.logout" :aria-label="t.nav.logout" @click="logout">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M10 17l5-5-5-5" /><path d="M15 12H3" /><path d="M21 3v18" /></svg>
    </button>
    <Avatar v-if="auth.user" :id="auth.user.id" :name="auth.user.display_name" :size="36" presence />
  </nav>
</template>

<style scoped>
.rail { width: var(--rail-w); flex: 0 0 var(--rail-w); display: flex; flex-direction: column; align-items: center; gap: 6px; padding: 14px 0 12px; }
.logo { display: flex; margin-bottom: 12px; }
.item { width: 44px; height: 44px; border-radius: 14px; display: flex; align-items: center; justify-content: center; color: var(--faint); border: 0; background: transparent; }
.item:hover { color: var(--ink); background: var(--chip); }
.item.on { background: var(--chip); color: var(--ink); }
.grow { flex-grow: 1; }
.avatar { margin-top: 6px; }
</style>
