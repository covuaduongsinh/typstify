// Wire format of /ws/collab: [type][channel][payload]. Mirrors the
// constants in service/collab/room.go.
import type { ProjectRole } from '../../api/types'

export const MsgType = {
  Update: 0,
  Awareness: 1,
  Control: 2,
  Synced: 3,
  Compact: 4,
  Save: 5,
} as const

export const Channel = { Text: 0, Comments: 1 } as const
export type ChannelId = (typeof Channel)[keyof typeof Channel]

export interface PeerInfo {
  id: number
  user: string
  name: string
  role: ProjectRole
  leader: boolean
}

export type ControlMessage =
  | {
      type: 'hello'
      you: number
      role: ProjectRole
      epoch: string
      /** Hash of the file on disk the room's text last matched. */
      diskHash: string
      leader: boolean
      peers: PeerInfo[]
    }
  | { type: 'peers'; peers: PeerInfo[] }
  | { type: 'leader'; leader?: boolean }
  | { type: 'compact'; channel: ChannelId }
  | { type: 'external'; base: string; text: string; hash: string }
  | { type: 'error'; message: string }

export function frame(type: number, channel: number, payload: Uint8Array = new Uint8Array()): Uint8Array<ArrayBuffer> {
  const out = new Uint8Array(payload.length + 2)
  out[0] = type
  out[1] = channel
  out.set(payload, 2)
  return out
}

export function parseFrame(data: Uint8Array): { type: number; channel: number; payload: Uint8Array } | null {
  if (data.length < 2) return null
  return { type: data[0], channel: data[1], payload: data.subarray(2) }
}

export function parseControl(payload: Uint8Array): ControlMessage | null {
  try {
    return JSON.parse(new TextDecoder().decode(payload)) as ControlMessage
  } catch {
    return null
  }
}
