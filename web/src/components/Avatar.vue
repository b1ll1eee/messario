<script setup lang="ts">
import { computed } from 'vue'
import { avatarStyle, initials } from '../format'

const props = defineProps<{ id: number; name: string; kind?: 'user' | 'bot' | 'channel'; size?: 'sm' | 'md' | 'lg' }>()
const style = computed(() => (props.kind === 'bot' || props.kind === 'channel' ? undefined : avatarStyle(props.id)))
</script>

<template>
  <span class="avatar" :class="[size ?? 'md', kind]" :style="style" aria-hidden="true">
    <svg v-if="kind === 'bot'" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="8" width="16" height="12" rx="3" /><path d="M12 4v4" /><circle cx="9" cy="14" r="1" /><circle cx="15" cy="14" r="1" /></svg>
    <svg v-else-if="kind === 'channel'" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 11v2a1 1 0 0 0 1 1h2l5 4V6L6 10H4a1 1 0 0 0-1 1z" /><path d="M15 9a4 4 0 0 1 0 6" /></svg>
    <template v-else>{{ initials(name) }}</template>
  </span>
</template>
