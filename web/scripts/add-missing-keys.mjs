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
  return `${JSON.stringify(obj, null, 2)}\n`
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

const phase2English = {
  'All charts share this time window. Relative windows refresh automatically.':
    'All charts share this time window. Relative windows refresh automatically.',
  'Apply custom range': 'Apply custom range',
  'Chart time window': 'Chart time window',
  'CPU, memory, and cache utilization over time.':
    'CPU, memory, and cache utilization over time.',
  'Custom range': 'Custom range',
  'Expand {{chart}}': 'Expand {{chart}}',
  'Failed to load telemetry trends': 'Failed to load telemetry trends',
  'Generated token throughput over time.':
    'Generated token throughput over time.',
  'Historical Trends': 'Historical Trends',
  'Host CPU utilization over time.': 'Host CPU utilization over time.',
  'Host memory utilization over time.': 'Host memory utilization over time.',
  'Inference engine cache utilization over time.':
    'Inference engine cache utilization over time.',
  'No samples were stored in the selected time window.':
    'No samples were stored in the selected time window.',
  'No successful samples contain this metric in the time window.':
    'No successful samples contain this metric in the time window.',
  'No telemetry history': 'No telemetry history',
  'No trend data': 'No trend data',
  'Per-GPU Power': 'Per-GPU Power',
  'Poll Success': 'Poll Success',
  'Poll success and component availability in the selected time window.':
    'Poll success and component availability in the selected time window.',
  'Power draw for each GPU device over time.':
    'Power draw for each GPU device over time.',
  'Relative range': 'Relative range',
  'Request Pressure': 'Request Pressure',
  'Requests currently running in the inference engine.':
    'Requests currently running in the inference engine.',
  'Requests waiting for inference engine capacity.':
    'Requests waiting for inference engine capacity.',
  'Resource Utilization': 'Resource Utilization',
  'Running and waiting request counts over time.':
    'Running and waiting request counts over time.',
  'Select chart time window': 'Select chart time window',
  'Telemetry Availability': 'Telemetry Availability',
  'The selected range exceeds the {{days}}-day retention.':
    'The selected range exceeds the {{days}}-day retention.',
  'The start time must be earlier than the end time.':
    'The start time must be earlier than the end time.',
  'This time window applies to every chart on the current page.':
    'This time window applies to every chart on the current page.',
  'This time window applies only to the expanded chart.':
    'This time window applies only to the expanded chart.',
  'Tokens currently held by active engine requests.':
    'Tokens currently held by active engine requests.',
  'Total power reported across all GPU boards.':
    'Total power reported across all GPU boards.',
  '{{count}} samples · {{seconds}}s buckets':
    '{{count}} samples · {{seconds}}s buckets',
}

