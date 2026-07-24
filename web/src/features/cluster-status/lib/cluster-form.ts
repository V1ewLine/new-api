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
import { z } from 'zod'

import { normalizeAgentAddress } from './connection.ts'

export const clusterFormSchema = z.object({
  modelId: z.number().int().positive('Please select a model'),
  name: z
    .string()
    .trim()
    .min(1, 'Cluster name is required')
    .max(128, 'Cluster name must be 128 characters or fewer'),
  agentAddress: z
    .string()
    .trim()
    .min(1, 'Agent address is required')
    .max(2048, 'Agent address is too long')
    .refine(
      (value) => normalizeAgentAddress(value) !== null,
      'Agent address must include an IP or hostname and port'
    ),
})

export type ClusterFormValues = z.infer<typeof clusterFormSchema>
