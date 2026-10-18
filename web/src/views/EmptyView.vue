<script setup lang="ts">
import { t } from '../strings'
import Logo from '../components/Logo.vue'
</script>

<template>
  <div class="empty">
    <div class="mark"><Logo :size="64" /></div>
    <h2>{{ t.empty.title }}</h2>
    <p class="muted">{{ t.empty.body }}</p>
  </div>
</template>

<style scoped>
.empty { flex-grow: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 6px; }
.mark { margin-bottom: 10px; opacity: 0.9; }
h2 { font-size: 18px; font-weight: 800; letter-spacing: -0.02em; }
</style>
