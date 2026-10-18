<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useAuth } from '../stores/auth'
import { useChats } from '../stores/chats'
import { useUsers } from '../stores/users'
import { listTime } from '../format'
import { t } from '../strings'
import type { ChatSummary } from '../types'
import Avatar from './Avatar.vue'
import NewChatDialog from './NewChatDialog.vue'

const auth = useAuth()
const chats = useChats()
const users = useUsers()
const query = ref('')
const filter = ref<'all' | 'unread' | 'channel' | 'bot'>('all')
const showNew = ref(false)

const filters = [
  { key: 'all', label: t.list.all },
  { key: 'unread', label: t.list.unread },
  { key: 'channel', label: t.nav.channels },
  { key: 'bot', label: t.admin.bots },
] as const

function isBotChat(c: ChatSummary) {
  return c.type === 'direct' && !!c.peer_id && users.isBot(c.peer_id)
}

const visible = computed(() => {
  const q = query.value.trim().toLowerCase()
  return chats.list.filter((c) => {
    if (q && !c.title.toLowerCase().includes(q) && !c.last_body.toLowerCase().includes(q)) return false
    switch (filter.value) {
      case 'unread': return c.unread > 0
      case 'channel': return c.type === 'channel'
      case 'bot': return isBotChat(c)
      default: return true
    }
  })
})

function preview(c: ChatSummary) {
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

function kindOf(c: ChatSummary): 'user' | 'bot' | 'channel' {
  if (c.type === 'channel') return 'channel'
  return isBotChat(c) ? 'bot' : 'user'
}
</script>

<template>
  <aside class="list panel">
    <header>
      <div class="row">
        <h1>{{ t.list.title }}</h1>
        <button class="btn primary new" type="button" :aria-label="t.list.newChat" :title="t.list.newChat" @click="showNew = true">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><path d="M12 5v14M5 12h14" /></svg>
        </button>
      </div>
      <label class="search">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="7" /><path d="m20 20-3.5-3.5" /></svg>
        <input v-model="query" type="search" :placeholder="t.list.search" />
        <kbd>⌘K</kbd>
      </label>
      <div class="filters">
        <button v-for="f in filters" :key="f.key" type="button" class="chip" :class="{ on: filter === f.key }" @click="filter = f.key">{{ f.label }}</button>
      </div>
      <div v-if="!chats.connected" class="offline">{{ t.chat.offline }}</div>
    </header>

    <div class="scroll">
      <p v-if="!visible.length" class="empty muted">{{ t.list.empty }}</p>
      <RouterLink v-for="c in visible" :key="c.id" :to="{ name: 'chat', params: { id: c.id } }" class="item" :class="{ on: c.id === chats.activeId }">
        <Avatar :id="c.type === 'direct' ? (c.peer_id ?? c.id) : c.id" :name="c.title" :kind="kindOf(c)" :size="44" />
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
.list { width: var(--list-w); flex: 0 0 var(--list-w); display: flex; flex-direction: column; min-height: 0; padding: 18px 10px 10px; }
header { padding: 0 10px 8px; display: flex; flex-direction: column; gap: 14px; }
.row { display: flex; align-items: center; justify-content: space-between; }
h1 { font-size: 24px; font-weight: 800; letter-spacing: -0.03em; }
.new { width: 38px; padding: 0; }
.search { display: flex; align-items: center; gap: 10px; height: 42px; padding: 0 12px 0 14px; border-radius: 14px; background: var(--surface-2); border: 1px solid var(--border); color: var(--faint); }
.search input { border: 0; outline: 0; background: transparent; flex-grow: 1; min-width: 0; font-weight: 500; }
.search input::placeholder { color: var(--faint); }
kbd { font: 700 11px var(--font); padding: 2px 7px; border-radius: 6px; background: var(--chip); color: var(--faint); }
.filters { display: flex; gap: 8px; flex-wrap: wrap; }
.offline { font-size: 12.5px; font-weight: 600; color: var(--danger); background: var(--danger-soft); padding: 8px 12px; border-radius: 12px; }
.scroll { flex-grow: 1; overflow: auto; min-height: 0; display: flex; flex-direction: column; gap: 2px; padding-top: 6px; }
.empty { padding: 12px; text-align: center; }
.item { display: flex; gap: 12px; align-items: center; padding: 10px 12px; border-radius: var(--r-row); }
.item:hover { background: var(--chip); }
.item.on { background: var(--chip-strong); }
.text { flex-grow: 1; min-width: 0; }
.top, .bottom { display: flex; justify-content: space-between; gap: 8px; align-items: baseline; }
.bottom { align-items: center; }
.name { font-weight: 700; font-size: 15px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.time { font-size: 11.5px; font-weight: 600; color: var(--faint); flex: 0 0 auto; }
.preview { font-size: 13.5px; font-weight: 500; color: var(--muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.preview.unread { color: var(--ink); font-weight: 600; }
@media (max-width: 767px) { .list { width: 100%; flex: 1 1 auto; padding-top: max(18px, env(safe-area-inset-top)); } }
</style>
