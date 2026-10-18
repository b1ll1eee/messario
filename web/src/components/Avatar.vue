<script setup lang="ts">
import { computed } from 'vue'
import { avatarStyle, initials } from '../format'

const props = defineProps<{ id: number; name: string; kind?: 'user' | 'bot' | 'channel'; size?: number; presence?: boolean }>()
const px = computed(() => props.size ?? 36)
const style = computed(() => ({
  width: px.value + 'px',
  height: px.value + 'px',
  fontSize: Math.round(px.value * 0.34) + 'px',
  ...(props.kind === 'bot' || props.kind === 'channel' ? {} : avatarStyle(props.id)),
}))
const dot = computed(() => ({ width: Math.round(px.value * 0.28) + 'px', height: Math.round(px.value * 0.28) + 'px' }))
</script>

<template>
  <span class="avatar" :class="kind" :style="style" aria-hidden="true">
    <svg v-if="kind === 'bot'" :width="Math.round(px * 0.5)" :height="Math.round(px * 0.5)" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="8" width="16" height="12" rx="4" /><path d="M12 4v4" /><circle cx="9" cy="14" r="1" /><circle cx="15" cy="14" r="1" /></svg>
    <svg v-else-if="kind === 'channel'" :width="Math.round(px * 0.48)" :height="Math.round(px * 0.48)" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M3 11v2a1 1 0 0 0 1 1h2l5 4V6L6 10H4a1 1 0 0 0-1 1z" /><path d="M15 9a4 4 0 0 1 0 6" /></svg>
    <template v-else>{{ initials(name) }}</template>
    <span v-if="presence" class="presence" :style="dot"></span>
  </span>
</template>
