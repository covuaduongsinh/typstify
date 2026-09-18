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

export interface SearchResponse {
  packages: TypstPackage[]
  total: number
}
