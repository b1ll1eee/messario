<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '../stores/auth'
import { useChats } from '../stores/chats'
import { useUsers } from '../stores/users'
import { t } from '../strings'
import Avatar from './Avatar.vue'

const emit = defineEmits<{ close: [] }>()
const auth = useAuth()
const chats = useChats()
const users = useUsers()
const router = useRouter()

const mode = ref<'person' | 'group' | 'channel'>('person')
const query = ref('')
const title = ref('')
const picked = ref<Set<number>>(new Set())
const busy = ref(false)
const error = ref('')

onMounted(() => {
  if (!users.loaded) void users.load()
})

const candidates = computed(() => {
  const q = query.value.trim().toLowerCase()
  return Object.values(users.byId)
    .filter((u) => u.id !== auth.user?.id && u.status === 'active')
    .filter((u) => !q || u.display_name.toLowerCase().includes(q) || u.username.includes(q))
    .sort((a, b) => a.display_name.localeCompare(b.display_name, 'th'))
})

async function openDirect(id: number) {
  busy.value = true
  try {
    const c = await chats.openDirect(id)
    emit('close')
    router.push({ name: 'chat', params: { id: c.id } })
  } finally {
    busy.value = false
  }
}

function toggle(id: number) {
  const s = new Set(picked.value)
  s.has(id) ? s.delete(id) : s.add(id)
  picked.value = s
}

async function create() {
  error.value = ''
  busy.value = true
  try {
    const c = await chats.createGroup(mode.value === 'channel' ? 'channel' : 'group', title.value, [...picked.value])
    emit('close')
    router.push({ name: 'chat', params: { id: c.id } })
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="dialog-backdrop" @click.self="emit('close')">
    <div class="dialog" role="dialog" aria-modal="true">
      <h2>{{ t.newChat.title }}</h2>
      <div class="tabs">
        <button type="button" class="btn sm" :class="{ primary: mode === 'person' }" @click="mode = 'person'">{{ t.newChat.person }}</button>
        <button type="button" class="btn sm" :class="{ primary: mode === 'group' }" @click="mode = 'group'">{{ t.newChat.group }}</button>
        <button v-if="auth.user?.role === 'admin'" type="button" class="btn sm" :class="{ primary: mode === 'channel' }" @click="mode = 'channel'">{{ t.newChat.channel }}</button>
      </div>
      <input v-if="mode !== 'person'" v-model="title" class="input" :placeholder="mode === 'group' ? t.newChat.groupTitle : t.newChat.channelTitle" />
      <input v-model="query" class="input" type="search" :placeholder="t.newChat.search" />
      <ul class="people">
        <li v-for="u in candidates" :key="u.id">
          <button type="button" class="person" :class="{ on: picked.has(u.id) }" :disabled="busy" @click="mode === 'person' ? openDirect(u.id) : toggle(u.id)">
            <Avatar :id="u.id" :name="u.display_name" :kind="u.kind" size="sm" />
            <span class="pname">{{ u.display_name }}</span>
            <span class="muted small">{{ u.department || u.username }}</span>
            <span v-if="u.kind === 'bot'" class="badge bot">BOT</span>
            <span v-if="mode !== 'person' && picked.has(u.id)" class="check" aria-hidden="true">✓</span>
          </button>
        </li>
      </ul>
      <p v-if="error" class="error">{{ error }}</p>
      <div class="actions">
        <button type="button" class="btn" @click="emit('close')">{{ t.newChat.cancel }}</button>
        <button v-if="mode !== 'person'" type="button" class="btn primary" :disabled="busy || !title.trim()" @click="create">{{ t.newChat.create }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tabs { display: flex; gap: 8px; flex-wrap: wrap; }
.people { max-height: 320px; overflow: auto; border: 1px solid var(--border); border-radius: var(--radius); }
.person { width: 100%; display: flex; align-items: center; gap: 10px; padding: 8px 10px; border: 0; background: transparent; text-align: left; min-height: 44px; }
.person:hover { background: var(--surface-2); }
.person.on { background: var(--accent-soft); }
.pname { font-weight: 500; }
.small { font-size: 12px; flex-grow: 1; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.check { color: var(--accent); font-weight: 700; }
.actions { display: flex; justify-content: flex-end; gap: 8px; }
</style>
