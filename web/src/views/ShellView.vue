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
  await Promise.all([users.load(), chats.loadList()])
  chats.start()
})

onBeforeUnmount(() => chats.stop())
</script>

<template>
  <div class="shell" :class="{ detail: showDetail }">
    <Rail />
    <ChatList class="col-list" />
    <main class="col-detail panel">
      <RouterView />
    </main>
  </div>
</template>

<style scoped>
.shell { display: flex; gap: var(--gap); height: 100%; min-height: 0; padding: var(--gap); }
.col-detail { flex: 1 1 auto; min-width: 0; display: flex; position: relative; overflow: hidden; }
@media (max-width: 767px) {
  .shell { padding: 0; gap: 0; }
  .shell > :deep(.rail) { display: none; }
  .shell:not(.detail) .col-detail { display: none; }
  .shell.detail .col-list { display: none; }
  .col-detail, .shell :deep(.list) { border-radius: 0; border: 0; }
}
</style>
