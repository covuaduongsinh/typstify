export interface DropboxAccount {
  account_id: string
  display_name: string
  email: string
}

export interface DropboxSyncResult {
  mode: 'two-way' | 'push' | 'pull'
  uploaded: string[]
  downloaded: string[]
  conflicts: string[]
  errors: string[]
  duration_ms: number
  timestamp: string
  success: boolean
}

export interface DropboxStatus {
  connected: boolean
  account?: DropboxAccount
  appKey?: string
  syncFolder: string
  autoSync: boolean
  autoSyncInterval: number
  syncOnSave: boolean
  lastSyncTime?: string
  isSyncing: boolean
  lastResult?: DropboxSyncResult
}

export interface DropboxProjectItem {
  name: string
  path: string
  server_modified?: string
}