const phase2Translations = {
  zh: {
    'All charts share this time window. Relative windows refresh automatically.':
      '所有图表共用此时间窗口，相对时间窗口会自动刷新。',
    'Apply custom range': '应用自定义范围',
    'Chart time window': '图表时间窗口',
    'CPU, memory, and cache utilization over time.':
      'CPU、内存和缓存利用率随时间的变化。',
    'Custom range': '自定义范围',
    'Expand {{chart}}': '放大{{chart}}',
    'Failed to load telemetry trends': '遥测趋势加载失败',
    'Generated token throughput over time.': '生成 Token 吞吐量随时间的变化。',
    'Historical Trends': '历史趋势',
    'Host CPU utilization over time.': '主机 CPU 利用率随时间的变化。',
    'Host memory utilization over time.': '主机内存利用率随时间的变化。',
    'Inference engine cache utilization over time.':
      '推理引擎缓存利用率随时间的变化。',
    'No samples were stored in the selected time window.':
      '所选时间窗口内没有已存储的采样。',
    'No successful samples contain this metric in the time window.':
      '此时间窗口内的成功采样均不包含该指标。',
    'No telemetry history': '暂无遥测历史',
    'No trend data': '暂无趋势数据',
    'Per-GPU Power': '单个 GPU 功耗',
    'Poll Success': '采集成功率',
    'Poll success and component availability in the selected time window.':
      '所选时间窗口内的采集成功率和组件可用性。',
    'Power draw for each GPU device over time.':
      '每个 GPU 设备功耗随时间的变化。',
    'Relative range': '相对时间范围',
    'Request Pressure': '请求压力',
    'Requests currently running in the inference engine.':
      '推理引擎中当前正在运行的请求。',
    'Requests waiting for inference engine capacity.':
      '等待推理引擎可用容量的请求。',
    'Resource Utilization': '资源利用率',
    'Running and waiting request counts over time.':
      '运行中和等待中请求数随时间的变化。',
    'Select chart time window': '选择图表时间窗口',
    'Telemetry Availability': '遥测可用性',
    'The selected range exceeds the {{days}}-day retention.':
      '所选范围超过 {{days}} 天的数据保留期限。',
    'The start time must be earlier than the end time.':
      '开始时间必须早于结束时间。',
    'This time window applies to every chart on the current page.':
      '此时间窗口会应用到当前页面的所有图表。',
    'This time window applies only to the expanded chart.':
      '此时间窗口仅应用到当前放大的图表。',
    'Tokens currently held by active engine requests.':
      '当前活动引擎请求占用的 Token 数。',
    'Total power reported across all GPU boards.':
      '所有 GPU 板卡上报的总功耗。',
    '{{count}} samples · {{seconds}}s buckets':
      '{{count}} 个采样 · {{seconds}} 秒分桶',
  },
  'zh-TW': {
    'All charts share this time window. Relative windows refresh automatically.':
      '所有圖表共用此時間視窗，相對時間視窗會自動重新整理。',
    'Apply custom range': '套用自訂範圍',
    'Chart time window': '圖表時間視窗',
    'CPU, memory, and cache utilization over time.':
      'CPU、記憶體與快取使用率隨時間的變化。',
    'Custom range': '自訂範圍',
    'Expand {{chart}}': '放大{{chart}}',
    'Failed to load telemetry trends': '遙測趨勢載入失敗',
    'Generated token throughput over time.': '產生 Token 吞吐量隨時間的變化。',
    'Historical Trends': '歷史趨勢',
    'Host CPU utilization over time.': '主機 CPU 使用率隨時間的變化。',
    'Host memory utilization over time.': '主機記憶體使用率隨時間的變化。',
    'Inference engine cache utilization over time.':
      '推理引擎快取使用率隨時間的變化。',
    'No samples were stored in the selected time window.':
      '所選時間視窗內沒有已儲存的採樣。',
    'No successful samples contain this metric in the time window.':
      '此時間視窗內的成功採樣均不包含此指標。',
    'No telemetry history': '暫無遙測歷史',
    'No trend data': '暫無趨勢資料',
    'Per-GPU Power': '單一 GPU 功耗',
    'Poll Success': '採集成功率',
    'Poll success and component availability in the selected time window.':
      '所選時間視窗內的採集成功率與元件可用性。',
    'Power draw for each GPU device over time.':
      '每個 GPU 裝置功耗隨時間的變化。',
    'Relative range': '相對時間範圍',
    'Request Pressure': '請求壓力',
    'Requests currently running in the inference engine.':
      '推理引擎中目前正在執行的請求。',
    'Requests waiting for inference engine capacity.':
      '等待推理引擎可用容量的請求。',
    'Resource Utilization': '資源使用率',
    'Running and waiting request counts over time.':
      '執行中與等待中請求數隨時間的變化。',
    'Select chart time window': '選擇圖表時間視窗',
    'Telemetry Availability': '遙測可用性',
    'The selected range exceeds the {{days}}-day retention.':
      '所選範圍超過 {{days}} 天的資料保留期限。',
    'The start time must be earlier than the end time.':
      '開始時間必須早於結束時間。',
    'This time window applies to every chart on the current page.':
      '此時間視窗會套用到目前頁面的所有圖表。',
    'This time window applies only to the expanded chart.':
      '此時間視窗僅套用到目前放大的圖表。',
    'Tokens currently held by active engine requests.':
      '目前活動引擎請求占用的 Token 數。',
    'Total power reported across all GPU boards.':
      '所有 GPU 板卡回報的總功耗。',
    '{{count}} samples · {{seconds}}s buckets':
      '{{count}} 個採樣 · {{seconds}} 秒分桶',
  },
  fr: {
    'All charts share this time window. Relative windows refresh automatically.':
      'Tous les graphiques partagent cette fenêtre temporelle. Les fenêtres relatives sont actualisées automatiquement.',
    'Apply custom range': 'Appliquer la plage personnalisée',
    'Chart time window': 'Fenêtre temporelle des graphiques',
    'CPU, memory, and cache utilization over time.':
      "Évolution de l'utilisation du processeur, de la mémoire et du cache.",
    'Custom range': 'Plage personnalisée',
    'Expand {{chart}}': 'Agrandir {{chart}}',
    'Failed to load telemetry trends':
      'Échec du chargement des tendances de télémétrie',
    'Generated token throughput over time.':
      'Évolution du débit de tokens générés.',
    'Historical Trends': 'Tendances historiques',
    'Host CPU utilization over time.':
      "Évolution de l'utilisation du processeur hôte.",
    'Host memory utilization over time.':
      "Évolution de l'utilisation de la mémoire hôte.",
    'Inference engine cache utilization over time.':
      "Évolution de l'utilisation du cache du moteur d'inférence.",
    'No samples were stored in the selected time window.':
      "Aucun échantillon n'a été stocké dans la fenêtre sélectionnée.",
    'No successful samples contain this metric in the time window.':
      'Aucun échantillon réussi ne contient cette métrique dans la fenêtre sélectionnée.',
    'No telemetry history': 'Aucun historique de télémétrie',
    'No trend data': 'Aucune donnée de tendance',
    'Per-GPU Power': 'Puissance par GPU',
    'Poll Success': 'Collecte réussie',
    'Poll success and component availability in the selected time window.':
      'Réussite des collectes et disponibilité des composants dans la fenêtre sélectionnée.',
    'Power draw for each GPU device over time.':
      'Évolution de la puissance consommée par chaque GPU.',
    'Relative range': 'Plage relative',
    'Request Pressure': 'Pression des requêtes',
    'Requests currently running in the inference engine.':
      "Requêtes actuellement exécutées dans le moteur d'inférence.",
    'Requests waiting for inference engine capacity.':
      "Requêtes en attente de capacité du moteur d'inférence.",
    'Resource Utilization': 'Utilisation des ressources',
    'Running and waiting request counts over time.':
      'Évolution du nombre de requêtes actives et en attente.',
    'Select chart time window': 'Sélectionner la fenêtre temporelle',
    'Telemetry Availability': 'Disponibilité de la télémétrie',
    'The selected range exceeds the {{days}}-day retention.':
      'La plage sélectionnée dépasse la rétention de {{days}} jours.',
    'The start time must be earlier than the end time.':
      "L'heure de début doit précéder l'heure de fin.",
    'This time window applies to every chart on the current page.':
      "Cette fenêtre temporelle s'applique à tous les graphiques de la page.",
    'This time window applies only to the expanded chart.':
      'Cette fenêtre temporelle ne s’applique qu’au graphique agrandi.',
    'Tokens currently held by active engine requests.':
      'Tokens actuellement utilisés par les requêtes actives.',
    'Total power reported across all GPU boards.':
      'Puissance totale signalée par toutes les cartes GPU.',
    '{{count}} samples · {{seconds}}s buckets':
      '{{count}} échantillons · intervalles de {{seconds}} s',
  },
  ja: {
    'All charts share this time window. Relative windows refresh automatically.':
      'すべてのグラフでこの時間範囲を共有します。相対範囲は自動更新されます。',
    'Apply custom range': 'カスタム範囲を適用',
    'Chart time window': 'グラフの時間範囲',
    'CPU, memory, and cache utilization over time.':
      'CPU、メモリ、キャッシュ使用率の推移。',
    'Custom range': 'カスタム範囲',
    'Expand {{chart}}': '{{chart}}を拡大',
    'Failed to load telemetry trends': 'テレメトリ傾向の読み込みに失敗しました',
    'Generated token throughput over time.': '生成 Token スループットの推移。',
    'Historical Trends': '履歴トレンド',
    'Host CPU utilization over time.': 'ホスト CPU 使用率の推移。',
    'Host memory utilization over time.': 'ホストメモリ使用率の推移。',
    'Inference engine cache utilization over time.':
      '推論エンジンのキャッシュ使用率の推移。',
    'No samples were stored in the selected time window.':
      '選択した時間範囲に保存済みサンプルはありません。',
    'No successful samples contain this metric in the time window.':
      'この時間範囲の成功サンプルには、このメトリクスが含まれていません。',
    'No telemetry history': 'テレメトリ履歴がありません',
    'No trend data': 'トレンドデータがありません',
    'Per-GPU Power': 'GPU ごとの電力',
    'Poll Success': '収集成功率',
    'Poll success and component availability in the selected time window.':
      '選択した時間範囲の収集成功率とコンポーネント可用性。',
    'Power draw for each GPU device over time.': '各 GPU の消費電力の推移。',
    'Relative range': '相対時間範囲',
    'Request Pressure': 'リクエスト負荷',
    'Requests currently running in the inference engine.':
      '推論エンジンで現在実行中のリクエスト。',
    'Requests waiting for inference engine capacity.':
      '推論エンジンの空き容量を待つリクエスト。',
    'Resource Utilization': 'リソース使用率',
    'Running and waiting request counts over time.':
      '実行中および待機中リクエスト数の推移。',
    'Select chart time window': 'グラフの時間範囲を選択',
    'Telemetry Availability': 'テレメトリ可用性',
    'The selected range exceeds the {{days}}-day retention.':
      '選択した範囲は {{days}} 日の保存期間を超えています。',
    'The start time must be earlier than the end time.':
      '開始時刻は終了時刻より前にしてください。',
    'This time window applies to every chart on the current page.':
      'この時間範囲は現在のページのすべてのグラフに適用されます。',
    'This time window applies only to the expanded chart.':
      'この時間範囲は拡大表示中のグラフだけに適用されます。',
    'Tokens currently held by active engine requests.':
      'アクティブなエンジンリクエストが現在使用中の Token 数。',
    'Total power reported across all GPU boards.':
      'すべての GPU ボードから報告された合計電力。',
    '{{count}} samples · {{seconds}}s buckets':
      '{{count}} サンプル · {{seconds}} 秒バケット',
  },
  ru: {
    'All charts share this time window. Relative windows refresh automatically.':
      'Все графики используют этот временной интервал. Относительные интервалы обновляются автоматически.',
    'Apply custom range': 'Применить свой диапазон',
    'Chart time window': 'Временной интервал графиков',
    'CPU, memory, and cache utilization over time.':
      'Изменение загрузки CPU, памяти и кэша.',
    'Custom range': 'Свой диапазон',
    'Expand {{chart}}': 'Развернуть «{{chart}}»',
    'Failed to load telemetry trends': 'Не удалось загрузить тренды телеметрии',
    'Generated token throughput over time.':
      'Изменение пропускной способности генерации токенов.',
    'Historical Trends': 'Исторические тренды',
    'Host CPU utilization over time.': 'Изменение загрузки CPU хоста.',
    'Host memory utilization over time.': 'Изменение загрузки памяти хоста.',
    'Inference engine cache utilization over time.':
      'Изменение использования кэша движка вывода.',
    'No samples were stored in the selected time window.':
      'В выбранном интервале нет сохранённых образцов.',
    'No successful samples contain this metric in the time window.':
      'Успешные образцы в этом интервале не содержат данную метрику.',
    'No telemetry history': 'Нет истории телеметрии',
    'No trend data': 'Нет данных тренда',
    'Per-GPU Power': 'Мощность отдельных GPU',
    'Poll Success': 'Успешность сбора',
    'Poll success and component availability in the selected time window.':
      'Успешность сбора и доступность компонентов в выбранном интервале.',
    'Power draw for each GPU device over time.':
      'Изменение мощности каждого устройства GPU.',
    'Relative range': 'Относительный диапазон',
    'Request Pressure': 'Нагрузка запросами',
    'Requests currently running in the inference engine.':
      'Запросы, выполняемые сейчас в движке вывода.',
    'Requests waiting for inference engine capacity.':
      'Запросы, ожидающие доступной мощности движка вывода.',
    'Resource Utilization': 'Использование ресурсов',
    'Running and waiting request counts over time.':
      'Изменение числа выполняемых и ожидающих запросов.',
    'Select chart time window': 'Выбрать временной интервал графиков',
    'Telemetry Availability': 'Доступность телеметрии',
    'The selected range exceeds the {{days}}-day retention.':
      'Выбранный диапазон превышает срок хранения {{days}} дн.',
    'The start time must be earlier than the end time.':
      'Время начала должно быть раньше времени окончания.',
    'This time window applies to every chart on the current page.':
      'Этот временной интервал применяется ко всем графикам страницы.',
    'This time window applies only to the expanded chart.':
      'Этот временной интервал применяется только к развёрнутому графику.',
    'Tokens currently held by active engine requests.':
      'Токены, занятые активными запросами движка.',
    'Total power reported across all GPU boards.':
      'Суммарная мощность всех плат GPU.',
    '{{count}} samples · {{seconds}}s buckets':
      '{{count}} образцов · интервалы по {{seconds}} с',
  },
  vi: {
    'All charts share this time window. Relative windows refresh automatically.':
      'Tất cả biểu đồ dùng chung khoảng thời gian này. Khoảng tương đối sẽ tự động làm mới.',
    'Apply custom range': 'Áp dụng khoảng tùy chỉnh',
    'Chart time window': 'Khoảng thời gian biểu đồ',
    'CPU, memory, and cache utilization over time.':
      'Mức sử dụng CPU, bộ nhớ và bộ nhớ đệm theo thời gian.',
    'Custom range': 'Khoảng tùy chỉnh',
    'Expand {{chart}}': 'Phóng to {{chart}}',
    'Failed to load telemetry trends': 'Không thể tải xu hướng đo từ xa',
    'Generated token throughput over time.':
      'Thông lượng token được tạo theo thời gian.',
    'Historical Trends': 'Xu hướng lịch sử',
    'Host CPU utilization over time.':
      'Mức sử dụng CPU máy chủ theo thời gian.',
    'Host memory utilization over time.':
      'Mức sử dụng bộ nhớ máy chủ theo thời gian.',
    'Inference engine cache utilization over time.':
      'Mức sử dụng bộ nhớ đệm của engine suy luận theo thời gian.',
    'No samples were stored in the selected time window.':
      'Không có mẫu nào được lưu trong khoảng thời gian đã chọn.',
    'No successful samples contain this metric in the time window.':
      'Không có mẫu thành công nào chứa chỉ số này trong khoảng thời gian.',
    'No telemetry history': 'Không có lịch sử đo từ xa',
    'No trend data': 'Không có dữ liệu xu hướng',
    'Per-GPU Power': 'Công suất từng GPU',
    'Poll Success': 'Thu thập thành công',
    'Poll success and component availability in the selected time window.':
      'Tỷ lệ thu thập thành công và tính sẵn sàng của thành phần trong khoảng đã chọn.',
    'Power draw for each GPU device over time.':
      'Công suất tiêu thụ của từng GPU theo thời gian.',
    'Relative range': 'Khoảng tương đối',
    'Request Pressure': 'Áp lực yêu cầu',
    'Requests currently running in the inference engine.':
      'Các yêu cầu đang chạy trong engine suy luận.',
    'Requests waiting for inference engine capacity.':
      'Các yêu cầu đang chờ dung lượng engine suy luận.',
    'Resource Utilization': 'Mức sử dụng tài nguyên',
    'Running and waiting request counts over time.':
      'Số yêu cầu đang chạy và đang chờ theo thời gian.',
    'Select chart time window': 'Chọn khoảng thời gian biểu đồ',
    'Telemetry Availability': 'Tính sẵn sàng của đo từ xa',
    'The selected range exceeds the {{days}}-day retention.':
      'Khoảng đã chọn vượt quá thời gian lưu giữ {{days}} ngày.',
    'The start time must be earlier than the end time.':
      'Thời gian bắt đầu phải sớm hơn thời gian kết thúc.',
    'This time window applies to every chart on the current page.':
      'Khoảng thời gian này áp dụng cho mọi biểu đồ trên trang hiện tại.',
    'This time window applies only to the expanded chart.':
      'Khoảng thời gian này chỉ áp dụng cho biểu đồ đang phóng to.',
    'Tokens currently held by active engine requests.':
      'Số token hiện được các yêu cầu engine đang hoạt động sử dụng.',
    'Total power reported across all GPU boards.':
      'Tổng công suất được báo cáo trên tất cả bo mạch GPU.',
    '{{count}} samples · {{seconds}}s buckets':
      '{{count}} mẫu · nhóm {{seconds}} giây',
  },
}

const newKeys = {
  en: { ...english, ...phase2English },
  zh: {
    ...phase2Translations.zh,
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
  'zh-TW': { ...english, ...phase2Translations['zh-TW'] },
  fr: { ...english, ...phase2Translations.fr },
  ja: { ...english, ...phase2Translations.ja },
  ru: { ...english, ...phase2Translations.ru },
  vi: { ...english, ...phase2Translations.vi },
}

async function main() {
  let totalAdded = 0

  for (const [locale, trans] of Object.entries(newKeys)) {
    const filePath = path.join(LOCALES_DIR, `${locale}.json`)
    const json = JSON.parse(await fs.readFile(filePath, 'utf8'))

    let count = 0
    for (const [key, value] of Object.entries(trans)) {
      if (!Object.hasOwn(json.translation, key)) {
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
