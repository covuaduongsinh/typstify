import { CHANGE_SERVER_PATH, isMobileApp } from '../lib/platform'

/** "Đổi máy chủ", shown only in the mobile app. */
export function ChangeServerLink({ className = 'btn-link' }: { className?: string }) {
  if (!isMobileApp()) return null
  return (
    <a className={className} href={CHANGE_SERVER_PATH}>
      Đổi máy chủ
    </a>
  )
}
