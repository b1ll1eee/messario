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
  el.style.height = Math.min(el.scrollHeight, 180) + 'px'
}
</script>

<template>
  <form class="composer" @submit.prevent="submit">
    <div class="box" :class="{ disabled }">
      <label for="composer" class="sr-only">{{ placeholder }}</label>
      <textarea id="composer" ref="area" v-model="text" rows="1" :placeholder="disabled ? t.chat.readOnly : placeholder" :disabled="disabled" @keydown="onKey" @input="resize"></textarea>
      <input ref="picker" type="file" multiple class="sr-only" :disabled="disabled" @change="onPick" />
      <button type="button" class="tool" :disabled="disabled" :aria-label="t.chat.attach" :title="t.chat.attach" @click="picker?.click()">
        <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="m21 11.5-8.5 8.5a5.5 5.5 0 0 1-7.8-7.8l8.5-8.5a3.5 3.5 0 0 1 5 5L9.7 17.2a1.5 1.5 0 0 1-2.1-2.1L15 7.7" /></svg>
      </button>
      <button v-if="channel" type="submit" class="btn primary" :disabled="disabled || !text.trim()">{{ t.chat.post }}</button>
      <button v-else type="submit" class="send" :disabled="disabled || !text.trim()" aria-label="ส่ง">
        <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14M13 6l6 6-6 6" /></svg>
      </button>
    </div>
  </form>
</template>

<style scoped>
.composer { padding: 10px 20px 18px; }
.box { display: flex; align-items: flex-end; gap: 6px; min-height: 54px; padding: 7px 7px 7px 18px; border-radius: 18px; background: var(--panel-solid); border: 1px solid var(--border); box-shadow: var(--shadow); }
.box:focus-within { border-color: var(--border-strong); }
.box.disabled { opacity: 0.7; }
textarea { flex-grow: 1; border: 0; outline: 0; resize: none; background: transparent; padding: 9px 0; min-width: 0; max-height: 180px; line-height: 1.5; font-size: 15px; font-weight: 500; }
textarea::placeholder { color: var(--faint); }
.tool { width: 38px; height: 38px; border-radius: 12px; border: 0; background: transparent; color: var(--muted); display: flex; align-items: center; justify-content: center; flex: 0 0 38px; }
.tool:hover { color: var(--ink); background: var(--chip); }
.send { width: 40px; height: 40px; border-radius: 13px; border: 0; background: var(--accent-grad); color: var(--accent-ink); display: flex; align-items: center; justify-content: center; flex: 0 0 40px; box-shadow: var(--accent-glow); }
.send:disabled { opacity: 0.4; box-shadow: none; cursor: default; }
.sr-only { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); }
@media (max-width: 767px) { .composer { padding: 8px 12px calc(12px + env(safe-area-inset-bottom)); } }
</style>
