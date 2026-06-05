const rawBasePath = import.meta.env.BASE_URL || '/'

export function withBasePath(path: string): string {
  const cleanPath = path.startsWith('/') ? path : `/${path}`
  if (rawBasePath === '/') return cleanPath
  return `${rawBasePath.replace(/\/$/, '')}${cleanPath}`
}

export function stripBasePath(pathname: string): string {
  if (rawBasePath === '/') return pathname || '/'

  const base = rawBasePath.replace(/\/$/, '')
  if (pathname === base) return '/'
  if (pathname.startsWith(`${base}/`)) {
    return pathname.slice(base.length) || '/'
  }
  return pathname || '/'
}

export function staticAssetPath(path: string): string {
  if (/^(https?:|mailto:|tel:|data:|blob:)/i.test(path)) return path
  return withBasePath(path)
}
