<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ApiError } from '../api/http'
import { useAuth } from '../stores/auth'
import { t } from '../strings'
import Logo from '../components/Logo.vue'

const auth = useAuth()
const router = useRouter()
const route = useRoute()
const username = ref('')
const password = ref('')
const error = ref('')
const busy = ref(false)

async function submit() {
  error.value = ''
  busy.value = true
  try {
    await auth.login(username.value, password.value)
    const next = typeof route.query.next === 'string' && route.query.next.startsWith('/') ? route.query.next : '/'
    router.replace(next)
  } catch (e) {
    if (e instanceof ApiError && e.code === 'suspended') error.value = t.login.suspended
    else if (e instanceof ApiError && e.status === 401) error.value = t.login.failed
    else error.value = (e as Error).message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <main class="login">
    <form class="card panel" @submit.prevent="submit">
      <div class="brand"><Logo :size="44" /></div>
      <h1>{{ t.login.title }}</h1>
      <p class="muted">{{ t.appName }}</p>
      <label>
        <span class="label">{{ t.login.username }}</span>
        <input v-model="username" class="input" autocomplete="username" autocapitalize="off" required autofocus />
      </label>
      <label>
        <span class="label">{{ t.login.password }}</span>
        <input v-model="password" class="input" type="password" autocomplete="current-password" required />
      </label>
      <p v-if="error" class="error" role="alert">{{ error }}</p>
      <button class="btn primary" type="submit" :disabled="busy">{{ t.login.submit }}</button>
    </form>
  </main>
</template>

<style scoped>
.login { min-height: 100%; display: flex; align-items: center; justify-content: center; padding: 24px 16px; }
.card { width: 100%; max-width: 400px; padding: 32px; display: flex; flex-direction: column; gap: 14px; box-shadow: var(--shadow); }
.brand { display: flex; margin-bottom: 6px; }
h1 { font-size: 26px; font-weight: 800; letter-spacing: -0.03em; }
.card > .muted { margin-top: -10px; font-weight: 600; }
label { display: flex; flex-direction: column; gap: 6px; }
.btn { height: 46px; margin-top: 6px; font-size: 15px; }
</style>
