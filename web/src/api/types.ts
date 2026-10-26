// Mirrors server/fonts_api.go's fontFileInfo.
export interface FontFileInfo {
  name: string
  size: number
}

export interface TreeEntry {
  name: string
  path: string
  isDir: boolean
  size: number
  modTime: string
}

export interface RecentProject {
  Path: string
  RelPath: string
  LastAccessAt: string
}

export interface GeneralSettings {
  rootDir: string
  language: string
  textSize: number
  fontType: string
  theme: string
  checkUpdate: string
  deviceId: string
  externalTypst: string
  externalTinymist: string
}

export interface TypstSettings {
  version: string
  cacheDir: string
  localPkgDir: string
  extraFontPath: string
  useSysInputs: number
  ignoreSystemFonts: number
  ignoreEmbeddedFonts: number
  buildDeps: number
  outputDir: string
}

export interface LspSettings {
  enableLspLogs: number
  enablePowerSaving: number
  openPreviewInBrowser: number
  enablePartialRenderPreview: boolean
  /** Projects the server keeps a tinymist for at once; 0 = default (3). */
  maxOpenProjects: number
}

export interface AgentSettings {
  agentId: string
  agentName: string
  cmd: string
  args: string
  env: string
}

export interface AuthStatus {
  authRequired: boolean
  authenticated: boolean
  username?: string
  displayName?: string
  hasUsers?: boolean
  /** Admins manage every project and the server-wide settings. */
  admin?: boolean
  mfaEnabled?: boolean
  /** The desktop app's server (apps/tauri): sign-in happens at launch. */
  desktop?: boolean
}

// Mirrors service/projects (Role, Project, ShareLink) and server/projects_api.go.
export type ProjectRole = 'owner' | 'editor' | 'commenter' | 'viewer'

export const ROLE_LABELS: Record<ProjectRole, string> = {
  owner: 'Chủ sở hữu',
  editor: 'Chỉnh sửa',
  commenter: 'Góp ý',
  viewer: 'Chỉ xem',
}

// The server's trimmed projectView: no server path, no member list (those
// come from the owner-only /members endpoint).
export interface ProjectInfo {
  id: string
  name: string
  owner: string
  createdAt: string
  updatedAt: string
  role: ProjectRole
}

export interface ProjectMember {
  username: string
  role: ProjectRole
}

export interface ShareLinkInfo {
  id: string
  projectId: string
  role: ProjectRole
  createdBy: string
  createdAt: string
  expiresAt?: string
  uses: number
}

export interface WorkspaceRole {
  registered: boolean
  role: ProjectRole
  project?: ProjectInfo
}

export interface User {
  username: string
  displayName?: string
}

export interface RegisterRequest {
  username: string
  password: string
  displayName?: string
}

export interface LoginRequest {
  username?: string
  password: string
}

export interface ChangePasswordRequest {
  currentPassword: string
  newPassword: string
}

// Mirrors service/history (Snapshot, Change).
export type HistoryKind = 'auto' | 'manual' | 'restore'

export interface HistorySnapshot {
  id: string
  at: string
  author: string
  label?: string
  kind: HistoryKind
  files: number
  added: number
  modified: number
  deleted: number
}

export type HistoryStatus = 'unchanged' | 'modified' | 'deleted' | 'added'

export interface HistoryChange {
  path: string
  status: HistoryStatus
  size: number
}
