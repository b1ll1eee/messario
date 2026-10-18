export type UserKind = 'user' | 'bot'
export type Role = 'admin' | 'member'
export type UserStatus = 'active' | 'suspended'
export type ChatType = 'direct' | 'group' | 'channel'
export type MemberRole = 'owner' | 'admin' | 'member'
export type MessageKind = 'text' | 'system' | 'file'

export interface User {
  id: number
  kind: UserKind
  username: string
  display_name: string
  email?: string
  department?: string
  title?: string
  role: Role
  status: UserStatus
  created_at: string
}

export interface Chat {
  id: number
  type: ChatType
  title: string
  created_by: number
  last_seq: number
  created_at: string
}

export interface ChatSummary extends Chat {
  peer_id?: number
  read_seq: number
  unread: number
  last_body: string
  last_kind: string
  last_sender_id: number
  last_at: string
  member_count: number
}

export interface Attachment {
  file_id: number
  name: string
  size: number
  mime: string
}

export interface Message {
  id: number
  chat_id: number
  seq: number
  sender_id: number
  client_msg_id?: string
  kind: MessageKind
  body: string
  attachment?: Attachment
  created_at: string
  /** client-only: queued in the outbox, not yet acknowledged */
  pending?: boolean
  /** client-only: the server rejected it */
  failed?: boolean
}

export interface Member {
  chat_id: number
  user_id: number
  role: MemberRole
  read_seq: number
  joined_at: string
}

export interface ReadInfo {
  count: number
  total: number
  readers: Member[]
}

export interface ApiErrorBody {
  error: { code: string; message: string }
}

export interface LicenseStatus {
  seats: number
  used_seats: number
  status: string
}
