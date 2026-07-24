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
import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')

function stableStringify(obj) {
  return JSON.stringify(obj, null, 2) + '\n'
}

const english = {
  'Cluster data exported': 'Cluster data exported',
  'Cluster list (CSV)': 'Cluster list (CSV)',
  'Complete cluster snapshot (ZIP)': 'Complete cluster snapshot (ZIP)',
  'Complete snapshot (ZIP)': 'Complete snapshot (ZIP)',
  Export: 'Export',
  'Export Cluster Data': 'Export Cluster Data',
  'Export content': 'Export content',
  'Exporting...': 'Exporting...',
  'Exports one row for every cluster linked to this model.':
    'Exports one row for every cluster linked to this model.',
  'Exports one row per model using all records that match the current filters.':
    'Exports one row per model using all records that match the current filters.',
  'Exports the complete normalized latest snapshot as a JSON file.':
    'Exports the complete normalized latest snapshot as a JSON file.',
  'Exports the latest stored snapshot. Agent addresses, Bearer Tokens, and diagnostic payloads are excluded.':
    'Exports the latest stored snapshot. Agent addresses, Bearer Tokens, and diagnostic payloads are excluded.',
  'Exported cluster telemetry ({{scope}}, {{format}}, {{count}} clusters)':
    'Exported cluster telemetry ({{scope}}, {{format}}, {{count}} clusters)',
  'Failed to export cluster data': 'Failed to export cluster data',
  'Includes model, cluster, GPU device, engine load, and normalized telemetry files.':
    'Includes model, cluster, GPU device, engine load, and normalized telemetry files.',
  'Includes the cluster row, GPU devices, engine loads, and normalized telemetry.':
    'Includes the cluster row, GPU devices, engine loads, and normalized telemetry.',
  'Model summary (CSV)': 'Model summary (CSV)',
  'Normalized snapshot (JSON)': 'Normalized snapshot (JSON)',
}

const newKeys = {
  en: english,
  zh: {
    'Cluster data exported': '集群数据已导出',
    'Cluster list (CSV)': '集群列表（CSV）',
    'Complete cluster snapshot (ZIP)': '完整集群快照（ZIP）',
    'Complete snapshot (ZIP)': '完整快照（ZIP）',
    Export: '导出',
    'Export Cluster Data': '导出集群数据',
    'Export content': '导出内容',
    'Exporting...': '正在导出...',
    'Exports one row for every cluster linked to this model.':
      '导出该模型关联的所有集群，每个集群一行。',
    'Exports one row per model using all records that match the current filters.':
      '根据当前筛选条件导出全部匹配记录，每个模型一行。',
    'Exports the complete normalized latest snapshot as a JSON file.':
      '将完整的最新标准化快照导出为 JSON 文件。',
    'Exports the latest stored snapshot. Agent addresses, Bearer Tokens, and diagnostic payloads are excluded.':
      '导出已存储的最新快照，不包含 Agent 地址、Bearer Token 和诊断载荷。',
    'Exported cluster telemetry ({{scope}}, {{format}}, {{count}} clusters)':
      '已导出集群遥测数据（范围：{{scope}}，格式：{{format}}，共 {{count}} 个集群）',
    'Failed to export cluster data': '集群数据导出失败',
    'Includes model, cluster, GPU device, engine load, and normalized telemetry files.':
      '包含模型、集群、GPU 设备、引擎负载和标准化遥测文件。',
    'Includes the cluster row, GPU devices, engine loads, and normalized telemetry.':
      '包含集群记录、GPU 设备、引擎负载和标准化遥测数据。',
    'Model summary (CSV)': '模型汇总（CSV）',
    'Normalized snapshot (JSON)': '标准化快照（JSON）',
  },
  'zh-TW': english,
  fr: english,
  ja: english,
  ru: english,
  vi: english,
}

async function main() {
  let totalAdded = 0

  for (const [locale, trans] of Object.entries(newKeys)) {
    const filePath = path.join(LOCALES_DIR, `${locale}.json`)
    const json = JSON.parse(await fs.readFile(filePath, 'utf8'))

    let count = 0
    for (const [key, value] of Object.entries(trans)) {
      if (!Object.prototype.hasOwnProperty.call(json.translation, key)) {
        json.translation[key] = value
        count++
      } else if (json.translation[key] !== value) {
        json.translation[key] = value
        count++
      }
    }

    if (count > 0) {
      json.translation = Object.fromEntries(
        Object.entries(json.translation).sort(([a], [b]) => a.localeCompare(b))
      )
      await fs.writeFile(filePath, stableStringify(json), 'utf8')
    }

    console.log(`${locale}: ${count} translations applied`)
    totalAdded += count
  }

  console.log(`\nTotal: ${totalAdded} translations applied`)
}

main().catch((err) => {
  console.error(err)
  process.exitCode = 1
})
