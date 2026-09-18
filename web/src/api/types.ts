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
}
