<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { get, patch, post } from '../api/http'
import { useUsers } from '../stores/users'
import { fullTime } from '../format'
import { t } from '../strings'
import type { LicenseStatus, User } from '../types'
import Avatar from '../components/Avatar.vue'

interface AuditEntry { id: number; actor_id: number; action: string; target: string; meta?: Record<string, unknown>; created_at: string }

const users = useUsers()
const tab = ref<'users' | 'bots' | 'audit'>('users')
const error = ref('')
const audit = ref<AuditEntry[]>([])

const people = computed(() => Object.values(users.byId).filter((u) => u.kind === 'user').sort((a, b) => a.display_name.localeCompare(b.display_name, 'th')))
const bots = computed(() => Object.values(users.byId).filter((u) => u.kind === 'bot').sort((a, b) => a.display_name.localeCompare(b.display_name, 'th')))

const newUser = ref({ username: '', display_name: '', email: '', department: '', title: '', role: 'member', password: '' })
const newBot = ref({ username: '', display_name: '' })
const token = ref<{ name: string; value: string } | null>(null)
const copied = ref(false)
const license = ref<LicenseStatus | null>(null)

onMounted(() => {
  void users.load()
  void get<LicenseStatus>('/api/admin/license').then((l) => (license.value = l)).catch(() => {})
})

async function run(fn: () => Promise<void>) {
  error.value = ''
  try {
    await fn()
  } catch (e) {
    error.value = (e as Error).message
  }
}

const createUser = () =>
  run(async () => {
    await post<User>('/api/admin/users', newUser.value)
    newUser.value = { username: '', display_name: '', email: '', department: '', title: '', role: 'member', password: '' }
    await users.load()
  })

const setStatus = (u: User, status: 'active' | 'suspended') =>
  run(async () => {
    await patch(`/api/admin/users/${u.id}`, { status })
    await users.load()
  })

const createBot = () =>
  run(async () => {
    const r = await post<{ bot: User; token: string }>('/api/admin/bots', newBot.value)
    token.value = { name: r.bot.display_name, value: r.token }
    newBot.value = { username: '', display_name: '' }
    await users.load()
  })

const regenerate = (b: User) =>
  run(async () => {
    const r = await post<{ token: string }>(`/api/admin/bots/${b.id}/token`)
    token.value = { name: b.display_name, value: r.token }
  })

const loadAudit = () =>
  run(async () => {
    audit.value = await get<AuditEntry[]>('/api/admin/audit?limit=200')
  })

async function copyToken() {
  if (!token.value) return
  await navigator.clipboard.writeText(token.value.value)
  copied.value = true
  window.setTimeout(() => (copied.value = false), 1500)
}
</script>

