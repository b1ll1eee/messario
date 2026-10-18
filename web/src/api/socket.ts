/**
 * WebSocket client with reconnect (exponential backoff + jitter), heartbeat
 * and request/ack correlation. It knows nothing about chats; the store
 * subscribes to events and decides what to do.
 */

export interface ServerFrame {
  t: string
  id?: string
  p?: unknown
}

export interface ClientFrame {
  t: string
  id?: string
  [key: string]: unknown
}

type Listener = (frame: ServerFrame) => void

const BACKOFF_MIN_MS = 1000
const BACKOFF_MAX_MS = 30000
const PING_EVERY_MS = 20000
const PONG_WITHIN_MS = 10000
const ACK_TIMEOUT_MS = 15000

export class Socket {
  private ws: WebSocket | null = null
  private listeners = new Set<Listener>()
  private attempts = 0
  private closedByUser = false
  private reconnectTimer: number | undefined
  private pingTimer: number | undefined
  private pongTimer: number | undefined
  private seq = 0
  private pending = new Map<string, { resolve: (f: ServerFrame) => void; reject: (e: Error) => void; timer: number }>()

  connected = false
  onStatus: (connected: boolean) => void = () => {}

  connect() {
    this.closedByUser = false
    this.open()
  }

  close() {
    this.closedByUser = true
    window.clearTimeout(this.reconnectTimer)
    this.stopHeartbeat()
    this.ws?.close()
    this.ws = null
  }

  subscribe(fn: Listener) {
    this.listeners.add(fn)
    return () => this.listeners.delete(fn)
  }

  /** Fire-and-forget frame (e.g. read pointer). Dropped silently when offline. */
  send(frame: ClientFrame) {
    if (this.ws?.readyState === WebSocket.OPEN) this.ws.send(JSON.stringify(frame))
  }

  /** Frame that expects an ack/err with the same id. Rejects when offline. */
  request(frame: ClientFrame): Promise<ServerFrame> {
    if (this.ws?.readyState !== WebSocket.OPEN) return Promise.reject(new Error('offline'))
    const id = 'r' + ++this.seq
    return new Promise((resolve, reject) => {
      const timer = window.setTimeout(() => {
        this.pending.delete(id)
        reject(new Error('timeout'))
      }, ACK_TIMEOUT_MS)
      this.pending.set(id, { resolve, reject, timer })
      this.ws!.send(JSON.stringify({ ...frame, id }))
    })
  }

  private open() {
    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    const ws = new WebSocket(`${proto}://${location.host}/ws`)
    this.ws = ws
    ws.onopen = () => {
      this.attempts = 0
      this.connected = true
      this.onStatus(true)
      this.startHeartbeat()
    }
    ws.onmessage = (ev) => {
      let frame: ServerFrame
      try {
        frame = JSON.parse(ev.data)
      } catch {
        return
      }
      if (frame.t === 'pong') {
        window.clearTimeout(this.pongTimer)
        return
      }
      if (frame.id && this.pending.has(frame.id)) {
        const p = this.pending.get(frame.id)!
        this.pending.delete(frame.id)
        window.clearTimeout(p.timer)
        if (frame.t === 'err') p.reject(Object.assign(new Error((frame.p as { message?: string })?.message ?? 'error'), { frame }))
        else p.resolve(frame)
        return
      }
      for (const l of this.listeners) l(frame)
    }
    ws.onclose = () => {
      if (this.ws !== ws) return
      this.ws = null
      this.connected = false
      this.onStatus(false)
      this.stopHeartbeat()
      this.failPending()
      if (!this.closedByUser) this.scheduleReconnect()
    }
    ws.onerror = () => ws.close()
  }

  private scheduleReconnect() {
    const base = Math.min(BACKOFF_MAX_MS, BACKOFF_MIN_MS * 2 ** this.attempts++)
    const jitter = Math.random() * base * 0.3
    this.reconnectTimer = window.setTimeout(() => this.open(), base + jitter)
  }

  private startHeartbeat() {
    this.pingTimer = window.setInterval(() => {
      this.send({ t: 'ping' })
      window.clearTimeout(this.pongTimer)
      this.pongTimer = window.setTimeout(() => this.ws?.close(), PONG_WITHIN_MS)
    }, PING_EVERY_MS)
  }

  private stopHeartbeat() {
    window.clearInterval(this.pingTimer)
    window.clearTimeout(this.pongTimer)
  }

  private failPending() {
    for (const [, p] of this.pending) {
      window.clearTimeout(p.timer)
      p.reject(new Error('disconnected'))
    }
    this.pending.clear()
  }
}

export const socket = new Socket()
