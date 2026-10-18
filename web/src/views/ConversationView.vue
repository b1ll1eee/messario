<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { useAuth } from '../stores/auth'
import { useChats } from '../stores/chats'
import { useUsers } from '../stores/users'
import { dayLabel, isSameDay } from '../format'
import { t } from '../strings'
import type { Message } from '../types'
import Avatar from '../components/Avatar.vue'
import Composer from '../components/Composer.vue'
import MessageRow from '../components/MessageRow.vue'
import RightPanel from '../components/RightPanel.vue'

const props = defineProps<{ id: number }>()
const auth = useAuth()
const chats = useChats()
const users = useUsers()

const scroller = ref<HTMLElement | null>(null)
const showPanel = ref(false)
const loadingOlder = ref(false)
const stickToBottom = ref(true)
const uploadError = ref('')

const GROUP_WINDOW_MS = 5 * 60 * 1000

const chat = computed(() => chats.active)
const isChannel = computed(() => chat.value?.type === 'channel')
const canPost = computed(() => {
  if (!chat.value) return false
  if (!isChannel.value) return true
  const role = chats.myMembership?.role
  return role === 'owner' || role === 'admin'
})
const memberCount = computed(() => chat.value?.member_count ?? 0)
const members = computed(() => (chat.value ? (chats.members[chat.value.id] ?? []) : []))
const botCount = computed(() => members.value.filter((m) => users.isBot(m.user_id)).length)
const peer = computed(() => (chat.value?.type === 'direct' && chat.value.peer_id ? users.get(chat.value.peer_id) : undefined))
const headAvatars = computed(() => members.value.filter((m) => !users.isBot(m.user_id)).slice(0, 3).map((m) => m.user_id))

/** Rows with day dividers and grouping flags, computed once per change. */
const rows = computed(() => {
  const out: Array<{ key: string; day?: string; msg?: Message; continued: boolean; continues: boolean; receipt: boolean }> = []
  const msgs = chats.activeMessages
  const lastMineSeq = [...msgs].reverse().find((m) => m.sender_id === auth.user?.id && !m.pending && !m.failed)?.seq
  const sameGroup = (a: Message | undefined, b: Message | undefined) =>
    !!a && !!b && a.kind !== 'system' && b.kind !== 'system' && a.sender_id === b.sender_id &&
    Math.abs(new Date(b.created_at).getTime() - new Date(a.created_at).getTime()) < GROUP_WINDOW_MS &&
    isSameDay(new Date(a.created_at), new Date(b.created_at))
  for (let i = 0; i < msgs.length; i++) {
    const m = msgs[i]!
    const prev = msgs[i - 1]
    const next = msgs[i + 1]
    if (!prev || !isSameDay(new Date(prev.created_at), new Date(m.created_at))) {
      out.push({ key: 'day-' + m.created_at.slice(0, 10) + '-' + m.seq, day: dayLabel(m.created_at), continued: false, continues: false, receipt: false })
    }
    out.push({ key: m.client_msg_id || 's' + m.seq, msg: m, continued: sameGroup(prev, m), continues: sameGroup(m, next), receipt: m.seq === lastMineSeq })
  }
  return out
})

watch(
  () => props.id,
  async (id) => {
    stickToBottom.value = true
    uploadError.value = ''
    await chats.open(id)
    await nextTick()
    scrollToBottom()
  },
  { immediate: true },
)

watch(
  () => chats.activeMessages.length,
  async () => {
    if (!stickToBottom.value) return
    await nextTick()
    scrollToBottom()
  },
)

function scrollToBottom() {
  const el = scroller.value
  if (el) el.scrollTop = el.scrollHeight
}

async function onScroll() {
  const el = scroller.value
  if (!el) return
  stickToBottom.value = el.scrollHeight - el.scrollTop - el.clientHeight < 40
  if (el.scrollTop < 80 && !loadingOlder.value && chats.hasOlder[props.id] !== false) {
    loadingOlder.value = true
    const before = el.scrollHeight
    try {
      await chats.loadOlder(props.id)
      await nextTick()
      el.scrollTop += el.scrollHeight - before
    } finally {
      loadingOlder.value = false
    }
  }
}

function send(body: string) {
  chats.send(props.id, body)
  stickToBottom.value = true
}

async function attach(file: File) {
  uploadError.value = ''
  stickToBottom.value = true
  try {
    await chats.sendFile(props.id, file)
  } catch (e) {
    uploadError.value = `${t.chat.uploadFailed}: ${(e as Error).message}`
  }
}
</script>

