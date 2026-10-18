<script setup lang="ts">
import { t } from '../strings'
</script>

<template>
  <div class="empty">
    <div class="mark" aria-hidden="true">N</div>
    <h2>{{ t.empty.title }}</h2>
    <p class="muted">{{ t.empty.body }}</p>
  </div>
</template>

<style scoped>
.empty { flex-grow: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px; color: var(--muted); }
.mark { width: 48px; height: 48px; border-radius: 14px; background: var(--surface-2); color: var(--faint); display: flex; align-items: center; justify-content: center; font-weight: 700; font-size: 22px; margin-bottom: 8px; }
h2 { font-size: 16px; font-weight: 600; color: var(--ink); }
</style>
