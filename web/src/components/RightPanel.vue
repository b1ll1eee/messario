<script setup lang="ts">
import { computed, ref } from 'vue'
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

const chat = computed(() => chats.active)
const members = computed(() => (chat.value ? (chats.members[chat.value.id] ?? []) : []))
const people = computed(() => members.value.filter((m) => !users.isBot(m.user_id)))
const bots = computed(() => members.value.filter((m) => users.isBot(m.user_id)))
const canManage = computed(() => chats.myMembership?.role === 'owner' || chats.myMembership?.role === 'admin')

const adding = ref(false)
const query = ref('')
const candidates = computed(() => {
  const ids = new Set(members.value.map((m) => m.user_id))
  const q = query.value.trim().toLowerCase()
  return Object.values(users.byId)
    .filter((u) => !ids.has(u.id) && u.status === 'active')
    .filter((u) => !q || u.display_name.toLowerCase().includes(q))
    .slice(0, 50)
})

async function add(id: number) {
  if (chat.value) await chats.addMembers(chat.value.id, [id])
}
async function remove(id: number) {
  if (chat.value) await chats.removeMember(chat.value.id, id)
}
async function leave() {
  if (!chat.value || !auth.user) return
  await chats.removeMember(chat.value.id, auth.user.id)
  router.push({ name: 'home' })
}
async function rename() {
  if (!chat.value) return
  const title = window.prompt(t.panel.rename, chat.value.title)
  if (title && title.trim() && title !== chat.value.title) await chats.rename(chat.value.id, title.trim())
}
function roleLabel(r: string) {
  return r === 'owner' ? t.panel.owner : r === 'admin' ? t.panel.admin : ''
}
</script>

<template>
  <aside v-if="chat" class="side">
    <div class="head">
      <span class="label">{{ t.chat.info }}</span>
      <button type="button" class="btn icon ghost sm" aria-label="ปิด" @click="emit('close')">✕</button>
    </div>

    <section class="card">
      <div class="label">{{ t.panel.about }}</div>
      <p v-if="chat.type === 'direct'" class="muted">{{ t.panel.directAbout }}</p>
      <p v-else class="title">{{ chat.title }}</p>
      <button v-if="chat.type !== 'direct' && canManage" type="button" class="btn sm" @click="rename">{{ t.panel.rename }}</button>
    </section>

    <section class="card">
      <div class="label">{{ t.panel.members }} · {{ people.length }}</div>
      <ul>
        <li v-for="m in people" :key="m.user_id" class="member">
          <Avatar :id="m.user_id" :name="users.name(m.user_id)" :size="30" />
          <span class="mname">{{ users.name(m.user_id) }}</span>
          <span v-if="roleLabel(m.role)" class="badge">{{ roleLabel(m.role) }}</span>
          <button v-if="canManage && m.role !== 'owner' && m.user_id !== auth.user?.id" type="button" class="btn ghost sm danger" @click="remove(m.user_id)">{{ t.panel.remove }}</button>
        </li>
      </ul>
    </section>

    <section v-if="bots.length" class="card">
      <div class="label">{{ t.panel.botsInRoom }} · {{ bots.length }}</div>
      <ul>
        <li v-for="m in bots" :key="m.user_id" class="member">
          <Avatar :id="m.user_id" :name="users.name(m.user_id)" kind="bot" :size="30" />
          <span class="mname">{{ users.name(m.user_id) }}</span>
          <button v-if="canManage" type="button" class="btn ghost sm danger" @click="remove(m.user_id)">{{ t.panel.remove }}</button>
        </li>
      </ul>
    </section>

    <section v-if="chat.type !== 'direct' && canManage" class="card">
      <button type="button" class="btn sm" @click="adding = !adding">{{ t.panel.addMembers }}</button>
      <template v-if="adding">
        <input v-model="query" class="input" type="search" :placeholder="t.newChat.search" />
        <ul class="candidates">
          <li v-for="u in candidates" :key="u.id">
            <button type="button" class="pick" @click="add(u.id)">
              <Avatar :id="u.id" :name="u.display_name" :kind="u.kind" :size="30" />
              <span>{{ u.display_name }}</span>
            </button>
          </li>
        </ul>
      </template>
    </section>

    <section v-if="chat.type !== 'direct' && chats.myMembership?.role !== 'owner'">
      <button type="button" class="btn sm danger" @click="leave">{{ t.panel.leave }}</button>
    </section>
  </aside>
</template>

<style scoped>
.side { width: var(--panel-w); flex: 0 0 var(--panel-w); border-left: 1px solid var(--border); padding: 16px 14px; display: flex; flex-direction: column; gap: 12px; overflow: auto; background: var(--panel); }
.head { display: flex; align-items: center; justify-content: space-between; padding: 0 4px; }
.card { display: flex; flex-direction: column; gap: 10px; padding: 14px; border-radius: var(--r-card); background: var(--chip); }
.title { font-weight: 700; }
.member { display: flex; align-items: center; gap: 10px; min-height: 32px; }
.mname { flex-grow: 1; min-width: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; font-weight: 600; }
.candidates { max-height: 240px; overflow: auto; }
.pick { width: 100%; display: flex; align-items: center; gap: 10px; padding: 8px 10px; border: 0; background: transparent; text-align: left; min-height: 44px; border-radius: 12px; font-weight: 600; }
.pick:hover { background: var(--chip-strong); }
@media (max-width: 1023px) { .side { position: absolute; right: 0; top: 0; bottom: 0; box-shadow: var(--shadow); z-index: 10; backdrop-filter: blur(20px); } }
@media (max-width: 767px) { .side { width: 100%; } }
</style>