<template>
  <section class="admin">
    <header>
      <h2>{{ t.admin.title }}</h2>
      <nav class="tabs">
        <button type="button" class="chip" :class="{ on: tab === 'users' }" @click="tab = 'users'">{{ t.admin.users }}</button>
        <button type="button" class="chip" :class="{ on: tab === 'bots' }" @click="tab = 'bots'">{{ t.admin.bots }}</button>
        <button type="button" class="chip" :class="{ on: tab === 'audit' }" @click="(tab = 'audit'), loadAudit()">{{ t.admin.audit }}</button>
      </nav>
    </header>
    <p v-if="error" class="error" role="alert">{{ error }}</p>
    <div v-if="license" class="license" :class="{ full: license.seats > 0 && license.used_seats >= license.seats }">
      <span class="label">{{ t.admin.license }}</span>
      <span>{{ license.status }}</span>
      <span class="muted">· {{ t.admin.seatsUsed }} {{ license.used_seats }}<template v-if="license.seats > 0"> / {{ license.seats }}</template><template v-else> ({{ t.admin.unlimited }})</template></span>
    </div>

    <div v-if="token" class="token">
      <div><strong>{{ token.name }}</strong> · {{ t.admin.tokenOnce }}</div>
      <code>{{ token.value }}</code>
      <div class="row"><button type="button" class="btn sm" @click="copyToken">{{ copied ? t.admin.copied : t.admin.copy }}</button><button type="button" class="btn sm ghost" @click="token = null">✕</button></div>
    </div>

    <template v-if="tab === 'users'">
      <form class="form" @submit.prevent="createUser">
        <div class="label">{{ t.admin.newUser }}</div>
        <div class="grid">
          <input v-model="newUser.username" class="input" :placeholder="t.admin.username" required autocapitalize="off" />
          <input v-model="newUser.display_name" class="input" :placeholder="t.admin.displayName" required />
          <input v-model="newUser.email" class="input" type="email" :placeholder="t.admin.email" />
          <input v-model="newUser.department" class="input" :placeholder="t.admin.department" />
          <input v-model="newUser.title" class="input" :placeholder="t.admin.jobTitle" />
          <select v-model="newUser.role" class="input"><option value="member">{{ t.admin.member }}</option><option value="admin">{{ t.admin.adminRole }}</option></select>
          <input v-model="newUser.password" class="input" type="password" :placeholder="t.admin.password" required minlength="8" autocomplete="new-password" />
          <button type="submit" class="btn primary">{{ t.admin.create }}</button>
        </div>
      </form>
      <table>
        <thead><tr><th></th><th>{{ t.admin.displayName }}</th><th>{{ t.admin.username }}</th><th>{{ t.admin.department }}</th><th>{{ t.admin.role }}</th><th>{{ t.admin.status }}</th><th></th></tr></thead>
        <tbody>
          <tr v-for="u in people" :key="u.id">
            <td><Avatar :id="u.id" :name="u.display_name" :size="30" /></td>
            <td>{{ u.display_name }}</td>
            <td class="muted">{{ u.username }}</td>
            <td class="muted">{{ u.department }}</td>
            <td><span class="badge">{{ u.role === 'admin' ? t.admin.adminRole : t.admin.member }}</span></td>
            <td><span class="badge" :class="{ bot: u.status === 'active' }">{{ u.status === 'active' ? t.admin.active : t.admin.suspended }}</span></td>
            <td>
              <button v-if="u.status === 'active'" type="button" class="btn sm danger" @click="setStatus(u, 'suspended')">{{ t.admin.suspend }}</button>
              <button v-else type="button" class="btn sm" @click="setStatus(u, 'active')">{{ t.admin.activate }}</button>
            </td>
          </tr>
        </tbody>
      </table>
    </template>

    <template v-else-if="tab === 'bots'">
      <form class="form" @submit.prevent="createBot">
        <div class="label">{{ t.admin.newBot }}</div>
        <div class="grid">
          <input v-model="newBot.username" class="input" :placeholder="t.admin.username" required autocapitalize="off" />
          <input v-model="newBot.display_name" class="input" :placeholder="t.admin.displayName" />
          <button type="submit" class="btn primary">{{ t.admin.create }}</button>
        </div>
      </form>
      <table>
        <thead><tr><th></th><th>{{ t.admin.displayName }}</th><th>{{ t.admin.username }}</th><th>{{ t.admin.status }}</th><th></th></tr></thead>
        <tbody>
          <tr v-for="b in bots" :key="b.id">
            <td><Avatar :id="b.id" :name="b.display_name" kind="bot" :size="30" /></td>
            <td>{{ b.display_name }}</td>
            <td class="muted">{{ b.username }}</td>
            <td><span class="badge" :class="{ bot: b.status === 'active' }">{{ b.status === 'active' ? t.admin.active : t.admin.suspended }}</span></td>
            <td class="actions">
              <button type="button" class="btn sm" @click="regenerate(b)">{{ t.admin.regenerate }}</button>
              <button v-if="b.status === 'active'" type="button" class="btn sm danger" @click="setStatus(b, 'suspended')">{{ t.admin.suspend }}</button>
              <button v-else type="button" class="btn sm" @click="setStatus(b, 'active')">{{ t.admin.activate }}</button>
            </td>
          </tr>
        </tbody>
      </table>
    </template>

    <template v-else>
      <table>
        <thead><tr><th>{{ t.admin.when }}</th><th>{{ t.admin.actor }}</th><th>{{ t.admin.action }}</th><th></th></tr></thead>
        <tbody>
          <tr v-for="e in audit" :key="e.id">
            <td class="muted nowrap">{{ fullTime(e.created_at) }}</td>
            <td>{{ users.name(e.actor_id) }}</td>
            <td><code>{{ e.action }}</code> <span class="muted">{{ e.target }}</span></td>
            <td class="muted small">{{ e.meta ? JSON.stringify(e.meta) : '' }}</td>
          </tr>
        </tbody>
      </table>
    </template>
  </section>
</template>

<style scoped>
.admin { flex-grow: 1; overflow: auto; padding: 22px 24px; display: flex; flex-direction: column; gap: 18px; }
header { display: flex; align-items: center; gap: 20px; flex-wrap: wrap; }
h2 { font-size: 24px; font-weight: 800; letter-spacing: -0.03em; }
.tabs { display: flex; gap: 8px; }
.form { display: flex; flex-direction: column; gap: 10px; padding: 16px; border-radius: var(--r-card); background: var(--chip); }
.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 8px; }
table { width: 100%; border-collapse: collapse; font-size: 14px; }
th { text-align: left; font-size: 11px; font-weight: 600; letter-spacing: 0.08em; text-transform: uppercase; color: var(--faint); padding: 8px 10px; border-bottom: 1px solid var(--border); }
td { padding: 8px 10px; border-bottom: 1px solid var(--border); vertical-align: middle; }
.actions { display: flex; gap: 6px; }
.nowrap { white-space: nowrap; }
.small { font-size: 12px; }
.token { background: var(--accent-soft); color: var(--ink); border-radius: var(--r-card); padding: 14px 16px; display: flex; flex-direction: column; gap: 8px; }
.token code { font-size: 13px; word-break: break-all; background: var(--panel-solid); color: var(--ink); padding: 8px 10px; border-radius: 10px; }
.row { display: flex; gap: 8px; }
.license { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; padding: 10px 14px; border-radius: var(--r-ctl); background: var(--chip); font-size: 13px; font-weight: 600; }
.license.full { border-color: var(--danger); background: var(--danger-soft); }
</style>
