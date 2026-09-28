// Mirrors typst/pkg.TypstPkg (which embeds tpix-cli/api.SearchResult).
export interface TypstPackage {
  name: string
  namespace: string
  description: string
  latest_version: string
  published_at: string
  license: string
  is_template: boolean
  authors: string[] | null
  categories: string[] | null
  IsCached: boolean
}

// Mirrors tpix-cli/api.PackageVersionInfo.
export interface PackageVersionInfo {
  version: string
}

// Mirrors typst/pkg.TypstPkg as returned by GET /api/packages/detail, which
// additionally carries the full version list (TypstPackage only has the
// latest one).
export interface TypstPackageDetail extends TypstPackage {
  Versions: PackageVersionInfo[] | null
}

export interface SearchResponse {
  packages: TypstPackage[]
  total: number
}

// Preset categories a Typst package/template can be tagged with (mirrors
// typst/pkg.PresetCategories -- kept in sync manually since it's a small,
// rarely-changing constant list).
export const PACKAGE_CATEGORIES = [
  'components',
  'visualization',
  'model',
  'layout',
  'text',
  'languages',
  'scripting',
  'integration',
  'utility',
  'fun',
  'book',
  'report',
  'paper',
  'thesis',
  'poster',
  'flyer',
  'presentation',
  'cv',
  'office',
] as const
