// rolesFromToken decodes a JWT access token and extracts role names, supporting
// both a top-level "roles" claim and Keycloak's realm_access.roles.
export function rolesFromToken(token?: string): string[] {
  if (!token) return []
  const parts = token.split('.')
  if (parts.length < 2) return []
  try {
    const json = atob(parts[1].replace(/-/g, '+').replace(/_/g, '/'))
    const payload = JSON.parse(json) as {
      roles?: unknown
      realm_access?: { roles?: unknown }
    }
    if (Array.isArray(payload.roles)) return payload.roles.filter((r): r is string => typeof r === 'string')
    const ra = payload.realm_access?.roles
    if (Array.isArray(ra)) return ra.filter((r): r is string => typeof r === 'string')
    return []
  } catch {
    return []
  }
}

export function isAgent(roles: string[]): boolean {
  return roles.includes('agent') || roles.includes('admin')
}
