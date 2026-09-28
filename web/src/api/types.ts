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
