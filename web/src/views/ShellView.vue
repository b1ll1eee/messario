<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import { useChats } from '../stores/chats'
import { useUsers } from '../stores/users'
import Rail from '../components/Rail.vue'
import ChatList from '../components/ChatList.vue'

const chats = useChats()
const users = useUsers()
const route = useRoute()

// On phones only one column shows: the list on /, the conversation on /c/:id.
const showDetail = computed(() => route.name !== 'home')

onMounted(async () => {
  try {
    const saved = localStorage.getItem('theme')
    if (saved === 'dark' || saved === 'light') document.documentElement.dataset.theme = saved
  } catch {
    /* ignore */
  }
  await Promise.all([users.load(), chats.loadList()])
  chats.start()
})

onBeforeUnmount(() => chats.stop())
</script>

<template>
  <div class="shell" :class="{ detail: showDetail }">
    <Rail />
    <ChatList class="col-list" />
    <main class="col-detail">
      <RouterView />
    </main>
  </div>
</template>

<style scoped>
.shell { display: flex; height: 100%; min-height: 0; }
.col-detail { flex: 1 1 auto; min-width: 0; display: flex; position: relative; background: var(--surface); }
@media (max-width: 767px) {
  .shell > :deep(.rail) { display: none; }
  .shell:not(.detail) .col-detail { display: none; }
  .shell.detail .col-list { display: none; }
}
</style>
