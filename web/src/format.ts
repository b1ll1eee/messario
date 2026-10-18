import { t } from './strings'

const timeFmt = new Intl.DateTimeFormat('th-TH', { hour: '2-digit', minute: '2-digit' })
const dayFmt = new Intl.DateTimeFormat('th-TH', { day: 'numeric', month: 'short' })
const weekdayFmt = new Intl.DateTimeFormat('th-TH', { weekday: 'short' })
const fullFmt = new Intl.DateTimeFormat('th-TH', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })

export function timeOf(iso: string) {
  return timeFmt.format(new Date(iso))
}

export function fullTime(iso: string) {
  return fullFmt.format(new Date(iso))
}

export function dayLabel(iso: string) {
  const d = new Date(iso)
  return isSameDay(d, new Date()) ? t.chat.today : `${weekdayFmt.format(d)} ${dayFmt.format(d)}`
}

/** Chat-list timestamp: time today, "เมื่อวาน", weekday within a week, else date. */
export function listTime(iso: string) {
  const d = new Date(iso)
  const now = new Date()
  if (isSameDay(d, now)) return timeFmt.format(d)
  const yesterday = new Date(now)
  yesterday.setDate(now.getDate() - 1)
  if (isSameDay(d, yesterday)) return t.list.yesterday
  if (now.getTime() - d.getTime() < 6 * 86400000) return weekdayFmt.format(d)
  return dayFmt.format(d)
}

export function isSameDay(a: Date, b: Date) {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate()
}

/** Two-character initials for avatars; works for Thai and Latin names. */
export function initials(name: string) {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  const first = [...parts[0]!]
  if (parts.length === 1) return first.slice(0, 2).join('')
  return first[0]! + [...parts[1]!][0]!
}

const AVATAR_TINTS = [
  ['#dbeafe', '#1e3a8a'],
  ['#fce7f3', '#9d174d'],
  ['#ede9fe', '#4c1d95'],
  ['#dcfce7', '#14532d'],
  ['#fef3c7', '#78350f'],
  ['#cffafe', '#155e75'],
] as const

export function avatarStyle(id: number) {
  const [bg, fg] = AVATAR_TINTS[Math.abs(id) % AVATAR_TINTS.length]!
  return { background: bg, color: fg }
}

export function fileSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}
