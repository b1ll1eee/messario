<script setup lang="ts">
defineProps<{ size?: number }>()
</script>

<template>
  <svg :width="size ?? 34" :height="size ?? 34" viewBox="0 0 64 64" role="img" aria-label="Messario">
    <rect width="64" height="64" rx="18" fill="var(--accent)" />
    <path d="M12 13h26a6 6 0 0 1 6 6v11a6 6 0 0 1-6 6H24l-7 6v-6h-5a6 6 0 0 1-6-6V19a6 6 0 0 1 6-6z" fill="var(--accent-ink)" opacity="0.55" />
    <path d="M26 28h26a6 6 0 0 1 6 6v11a6 6 0 0 1-6 6h-5v6l-7-6H26a6 6 0 0 1-6-6V34a6 6 0 0 1 6-6z" fill="var(--accent-ink)" stroke="var(--accent)" stroke-width="3" stroke-linejoin="round" />
    <path d="M33 44V34l6 6 6-6v10" fill="none" stroke="var(--accent)" stroke-width="3.6" stroke-linecap="round" stroke-linejoin="round" />
  </svg>
</template>
