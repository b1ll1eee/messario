import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { get, post, del, patch } from '../api/http'
import { socket, type ServerFrame } from '../api/socket'
import type { Chat, ChatSummary, ChatType, Member, Message, ReadInfo } from '../types'
import { useAuth } from './auth'

const PAGE = 50
const READ_DEBOUNCE_MS = 600

interface Outgoing {
  chatId: number
  clientMsgId: string
  body: string
}

export const useChats = defineStore('chats', () => {
  const auth = useAuth()

  const list = ref<ChatSummary[]>([])
  const messages = ref<Record<number, Message[]>>({})
  const members = ref<Record<number, Member[]>>({})
  const hasOlder = ref<Record<number, boolean>>({})
  const activeId = ref<number | null>(null)
  const connected = ref(false)
  const outbox = ref<Outgoing[]>([])
  const readTimers: Record<number, number> = {}

  const active = computed(() => list.value.find((c) => c.id === activeId.value) ?? null)
  const activeMessages = computed(() => (activeId.value ? (messages.value[activeId.value] ?? []) : []))
  const myMembership = computed(() =>
    activeId.value && auth.user ? members.value[activeId.value]?.find((m) => m.user_id === auth.user!.id) : undefined,
  )

  // ---- loading ----

  async function loadList() {
    list.value = await get<ChatSummary[]>('/api/chats')
  }

  async function open(chatId: number) {
    activeId.value = chatId
    if (!messages.value[chatId]) {
      const page = await get<Message[]>(`/api/chats/${chatId}/messages?limit=${PAGE}`)
      messages.value[chatId] = page
      hasOlder.value[chatId] = page.length === PAGE
    } else {
      await syncChat(chatId)
    }
    if (!members.value[chatId]) await loadMembers(chatId)
    markRead(chatId)
  }

  async function loadMembers(chatId: number) {
    members.value[chatId] = await get<Member[]>(`/api/chats/${chatId}/members`)
  }

  async function loadOlder(chatId: number) {
    const msgs = messages.value[chatId]
    if (!msgs?.length || hasOlder.value[chatId] === false) return
    const first = msgs.find((m) => !m.pending)
    if (!first) return
    const page = await get<Message[]>(`/api/chats/${chatId}/messages?before_seq=${first.seq}&limit=${PAGE}`)
    messages.value[chatId] = [...page, ...msgs]
    hasOlder.value[chatId] = page.length === PAGE
  }

  /** Fill any gap since the newest message we hold (after reconnect). */
  async function syncChat(chatId: number) {
    const msgs = messages.value[chatId]
    if (!msgs) return
    const last = [...msgs].reverse().find((m) => !m.pending)
    if (!last) return
    let after = last.seq
    for (;;) {
      const page = await get<Message[]>(`/api/chats/${chatId}/messages?after_seq=${after}&limit=${PAGE}`)
      for (const m of page) insertMessage(m)
      if (page.length < PAGE) break
      after = page[page.length - 1]!.seq
    }
  }

  async function resyncAll() {
    await loadList()
    await Promise.all(Object.keys(messages.value).map((id) => syncChat(Number(id))))
    flushOutbox()
  }

  // ---- sending ----

  function send(chatId: number, body: string) {
    const clientMsgId = crypto.randomUUID()
    const local: Message = {
      id: 0,
      chat_id: chatId,
      seq: Number.MAX_SAFE_INTEGER,
      sender_id: auth.user!.id,
      client_msg_id: clientMsgId,
      kind: 'text',
      body,
      created_at: new Date().toISOString(),
      pending: true,
    }
    ;(messages.value[chatId] ??= []).push(local)
    outbox.value.push({ chatId, clientMsgId, body })
    flushOutbox()
  }

  async function flushOutbox() {
    if (!socket.connected) return
    for (const o of [...outbox.value]) {
      try {
        const ack = await socket.request({ t: 'send', chat_id: o.chatId, client_msg_id: o.clientMsgId, body: o.body })
        outbox.value = outbox.value.filter((x) => x !== o)
        insertMessage(ack.p as Message)
      } catch (e) {
        const frame = (e as { frame?: ServerFrame }).frame
        if (frame?.t === 'err') {
          // The server rejected it for good (forbidden, invalid); stop retrying.
          outbox.value = outbox.value.filter((x) => x !== o)
          const m = messages.value[o.chatId]?.find((x) => x.client_msg_id === o.clientMsgId)
          if (m) {
            m.pending = false
            m.failed = true
          }
        }
        // Otherwise (offline/timeout) keep it queued; the next flush retries.
        return
      }
    }
  }

  function retry(chatId: number, clientMsgId: string) {
    const m = messages.value[chatId]?.find((x) => x.client_msg_id === clientMsgId)
    if (!m) return
    m.failed = false
    m.pending = true
    outbox.value.push({ chatId, clientMsgId, body: m.body })
    flushOutbox()
  }

  /** Insert or replace by (chat, seq) / client_msg_id, keeping order by seq. */
  function insertMessage(m: Message) {
    const msgs = (messages.value[m.chat_id] ??= [])
    const byClient = m.client_msg_id ? msgs.findIndex((x) => x.client_msg_id === m.client_msg_id) : -1
    if (byClient >= 0) {
      msgs.splice(byClient, 1)
    } else if (msgs.some((x) => !x.pending && x.seq === m.seq)) {
      return
    }
    let i = msgs.length
    while (i > 0 && (msgs[i - 1]!.pending || msgs[i - 1]!.seq > m.seq)) i--
    msgs.splice(i, 0, m)
    bumpSummary(m)
  }

  function bumpSummary(m: Message) {
    const s = list.value.find((c) => c.id === m.chat_id)
    if (!s) {
      void loadList()
      return
    }
    if (m.seq > s.last_seq) {
      s.last_seq = m.seq
      s.last_body = m.body
      s.last_kind = m.kind
      s.last_sender_id = m.sender_id
      s.last_at = m.created_at
      const mine = m.sender_id === auth.user?.id
      if (mine) s.read_seq = Math.max(s.read_seq, m.seq)
      s.unread = Math.max(0, s.last_seq - s.read_seq)
      list.value.sort((a, b) => (a.last_at < b.last_at ? 1 : -1))
    }
    if (m.chat_id === activeId.value && document.hasFocus()) markRead(m.chat_id)
  }

  // ---- read receipts ----

  function markRead(chatId: number) {
    const s = list.value.find((c) => c.id === chatId)
    if (!s || s.read_seq >= s.last_seq) return
    window.clearTimeout(readTimers[chatId])
    readTimers[chatId] = window.setTimeout(() => {
      const seq = s.last_seq
      s.read_seq = seq
      s.unread = 0
      socket.send({ t: 'read', chat_id: chatId, seq })
      const me = members.value[chatId]?.find((m) => m.user_id === auth.user?.id)
      if (me) me.read_seq = seq
    }, READ_DEBOUNCE_MS)
  }

  function readInfo(chatId: number, seq: number) {
    return get<ReadInfo>(`/api/chats/${chatId}/read?seq=${seq}`)
  }

  /** For direct chats: the peer's read pointer, from members (updated live). */
  const peerReadSeq = computed(() => {
    if (!active.value || active.value.type !== 'direct') return 0
    return members.value[active.value.id]?.find((m) => m.user_id !== auth.user?.id)?.read_seq ?? 0
  })

  // ---- chat management ----

  async function openDirect(userId: number) {
    const c = await post<Chat>('/api/chats/direct', { user_id: userId })
    await loadList()
    return c
  }

  async function createGroup(type: ChatType, title: string, memberIds: number[]) {
    const c = await post<Chat>('/api/chats', { type, title, member_ids: memberIds })
    await loadList()
    return c
  }

  async function addMembers(chatId: number, userIds: number[]) {
    await post(`/api/chats/${chatId}/members`, { user_ids: userIds })
    await loadMembers(chatId)
    await loadList()
  }

  async function removeMember(chatId: number, userId: number) {
    await del(`/api/chats/${chatId}/members/${userId}`)
    if (userId === auth.user?.id) {
      delete messages.value[chatId]
      delete members.value[chatId]
      if (activeId.value === chatId) activeId.value = null
      await loadList()
    } else {
      await loadMembers(chatId)
    }
  }

  async function rename(chatId: number, title: string) {
    await patch(`/api/chats/${chatId}`, { title })
    await loadList()
  }

  // ---- live events ----

  function applyFrame(f: ServerFrame) {
    switch (f.t) {
      case 'hello':
        void resyncAll()
        break
      case 'msg':
        insertMessage(f.p as Message)
        break
      case 'read': {
        const p = f.p as { chat_id: number; user_id: number; seq: number }
        const m = members.value[p.chat_id]?.find((x) => x.user_id === p.user_id)
        if (m) m.read_seq = Math.max(m.read_seq, p.seq)
        if (p.user_id === auth.user?.id) {
          const s = list.value.find((c) => c.id === p.chat_id)
          if (s) {
            s.read_seq = Math.max(s.read_seq, p.seq)
            s.unread = Math.max(0, s.last_seq - s.read_seq)
          }
        }
        break
      }
      case 'chat':
        void loadList()
        break
      case 'left': {
        const p = f.p as { chat_id: number }
        delete messages.value[p.chat_id]
        delete members.value[p.chat_id]
        if (activeId.value === p.chat_id) activeId.value = null
        void loadList()
        break
      }
    }
  }

  function start() {
    socket.onStatus = (ok) => {
      connected.value = ok
    }
    socket.subscribe(applyFrame)
    socket.connect()
    window.addEventListener('focus', () => {
      if (activeId.value) markRead(activeId.value)
    })
  }

  function stop() {
    socket.close()
    list.value = []
    messages.value = {}
    members.value = {}
    activeId.value = null
    outbox.value = []
  }

  return {
    list, messages, members, hasOlder, activeId, active, activeMessages, myMembership, connected, peerReadSeq,
    loadList, open, loadOlder, loadMembers, send, retry, markRead, readInfo,
    openDirect, createGroup, addMembers, removeMember, rename, start, stop,
  }
})
