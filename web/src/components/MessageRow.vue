<script setup lang="ts">
import { computed, ref } from 'vue'
import { useAuth } from '../stores/auth'
import { useChats } from '../stores/chats'
import { useUsers } from '../stores/users'
import { fileSize, fullTime, timeOf } from '../format'
import { t } from '../strings'
import type { Message, ReadInfo } from '../types'
import Avatar from './Avatar.vue'

const props = defineProps<{
  message: Message
  /** hide avatar/name because the previous row is the same sender within a few minutes */
  continued: boolean
  /** show the read status under this row (newest own message) */
  showReceipt: boolean
  chatType: 'direct' | 'group' | 'channel'
  peerReadSeq: number
  memberCount: number
}>()

const auth = useAuth()
const users = useUsers()
const chats = useChats()
const info = ref<ReadInfo | null>(null)

const mine = computed(() => props.message.sender_id === auth.user?.id)
const sender = computed(() => users.get(props.message.sender_id))
const isBot = computed(() => sender.value?.kind === 'bot')
const name = computed(() => (mine.value ? `${sender.value?.display_name ?? ''} (${t.list.you})` : users.name(props.message.sender_id)))

interface SystemEvent { type: string; actor_id: number; user_ids?: number[]; title?: string }

/** System messages carry a JSON event; render it in the viewer's language. */
const systemText = computed(() => {
  if (props.message.kind !== 'system') return ''
  let ev: SystemEvent
  try {
    ev = JSON.parse(props.message.body)
  } catch {
    return props.message.body
  }
  const actor = users.name(ev.actor_id)
  const names = (ev.user_ids ?? []).map((id) => users.name(id)).join(', ')
  switch (ev.type) {
    case 'added': return `${actor} ${t.system.added} ${names}`
    case 'removed': return `${actor} ${t.system.removed} ${names}`
    case 'left': return `${actor} ${t.system.left}`
    case 'renamed': return `${actor} ${t.system.renamed} "${ev.title ?? ''}"`
    default: return props.message.body
  }
})

async function toggleReaders() {
  if (info.value) {
    info.value = null
    return
  }
  info.value = await chats.readInfo(props.message.chat_id, props.message.seq)
}
</script>

<template>
  <div v-if="message.kind === 'system'" class="system"><span>{{ systemText }}</span></div>

  <article v-else class="row" :class="{ continued, pending: message.pending, failed: message.failed }">
    <div class="gutter">
      <Avatar v-if="!continued" :id="message.sender_id" :name="sender?.display_name ?? ''" :kind="isBot ? 'bot' : 'user'" size="sm" />
      <span v-else class="hover-time">{{ timeOf(message.created_at) }}</span>
    </div>
    <div class="content">
      <div v-if="!continued" class="head">
        <span class="name">{{ name }}</span>
        <span v-if="isBot" class="badge bot">BOT</span>
        <time class="time" :title="fullTime(message.created_at)">{{ timeOf(message.created_at) }}</time>
      </div>
      <template v-if="message.kind === 'file' && message.attachment">
        <a v-if="message.attachment.mime.startsWith('image/') && message.attachment.file_id" :href="`/api/files/${message.attachment.file_id}`" target="_blank" rel="noopener" class="image">
          <img :src="`/api/files/${message.attachment.file_id}`" :alt="message.attachment.name" loading="lazy" />
        </a>
        <a v-else :href="message.attachment.file_id ? `/api/files/${message.attachment.file_id}` : undefined" class="file" :download="message.attachment.name">
          <span class="ext">{{ (message.attachment.name.split('.').pop() ?? 'FILE').slice(0, 4).toUpperCase() }}</span>
          <span class="fmeta"><span class="fname">{{ message.attachment.name }}</span><span class="fsize faint">{{ fileSize(message.attachment.size) }}</span></span>
        </a>
      </template>
      <p v-else class="body">{{ message.body }}</p>
      <div v-if="message.failed" class="status error">
        {{ t.chat.failed }} · <button type="button" class="link" @click="chats.retry(message.chat_id, message.client_msg_id!)">{{ t.chat.retry }}</button>
      </div>
      <div v-else-if="message.pending" class="status faint">{{ t.chat.sending }}</div>
      <div v-else-if="showReceipt && mine" class="status faint">
        <template v-if="chatType === 'direct'">
          <span v-if="peerReadSeq >= message.seq" class="read">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5" /></svg>
            {{ t.chat.read }}
          </span>
        </template>
        <template v-else>
          <button type="button" class="link" @click="toggleReaders">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5" /></svg>
            {{ t.chat.readBy }}<template v-if="info"> {{ info.count - 1 }} {{ t.chat.seenOf }} {{ info.total - 1 }}</template>
          </button>
          <span v-if="info" class="readers">{{ info.readers.filter((r) => r.user_id !== auth.user?.id).map((r) => users.name(r.user_id)).join(', ') }}</span>
        </template>
      </div>
    </div>
  </article>
</template>

<style scoped>
.system { display: flex; justify-content: center; padding: 4px 0; }
.system span { font-size: 12px; color: var(--muted); background: var(--surface-2); padding: 3px 10px; border-radius: 999px; }
.row { display: flex; gap: 12px; padding: 2px 0; }
.row.continued { padding-top: 0; }
.row:hover { background: color-mix(in srgb, var(--surface-2) 50%, transparent); }
.row:hover .hover-time { visibility: visible; }
.gutter { width: 28px; flex: 0 0 28px; display: flex; justify-content: center; align-items: flex-start; }
.hover-time { visibility: hidden; font-size: 11px; color: var(--faint); padding-top: 3px; }
.content { flex-grow: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.head { display: flex; align-items: center; gap: 8px; }
.name { font-weight: 600; }
.time { font-size: 12px; color: var(--faint); }
.body { white-space: pre-wrap; overflow-wrap: anywhere; }
.pending .body { opacity: 0.6; }
.failed .body { color: var(--danger); }
.status { font-size: 12px; display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.read { display: inline-flex; align-items: center; gap: 4px; }
.link { border: 0; background: transparent; padding: 0; color: var(--muted); display: inline-flex; align-items: center; gap: 4px; font-size: 12px; }
.link:hover { color: var(--ink); }
.readers { color: var(--faint); }
.image { display: inline-block; max-width: 360px; border-radius: 10px; overflow: hidden; border: 1px solid var(--border); margin-top: 2px; }
.image img { display: block; max-width: 100%; max-height: 320px; object-fit: contain; background: var(--surface-2); }
.file { display: inline-flex; align-items: center; gap: 10px; padding: 10px 12px; border: 1px solid var(--border); border-radius: 10px; background: var(--bg); max-width: 360px; margin-top: 2px; }
.file:hover { border-color: var(--border-strong); }
.ext { width: 34px; height: 34px; border-radius: 8px; background: var(--ink); color: var(--bg); display: inline-flex; align-items: center; justify-content: center; font-size: 10px; font-weight: 700; flex: 0 0 34px; }
.fmeta { display: flex; flex-direction: column; min-width: 0; }
.fname { font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.fsize { font-size: 12px; }
</style>