<template>
  <section v-if="chat" class="conv">
    <header class="head">
      <RouterLink :to="{ name: 'home' }" class="back btn icon ghost" aria-label="กลับ">
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6" /></svg>
      </RouterLink>
      <div class="faces">
        <Avatar v-if="chat.type === 'direct'" :id="chat.peer_id ?? chat.id" :name="chat.title" :kind="peer?.kind === 'bot' ? 'bot' : 'user'" :size="40" :presence="peer?.kind !== 'bot'" />
        <Avatar v-else-if="isChannel" :id="chat.id" :name="chat.title" kind="channel" :size="40" />
        <template v-else>
          <Avatar v-for="id in headAvatars" :key="id" :id="id" :name="users.name(id)" :size="36" />
          <span v-if="memberCount > headAvatars.length" class="avatar more">+{{ memberCount - headAvatars.length }}</span>
        </template>
      </div>
      <div class="title">
        <h2>{{ chat.title }}</h2>
        <div class="sub">
          <template v-if="chat.type === 'direct'">{{ peer?.kind === 'bot' ? 'BOT' : peer?.department || peer?.username }}</template>
          <template v-else><span class="dot">●</span> {{ memberCount }} {{ t.chat.members }}<template v-if="botCount"> · {{ botCount }} {{ t.chat.bots }}</template><template v-if="isChannel"> · {{ t.chat.readOnly }}</template></template>
        </div>
      </div>
      <button type="button" class="btn icon" :class="{ on: showPanel }" :aria-label="t.chat.info" :title="t.chat.info" @click="showPanel = !showPanel">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="9" /><path d="M12 11v5M12 8h.01" /></svg>
      </button>
    </header>

    <div class="body">
      <div ref="scroller" class="scroll" :class="{ channel: isChannel }" @scroll.passive="onScroll">
        <div v-if="loadingOlder" class="faint center">…</div>
        <template v-for="r in rows" :key="r.key">
          <div v-if="r.day" class="day"><span>{{ r.day }}</span></div>
          <MessageRow v-else-if="r.msg" :message="r.msg" :continued="r.continued" :continues="r.continues" :show-receipt="r.receipt" :chat-type="chat.type" :peer-read-seq="chats.peerReadSeq" />
        </template>
      </div>
      <RightPanel v-if="showPanel" @close="showPanel = false" />
    </div>

    <p v-if="uploadError" class="error upload-error" role="alert">{{ uploadError }}</p>
    <Composer :placeholder="isChannel ? `${t.chat.channelPlaceholder} ${memberCount} ${t.chat.members}` : `${t.chat.placeholder} ${chat.title}`" :disabled="!canPost" :channel="isChannel" @send="send" @attach="attach" />
  </section>
</template>

<style scoped>
.conv { flex-grow: 1; display: flex; flex-direction: column; min-width: 0; min-height: 0; }
.head { display: flex; align-items: center; gap: 14px; padding: 12px 16px 12px 20px; border-bottom: 1px solid var(--border); }
.back { display: none; }
.faces { display: flex; align-items: center; }
.faces .avatar + .avatar, .faces .avatar + .more { margin-left: -12px; }
.faces .avatar { border: 2px solid var(--panel-solid); box-sizing: content-box; }
.more { width: 36px; height: 36px; font-size: 12px; background: var(--chip-strong); color: var(--ink); }
.title { flex-grow: 1; min-width: 0; }
h2 { font-size: 17px; font-weight: 800; letter-spacing: -0.02em; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.sub { font-size: 12.5px; font-weight: 500; color: var(--muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.dot { color: var(--presence); font-size: 9px; vertical-align: 1px; }
.btn.on { background: var(--chip-strong); color: var(--ink); }
.body { flex-grow: 1; display: flex; min-height: 0; position: relative; }
.scroll { flex-grow: 1; overflow-y: auto; padding: 20px 24px 8px; display: flex; flex-direction: column; gap: 14px; min-width: 0; }
.scroll.channel { padding-left: max(24px, calc(50% - 360px)); padding-right: max(24px, calc(50% - 360px)); gap: 18px; }
.scroll.channel :deep(.row) { justify-content: flex-start; }
.scroll.channel :deep(.col) { max-width: 100%; width: 100%; align-items: stretch; }
.scroll.channel :deep(.bubble) { border-radius: var(--r-card); padding: 16px 20px; font-size: 15.5px; line-height: 1.65; background: var(--surface-2); color: var(--ink); font-weight: 400; }
.day { display: flex; justify-content: center; margin: 4px 0; }
.day span { font-size: 12px; font-weight: 700; color: var(--muted); padding: 5px 12px; border-radius: 999px; background: var(--chip); }
.center { text-align: center; }
.upload-error { padding: 0 24px; }
@media (max-width: 767px) {
  .head { padding: max(10px, env(safe-area-inset-top)) 10px 10px 6px; }
  .back { display: inline-flex; }
  .scroll { padding: 14px 12px 8px; }
}
</style>
