<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useAuth } from '../stores/auth'
import { useChats } from '../stores/chats'
import { useUsers } from '../stores/users'
import { fileSize, fullTime, timeOf } from '../format'
import { t } from '../strings'
import type { Message, ReadInfo } from '../types'
import Avatar from './Avatar.vue'

const props = defineProps<{
  message: Message
  /** previous row is the same sender within a few minutes */
  continued: boolean
  /** next row is the same sender within a few minutes */
  continues: boolean
  /** show read status under this row (newest own message) */
  showReceipt: boolean
  chatType: 'direct' | 'group' | 'channel'
  peerReadSeq: number
}>()

const auth = useAuth()
const users = useUsers()
const chats = useChats()
const info = ref<ReadInfo | null>(null)

const mine = computed(() => props.message.sender_id === auth.user?.id)
const sender = computed(() => users.get(props.message.sender_id))
const isBot = computed(() => sender.value?.kind === 'bot')
const name = computed(() => users.name(props.message.sender_id))
const att = computed(() => props.message.attachment)
const isImage = computed(() => !!att.value && att.value.mime.startsWith('image/') && att.value.file_id > 0)

interface SystemEvent { type: string; actor_id: number; user_ids?: number[]; title?: string }
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

/** Readers to show as an avatar stack: the peer in 1:1, fetched on demand in groups. */
const readers = computed(() => {
  if (!props.showReceipt || !mine.value) return []
  if (props.chatType === 'direct') {
    const peer = chats.members[props.message.chat_id]?.find((m) => m.user_id !== auth.user?.id)
    return peer && props.peerReadSeq >= props.message.seq ? [peer.user_id] : []
  }
  return (info.value?.readers ?? []).map((r) => r.user_id).filter((id) => id !== auth.user?.id)
})

async function loadReaders() {
  if (props.chatType === 'direct' || !props.showReceipt || !mine.value || props.message.pending) return
  info.value = await chats.readInfo(props.message.chat_id, props.message.seq).catch(() => null)
}
onMounted(loadReaders)
watch(() => [props.showReceipt, props.message.seq], loadReaders)
</script>

<template>
  <div v-if="message.kind === 'system'" class="system"><span>{{ systemText }}</span></div>

  <div v-else class="row" :class="{ mine, continued, continues, pending: message.pending, failed: message.failed }">
    <div v-if="!mine" class="gutter">
      <Avatar v-if="!continues" :id="message.sender_id" :name="sender?.display_name ?? ''" :kind="isBot ? 'bot' : 'user'" :size="34" />
    </div>
    <div class="col">
      <div v-if="!continued" class="head">
        <span v-if="!mine" class="name">{{ name }}</span>
        <span v-if="isBot" class="badge bot">BOT</span>
        <time class="time" :title="fullTime(message.created_at)">{{ timeOf(message.created_at) }}</time>
      </div>

      <a v-if="isImage" :href="`/api/files/${att!.file_id}`" target="_blank" rel="noopener" class="bubble image">
        <img :src="`/api/files/${att!.file_id}`" :alt="att!.name" loading="lazy" />
      </a>
      <a v-else-if="att" :href="att.file_id ? `/api/files/${att.file_id}` : undefined" class="bubble file" :download="att.name">
        <span class="ext">{{ (att.name.split('.').pop() ?? 'FILE').slice(0, 4).toUpperCase() }}</span>
        <span class="fmeta"><span class="fname">{{ att.name }}</span><span class="fsize">{{ fileSize(att.size) }}</span></span>
      </a>
      <div v-else class="bubble"><span class="body">{{ message.body }}</span></div>

      <div v-if="message.failed" class="status error">
        {{ t.chat.failed }} · <button type="button" class="link" @click="chats.retry(message.chat_id, message.client_msg_id!)">{{ t.chat.retry }}</button>
      </div>
      <div v-else-if="message.pending" class="status faint">{{ t.chat.sending }}</div>
      <div v-else-if="readers.length" class="status seen">
        <span class="faint">{{ t.chat.read }}</span>
        <span class="stack">
          <Avatar v-for="id in readers.slice(0, 5)" :key="id" :id="id" :name="users.name(id)" :size="18" />
          <span v-if="readers.length > 5" class="more">+{{ readers.length - 5 }}</span>
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.system { display: flex; justify-content: center; padding: 6px 0; }
.system span { font-size: 12px; font-weight: 600; color: var(--faint); }
.row { display: flex; gap: 10px; align-items: flex-end; }
.row.mine { justify-content: flex-end; }
.row.continued { margin-top: -8px; }
.gutter { width: 34px; flex: 0 0 34px; display: flex; align-items: flex-end; margin-bottom: 2px; }
.col { display: flex; flex-direction: column; gap: 4px; align-items: flex-start; max-width: min(72%, 620px); min-width: 0; }
.mine .col { align-items: flex-end; }
.head { display: flex; align-items: center; gap: 8px; margin: 0 6px 2px; }
.name { font-size: 13px; font-weight: 700; }
.time { font-size: 11.5px; font-weight: 600; color: var(--faint); }
.bubble { background: var(--surface-2); color: var(--ink); padding: 10px 16px; border-radius: 6px 20px 20px 20px; font-size: 15px; max-width: 100%; }
.mine .bubble { background: var(--accent-grad); color: var(--accent-ink); border-radius: 20px 20px 6px 20px; font-weight: 500; }
.continued:not(.mine) .bubble { border-radius: 20px; }
.continues:not(.mine) .bubble { border-radius: 6px 20px 20px 6px; }
.body { white-space: pre-wrap; overflow-wrap: anywhere; }
.pending .bubble { opacity: 0.6; }
.failed .bubble { background: var(--danger-soft); color: var(--danger); }
.bubble.image { padding: 4px; overflow: hidden; }
.bubble.image img { display: block; max-width: 360px; max-height: 320px; border-radius: 16px; object-fit: contain; }
.bubble.file { display: inline-flex; align-items: center; gap: 12px; padding: 10px 14px 10px 10px; }
.ext { width: 36px; height: 36px; border-radius: 10px; background: var(--ink); color: var(--bg); display: inline-flex; align-items: center; justify-content: center; font-size: 10px; font-weight: 800; flex: 0 0 36px; }
.mine .ext { background: rgba(255, 255, 255, 0.9); color: #0f766e; }
.fmeta { display: flex; flex-direction: column; min-width: 0; }
.fname { font-weight: 700; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.fsize { font-size: 12px; opacity: 0.75; }
.status { font-size: 11.5px; font-weight: 600; display: flex; align-items: center; gap: 6px; margin: 0 6px; }
.seen .stack { display: inline-flex; }
.seen .stack .avatar { margin-left: -6px; border: 2px solid var(--panel-solid); box-sizing: content-box; }
.seen .stack .avatar:first-child { margin-left: 0; }
.more { font-size: 10px; font-weight: 800; color: var(--muted); margin-left: 4px; }
.link { border: 0; background: transparent; padding: 0; color: var(--danger); font-weight: 700; font-size: 11.5px; }
@media (max-width: 767px) { .col { max-width: 84%; } }
</style>
