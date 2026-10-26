import { api } from '../api/client'

const IMAGE_EXT = /\.(png|jpe?g|gif|svg|webp)$/i
const IMAGE_TYPE = /^image\/(png|jpeg|gif|svg\+xml|webp)$/

/** Image formats Typst's #image accepts. */
export const isImageFile = (f: File) => IMAGE_TYPE.test(f.type) || IMAGE_EXT.test(f.name)

const EXT_BY_TYPE: Record<string, string> = {
  'image/png': 'png',
  'image/jpeg': 'jpg',
  'image/gif': 'gif',
  'image/svg+xml': 'svg',
  'image/webp': 'webp',
}

/** Clipboard images are all called "image.png"; give pasted ones a
 * dated name instead (anh-20261025-143005.png). Dropped files keep theirs. */
export function pastedImageName(file: File, now = new Date()): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  const stamp = `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}-${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`
  const ext = EXT_BY_TYPE[file.type] ?? file.name.split('.').pop()?.toLowerCase() ?? 'png'
  return `anh-${stamp}.${ext}`
}

/** Typst markup for an uploaded image. The path starts at the project
 * root, so it works from any file of the project. */
export const imageMarkup = (projectPath: string) => `#image("/${projectPath}", width: 80%)`

/** Uploads a file into dir (project-relative, "" = root) under name; the
 * server picks a free name if it is taken and returns the final path. */
export async function uploadFile(dir: string, file: Blob, name: string): Promise<string> {
  const q = `dir=${encodeURIComponent(dir)}&name=${encodeURIComponent(name)}`
  const res = await api.postBinary<{ path: string }>(`/api/workspace/upload?${q}`, file)
  return res.path
}
