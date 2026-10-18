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
const showPanel = ref(window.innerWidth >= 1200)
const loadingOlder = ref(false)
const stickToBottom = ref(true)

const GROUP_WINDOW_MS = 5 * 60 * 1000

const chat = computed(() => chats.active)
const canPost = computed(() => {
  if (!chat.value) return false
  if (chat.value.type !== 'channel') return true
  const role = chats.myMembership?.role
  return role === 'owner' || role === 'admin'
})
const memberCount = computed(() => chat.value?.member_count ?? 0)
const botCount = computed(() => (chat.value ? (chats.members[chat.value.id] ?? []).filter((m) => users.isBot(m.user_id)).length : 0))
const peerIsBot = computed(() => chat.value?.type === 'direct' && !!chat.value.peer_id && users.isBot(chat.value.peer_id))

/** Rows with day dividers and "continued" grouping computed once per change. */
const rows = computed(() => {
  const out: Array<{ key: string; day?: string; msg?: Message; continued: boolean; receipt: boolean }> = []
  const msgs = chats.activeMessages
  const lastMineSeq = [...msgs].reverse().find((m) => m.sender_id === auth.user?.id && !m.pending && !m.failed)?.seq
  let prev: Message | undefined
  for (const m of msgs) {
    const d = new Date(m.created_at)
    if (!prev || !isSameDay(new Date(prev.created_at), d)) out.push({ key: 'day-' + m.created_at.slice(0, 10) + '-' + m.seq, day: dayLabel(m.created_at), continued: false, receipt: false })
    const continued =
      !!prev && prev.kind !== 'system' && m.kind !== 'system' && prev.sender_id === m.sender_id && d.getTime() - new Date(prev.created_at).getTime() < GROUP_WINDOW_MS
    out.push({ key: m.client_msg_id || 's' + m.seq, msg: m, continued, receipt: m.seq === lastMineSeq })
    prev = m
  }
  return out
})

watch(
  () => props.id,
  async (id) => {
    stickToBottom.value = true
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
</script>

<template>
  <section v-if="chat" class="conv">
    <header class="head">
      <RouterLink :to="{ name: 'home' }" class="back btn icon ghost" aria-label="กลับ">
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6" /></svg>
      </RouterLink>
      <Avatar :id="chat.type === 'direct' ? (chat.peer_id ?? chat.id) : chat.id" :name="chat.title" :kind="chat.type === 'channel' ? 'channel' : peerIsBot ? 'bot' : 'user'" />
      <div class="title">
        <h2>{{ chat.title }}</h2>
        <div class="sub muted">
          <template v-if="chat.type === 'direct'">{{ peerIsBot ? 'BOT' : users.get(chat.peer_id ?? 0)?.department || users.get(chat.peer_id ?? 0)?.username }}</template>
          <template v-else>{{ memberCount }} {{ t.chat.members }}<template v-if="botCount"> · {{ botCount }} {{ t.chat.bots }}</template><template v-if="chat.type === 'channel'"> · {{ t.chat.readOnly }}</template></template>
        </div>
      </div>
      <button type="button" class="btn icon" :class="{ on: showPanel }" :aria-label="t.chat.info" :title="t.chat.info" @click="showPanel = !showPanel">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="9" /><path d="M12 11v5M12 8h.01" /></svg>
      </button>
    </header>

    <div class="body">
      <div ref="scroller" class="scroll" :class="{ channel: chat.type === 'channel' }" @scroll.passive="onScroll">
        <div v-if="loadingOlder" class="faint center">…</div>
        <template v-for="r in rows" :key="r.key">
          <div v-if="r.day" class="day"><span>{{ r.day }}</span></div>
          <MessageRow v-else-if="r.msg" :message="r.msg" :continued="r.continued" :show-receipt="r.receipt" :chat-type="chat.type" :peer-read-seq="chats.peerReadSeq" :member-count="memberCount" />
        </template>
      </div>
      <RightPanel v-if="showPanel" @close="showPanel = false" />
    </div>

    <Composer :placeholder="chat.type === 'channel' ? `${t.chat.channelPlaceholder} ${memberCount} ${t.chat.members}` : `${t.chat.placeholder} ${chat.title}`" :disabled="!canPost" :channel="chat.type === 'channel'" @send="send" />
  </section>
</template>

<style scoped>
.conv { flex-grow: 1; display: flex; flex-direction: column; min-width: 0; min-height: 0; }
.head { display: flex; align-items: center; gap: 12px; padding: 12px 20px; border-bottom: 1px solid var(--border); }
.back { display: none; }
.title { flex-grow: 1; min-width: 0; }
h2 { font-size: 16px; font-weight: 700; letter-spacing: -0.01em; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.sub { font-size: 12px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.btn.on { background: var(--surface-2); color: var(--ink); }
.body { flex-grow: 1; display: flex; min-height: 0; position: relative; }
.scroll { flex-grow: 1; overflow-y: auto; padding: 16px 20px 8px; display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.scroll.channel { padding-left: max(20px, calc(50% - 360px)); padding-right: max(20px, calc(50% - 360px)); gap: 14px; }
.scroll.channel :deep(.row) { border: 1px solid var(--border); border-radius: var(--radius-lg); padding: 16px 18px; background: var(--surface); box-shadow: var(--shadow); }
.scroll.channel :deep(.row:hover) { background: var(--surface); }
.scroll.channel :deep(.row .body) { font-size: 15px; line-height: 1.65; margin-top: 4px; }
.scroll.channel :deep(.row.continued) { padding-top: 16px; }
.day { display: flex; align-items: center; gap: 12px; color: var(--faint); font-size: 12px; margin: 8px 0; }
.day::before, .day::after { content: ''; flex-grow: 1; height: 1px; background: var(--border); }
.center { text-align: center; }
@media (max-width: 767px) {
  .head { padding: 10px 12px; }
  .back { display: inline-flex; }
  .scroll { padding: 12px 12px 8px; }
}
</style>
