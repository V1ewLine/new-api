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
const HTTP_SCHEME_PATTERN = /^https?:\/\//i
const HOST_PORT_PATTERN = /^[^:]+:\d+$/
const IPV6_PORT_PATTERN = /^\[[^\]]+\]:\d+$/

export function normalizeAgentAddress(value: string): string | null {
  const trimmed = value.trim()
  if (!trimmed) {
    return null
  }

  const candidate = HTTP_SCHEME_PATTERN.test(trimmed)
    ? trimmed
    : `http://${trimmed}`
  let parsed: URL
  try {
    parsed = new URL(candidate)
  } catch {
    return null
  }

  if (
    (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') ||
    !parsed.hostname ||
    parsed.username ||
    parsed.password ||
    parsed.search ||
    parsed.hash ||
    parsed.pathname !== '/'
  ) {
    return null
  }

  const authorityAndPath = candidate.slice(candidate.indexOf('://') + 3)
  const authority = authorityAndPath.split(/[/?#]/, 1)[0]
  if (
    !HOST_PORT_PATTERN.test(authority) &&
    !IPV6_PORT_PATTERN.test(authority)
  ) {
    return null
  }

  return parsed.origin
}
