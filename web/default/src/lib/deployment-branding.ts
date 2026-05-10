/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
export interface DeploymentBrandingConfig {
  systemName?: string
  logo?: string
}

const BRANDING_CONFIG_PATH = '/branding.json'

function normalizeString(value: unknown): string | undefined {
  if (typeof value !== 'string') return undefined

  const trimmed = value.trim()
  return trimmed || undefined
}

export function normalizeDeploymentBranding(
  value: unknown
): DeploymentBrandingConfig {
  if (!value || typeof value !== 'object') return {}

  const record = value as Record<string, unknown>

  return {
    systemName: normalizeString(record.systemName),
    logo: normalizeString(record.logo),
  }
}

export async function fetchDeploymentBranding(): Promise<DeploymentBrandingConfig> {
  try {
    const response = await fetch(BRANDING_CONFIG_PATH, {
      cache: 'no-store',
    })

    if (!response.ok) return {}

    const data = (await response.json()) as unknown
    return normalizeDeploymentBranding(data)
  } catch {
    return {}
  }
}
