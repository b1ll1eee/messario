<script setup lang="ts">
import { ref } from 'vue'
import { t } from '../strings'

const props = defineProps<{ placeholder: string; disabled?: boolean; channel?: boolean }>()
const emit = defineEmits<{ send: [body: string]; attach: [file: File] }>()
const text = ref('')
const area = ref<HTMLTextAreaElement | null>(null)
const picker = ref<HTMLInputElement | null>(null)

function onPick(e: Event) {
  const input = e.target as HTMLInputElement
  for (const f of input.files ?? []) emit('attach', f)
  input.value = ''
}

function submit() {
  const body = text.value.trim()
  if (!body || props.disabled) return
  emit('send', body)
  text.value = ''
  resize()
}

function onKey(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) {
    e.preventDefault()
    submit()
  }
}

function resize() {
  const el = area.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 200) + 'px'
}
</script>

<template>
  <form class="composer" @submit.prevent="submit">
    <div class="box" :class="{ disabled }">
      <label for="composer" class="sr-only">{{ placeholder }}</label>
      <textarea id="composer" ref="area" v-model="text" rows="1" :placeholder="placeholder" :disabled="disabled" @keydown="onKey" @input="resize"></textarea>
      <input ref="picker" type="file" multiple class="sr-only" :disabled="disabled" @change="onPick" />
      <button type="button" class="attach" :disabled="disabled" :aria-label="t.chat.attach" :title="t.chat.attach" @click="picker?.click()">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="m21 11.5-8.5 8.5a5.5 5.5 0 0 1-7.8-7.8l8.5-8.5a3.5 3.5 0 0 1 5 5L9.7 17.2a1.5 1.5 0 0 1-2.1-2.1L15 7.7" /></svg>
      </button>
      <button v-if="channel" type="submit" class="btn primary" :disabled="disabled || !text.trim()">{{ t.chat.post }}</button>
      <button v-else type="submit" class="send" :disabled="disabled || !text.trim()" aria-label="ส่ง">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14M13 6l6 6-6 6" /></svg>
      </button>
    </div>
    <div class="hint faint">{{ disabled ? t.chat.readOnly : t.chat.hint }}</div>
  </form>
</template>

<style scoped>
.composer { padding: 8px 20px 16px; display: flex; flex-direction: column; gap: 6px; }
.box { display: flex; align-items: flex-end; gap: 8px; border: 1px solid var(--border-strong); border-radius: 14px; padding: 6px 6px 6px 14px; background: var(--surface); box-shadow: var(--shadow); }
.box:focus-within { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-soft); }
.box.disabled { background: var(--surface-2); }
textarea { flex-grow: 1; border: 0; outline: 0; resize: none; background: transparent; padding: 6px 0; min-width: 0; max-height: 200px; line-height: 1.5; }
.attach { width: 36px; height: 36px; border-radius: 9px; border: 0; background: transparent; color: var(--muted); display: flex; align-items: center; justify-content: center; flex: 0 0 36px; }
.attach:hover { color: var(--ink); background: var(--surface-2); }
.send { width: 36px; height: 36px; border-radius: 9px; border: 0; background: var(--accent); color: var(--accent-ink); display: flex; align-items: center; justify-content: center; flex: 0 0 36px; }
.send:disabled { opacity: 0.4; cursor: default; }
.hint { font-size: 12px; }
.sr-only { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); }
@media (max-width: 767px) { .composer { padding: 8px 12px 12px; } .hint { display: none; } }
</style>
