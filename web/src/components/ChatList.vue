<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useAuth } from '../stores/auth'
import { useChats } from '../stores/chats'
import { useUsers } from '../stores/users'
import { listTime } from '../format'
import { t } from '../strings'
import Avatar from './Avatar.vue'
import NewChatDialog from './NewChatDialog.vue'

const auth = useAuth()
const chats = useChats()
const users = useUsers()
const query = ref('')
const showNew = ref(false)

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return chats.list
  return chats.list.filter((c) => c.title.toLowerCase().includes(q) || c.last_body.toLowerCase().includes(q))
})
const channels = computed(() => filtered.value.filter((c) => c.type === 'channel'))
const others = computed(() => filtered.value.filter((c) => c.type !== 'channel'))

function preview(c: (typeof chats.list)[number]) {
  if (!c.last_body) return ''
  if (c.last_kind === 'file') return `📎 ${c.last_body}`
  if (c.last_kind === 'system') {
    try {
      const ev = JSON.parse(c.last_body) as { type: string; actor_id: number }
      const label = { added: t.system.added, removed: t.system.removed, left: t.system.left, renamed: t.system.renamed }[ev.type] ?? ev.type
      return `${users.name(ev.actor_id)} ${label}`
    } catch {
      return c.last_body
    }
  }
  const who = c.last_sender_id === auth.user?.id ? t.list.you : c.type === 'direct' ? '' : users.name(c.last_sender_id)
  return who ? `${who}: ${c.last_body}` : c.last_body
}
</script>

<template>
  <aside class="list">
    <header>
      <div class="row">
        <h1>{{ t.list.title }}</h1>
        <button class="btn icon sm" type="button" :aria-label="t.list.newChat" :title="t.list.newChat" @click="showNew = true">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M12 5v14M5 12h14" /></svg>
        </button>
      </div>
      <label class="search">
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="7" /><path d="m20 20-3.5-3.5" /></svg>
        <input v-model="query" type="search" :placeholder="t.list.search" />
      </label>
      <div v-if="!chats.connected" class="offline">{{ t.chat.offline }}</div>
    </header>

    <div class="scroll">
      <template v-if="channels.length">
        <div class="label section">{{ t.nav.channels }}</div>
        <RouterLink v-for="c in channels" :key="c.id" :to="{ name: 'chat', params: { id: c.id } }" class="item" :class="{ on: c.id === chats.activeId }">
          <Avatar :id="c.id" :name="c.title" kind="channel" size="lg" />
          <div class="text">
            <div class="top"><span class="name">{{ c.title }}</span><span class="time">{{ listTime(c.last_at) }}</span></div>
            <div class="bottom"><span class="preview" :class="{ unread: c.unread > 0 }">{{ preview(c) }}</span><span v-if="c.unread" class="pill">{{ c.unread }}</span></div>
          </div>
        </RouterLink>
      </template>

      <div class="label section">{{ t.list.recent }}</div>
      <p v-if="!others.length" class="empty muted">{{ t.list.empty }}</p>
      <RouterLink v-for="c in others" :key="c.id" :to="{ name: 'chat', params: { id: c.id } }" class="item" :class="{ on: c.id === chats.activeId }">
        <Avatar :id="c.type === 'direct' ? (c.peer_id ?? c.id) : c.id" :name="c.title" :kind="c.type === 'direct' && c.peer_id && users.isBot(c.peer_id) ? 'bot' : 'user'" size="lg" />
        <div class="text">
          <div class="top"><span class="name">{{ c.title }}</span><span class="time">{{ listTime(c.last_at) }}</span></div>
          <div class="bottom"><span class="preview" :class="{ unread: c.unread > 0 }">{{ preview(c) }}</span><span v-if="c.unread" class="pill">{{ c.unread }}</span></div>
        </div>
      </RouterLink>
    </div>

    <NewChatDialog v-if="showNew" @close="showNew = false" />
  </aside>
</template>

<style scoped>
.list { width: var(--list-w); flex: 0 0 var(--list-w); display: flex; flex-direction: column; border-right: 1px solid var(--border); background: var(--bg); min-height: 0; }
header { padding: 18px 16px 10px; display: flex; flex-direction: column; gap: 12px; }
.row { display: flex; align-items: center; justify-content: space-between; }
h1 { font-size: 20px; font-weight: 700; letter-spacing: -0.02em; }
.search { display: flex; align-items: center; gap: 8px; height: 36px; padding: 0 10px; border-radius: 8px; background: var(--surface); border: 1px solid var(--border); color: var(--faint); }
.search input { border: 0; outline: 0; background: transparent; flex-grow: 1; min-width: 0; }
.offline { font-size: 12px; color: var(--danger); background: var(--danger-soft); padding: 6px 10px; border-radius: 8px; }
.scroll { flex-grow: 1; overflow: auto; min-height: 0; }
.section { padding: 10px 16px 4px; }
.empty { padding: 8px 16px; }
.item { display: flex; gap: 12px; align-items: center; padding: 10px 16px; border-top: 1px solid transparent; border-bottom: 1px solid transparent; }
.item:hover { background: var(--surface); }
.item.on { background: var(--surface); border-color: var(--border); box-shadow: inset 3px 0 0 var(--accent); }
.text { flex-grow: 1; min-width: 0; }
.top, .bottom { display: flex; justify-content: space-between; gap: 8px; align-items: center; }
.name { font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.time { font-size: 12px; color: var(--faint); flex: 0 0 auto; }
.preview { color: var(--muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.preview.unread { color: var(--ink); font-weight: 500; }
@media (max-width: 767px) { .list { width: 100%; flex: 1 1 auto; border-right: 0; } }
</style>
