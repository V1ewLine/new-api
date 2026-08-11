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

const phase3English = {
  'Agent address is blocked': 'Agent address is blocked',
  'Agent is unreachable': 'Agent is unreachable',
  'Agent rejected the Token': 'Agent rejected the Token',
  'Agent request timed out': 'Agent request timed out',
  'Auto-refreshes every {{seconds}} seconds.':
    'Auto-refreshes every {{seconds}} seconds.',
  'Check the Agent address, port, firewall, and service status, then retry.':
    'Check the Agent address, port, firewall, and service status, then retry.',
  'Cluster credential is unavailable': 'Cluster credential is unavailable',
  'Collect Now': 'Collect Now',
  'Collecting...': 'Collecting...',
  'Data freshness': 'Data freshness',
  'Displayed metrics come from the last successful sample at {{time}}. Last collection attempt: {{attempt}}.':
    'Displayed metrics come from the last successful sample at {{time}}. Last collection attempt: {{attempt}}.',
  'Failed {{count}} times in a row': 'Failed {{count}} times in a row',
  Fresh: 'Fresh',
  'Generate a new Agent Token and update the remote Agent configuration.':
    'Generate a new Agent Token and update the remote Agent configuration.',
  'Generate a new Agent Token, update the remote Agent, and test the connection again.':
    'Generate a new Agent Token, update the remote Agent, and test the connection again.',
  'Last attempt: {{time}}': 'Last attempt: {{time}}',
  'Last success: {{time}}': 'Last success: {{time}}',
  'Last Successful Sample': 'Last Successful Sample',
  'Latest telemetry poll failed': 'Latest telemetry poll failed',
  'New API did not receive a response in time. Check Agent load and network latency, then retry.':
    'New API did not receive a response in time. Check Agent load and network latency, then retry.',
  'No successful sample': 'No successful sample',
  'Open cluster details to review the failure, or retry the collection now.':
    'Open cluster details to review the failure, or retry the collection now.',
  'Retry now': 'Retry now',
  'Retrying...': 'Retrying...',
  Stale: 'Stale',
  'Telemetry data may be stale': 'Telemetry data may be stale',
  'The Agent response is incompatible with this New API version. Check the Agent version and schema.':
    'The Agent response is incompatible with this New API version. Check the Agent version and schema.',
  'The outbound request policy blocks this address. Review the cluster Agent allowlist settings.':
    'The outbound request policy blocks this address. Review the cluster Agent allowlist settings.',
  'Unsupported telemetry schema': 'Unsupported telemetry schema',
}

const phase3Translations = {
  zh: {
    'Agent address is blocked': 'Agent 地址已被拦截',
    'Agent is unreachable': '无法访问 Agent',
    'Agent rejected the Token': 'Agent 拒绝了 Token',
    'Agent request timed out': 'Agent 请求超时',
    'Auto-refreshes every {{seconds}} seconds.':
      '每 {{seconds}} 秒自动刷新一次。',
    'Check the Agent address, port, firewall, and service status, then retry.':
      '请检查 Agent 地址、端口、防火墙和服务状态，然后重试。',
    'Cluster credential is unavailable': '集群凭据不可用',
    'Collect Now': '立即采集',
    'Collecting...': '正在采集...',
    'Data freshness': '数据新鲜度',
    'Displayed metrics come from the last successful sample at {{time}}. Last collection attempt: {{attempt}}.':
      '当前指标来自 {{time}} 的最后一次成功采样，最近一次采集尝试时间为 {{attempt}}。',
    'Failed {{count}} times in a row': '已连续失败 {{count}} 次',
    Fresh: '新鲜',
    'Generate a new Agent Token and update the remote Agent configuration.':
      '请生成新的 Agent Token，并更新远端 Agent 配置。',
    'Generate a new Agent Token, update the remote Agent, and test the connection again.':
      '请生成新的 Agent Token，更新远端 Agent，然后重新测试连接。',
    'Last attempt: {{time}}': '最近尝试：{{time}}',
    'Last success: {{time}}': '最近成功：{{time}}',
    'Last Successful Sample': '最近成功采样',
    'Latest telemetry poll failed': '最近一次遥测采集失败',
    'New API did not receive a response in time. Check Agent load and network latency, then retry.':
      'New API 未能及时收到响应。请检查 Agent 负载和网络延迟，然后重试。',
    'No successful sample': '暂无成功采样',
    'Open cluster details to review the failure, or retry the collection now.':
      '请打开集群详情查看故障，或立即重试采集。',
    'Retry now': '立即重试',
    'Retrying...': '正在重试...',
    Stale: '已过期',
    'Telemetry data may be stale': '遥测数据可能已过期',
    'The Agent response is incompatible with this New API version. Check the Agent version and schema.':
      'Agent 响应与当前 New API 版本不兼容。请检查 Agent 版本和数据格式。',
    'The outbound request policy blocks this address. Review the cluster Agent allowlist settings.':
      '出站请求策略拦截了此地址。请检查集群 Agent 地址白名单设置。',
    'Unsupported telemetry schema': '不支持的遥测数据格式',
  },
  'zh-TW': {
    'Agent address is blocked': 'Agent 位址已被攔截',
    'Agent is unreachable': '無法連線至 Agent',
    'Agent rejected the Token': 'Agent 拒絕了 Token',
    'Agent request timed out': 'Agent 請求逾時',
    'Auto-refreshes every {{seconds}} seconds.':
      '每 {{seconds}} 秒自動重新整理一次。',
    'Check the Agent address, port, firewall, and service status, then retry.':
      '請檢查 Agent 位址、連接埠、防火牆與服務狀態，然後重試。',
    'Cluster credential is unavailable': '叢集憑證不可用',
    'Collect Now': '立即採集',
    'Collecting...': '正在採集...',
    'Data freshness': '資料新鮮度',
    'Displayed metrics come from the last successful sample at {{time}}. Last collection attempt: {{attempt}}.':
      '目前指標來自 {{time}} 的最後一次成功採樣，最近一次採集嘗試時間為 {{attempt}}。',
    'Failed {{count}} times in a row': '已連續失敗 {{count}} 次',
    Fresh: '最新',
    'Generate a new Agent Token and update the remote Agent configuration.':
      '請產生新的 Agent Token，並更新遠端 Agent 設定。',
    'Generate a new Agent Token, update the remote Agent, and test the connection again.':
      '請產生新的 Agent Token，更新遠端 Agent，然後重新測試連線。',
    'Last attempt: {{time}}': '最近嘗試：{{time}}',
    'Last success: {{time}}': '最近成功：{{time}}',
    'Last Successful Sample': '最近成功採樣',
    'Latest telemetry poll failed': '最近一次遙測採集失敗',
    'New API did not receive a response in time. Check Agent load and network latency, then retry.':
      'New API 未能及時收到回應。請檢查 Agent 負載與網路延遲，然後重試。',
    'No successful sample': '暫無成功採樣',
    'Open cluster details to review the failure, or retry the collection now.':
      '請開啟叢集詳情檢查故障，或立即重試採集。',
    'Retry now': '立即重試',
    'Retrying...': '正在重試...',
    Stale: '已過期',
    'Telemetry data may be stale': '遙測資料可能已過期',
    'The Agent response is incompatible with this New API version. Check the Agent version and schema.':
      'Agent 回應與目前 New API 版本不相容。請檢查 Agent 版本與資料格式。',
    'The outbound request policy blocks this address. Review the cluster Agent allowlist settings.':
      '外送請求策略攔截了此位址。請檢查叢集 Agent 位址允許清單設定。',
    'Unsupported telemetry schema': '不支援的遙測資料格式',
  },
  fr: {
    'Agent address is blocked': "L'adresse de l'Agent est bloquée",
    'Agent is unreachable': "L'Agent est inaccessible",
    'Agent rejected the Token': "L'Agent a rejeté le Token",
    'Agent request timed out': "La requête vers l'Agent a expiré",
    'Auto-refreshes every {{seconds}} seconds.':
      'Actualisation automatique toutes les {{seconds}} secondes.',
    'Check the Agent address, port, firewall, and service status, then retry.':
      "Vérifiez l'adresse, le port, le pare-feu et l'état du service Agent, puis réessayez.",
    'Cluster credential is unavailable':
      "L'identifiant du cluster est indisponible",
    'Collect Now': 'Collecter maintenant',
    'Collecting...': 'Collecte en cours...',
    'Data freshness': 'Fraîcheur des données',
    'Displayed metrics come from the last successful sample at {{time}}. Last collection attempt: {{attempt}}.':
      'Les métriques affichées proviennent du dernier échantillon réussi à {{time}}. Dernière tentative : {{attempt}}.',
    'Failed {{count}} times in a row': '{{count}} échecs consécutifs',
    Fresh: 'À jour',
    'Generate a new Agent Token and update the remote Agent configuration.':
      "Générez un nouveau Token Agent et mettez à jour la configuration de l'Agent distant.",
    'Generate a new Agent Token, update the remote Agent, and test the connection again.':
      "Générez un nouveau Token Agent, mettez à jour l'Agent distant, puis retestez la connexion.",
    'Last attempt: {{time}}': 'Dernière tentative : {{time}}',
    'Last success: {{time}}': 'Dernier succès : {{time}}',
    'Last Successful Sample': 'Dernier échantillon réussi',
    'Latest telemetry poll failed':
      'La dernière collecte de télémétrie a échoué',
    'New API did not receive a response in time. Check Agent load and network latency, then retry.':
      "New API n'a pas reçu de réponse à temps. Vérifiez la charge de l'Agent et la latence réseau, puis réessayez.",
    'No successful sample': 'Aucun échantillon réussi',
    'Open cluster details to review the failure, or retry the collection now.':
      'Ouvrez les détails du cluster pour examiner la panne ou relancez la collecte maintenant.',
    'Retry now': 'Réessayer maintenant',
    'Retrying...': 'Nouvelle tentative...',
    Stale: 'Obsolète',
    'Telemetry data may be stale':
      'Les données de télémétrie peuvent être obsolètes',
    'The Agent response is incompatible with this New API version. Check the Agent version and schema.':
      "La réponse de l'Agent est incompatible avec cette version de New API. Vérifiez la version et le schéma de l'Agent.",
    'The outbound request policy blocks this address. Review the cluster Agent allowlist settings.':
      "La stratégie des requêtes sortantes bloque cette adresse. Vérifiez la liste d'autorisation des Agents de cluster.",
    'Unsupported telemetry schema': 'Schéma de télémétrie non pris en charge',
  },
  ja: {
    'Agent address is blocked': 'Agent アドレスがブロックされています',
    'Agent is unreachable': 'Agent に到達できません',
    'Agent rejected the Token': 'Agent が Token を拒否しました',
    'Agent request timed out': 'Agent リクエストがタイムアウトしました',
    'Auto-refreshes every {{seconds}} seconds.':
      '{{seconds}} 秒ごとに自動更新します。',
    'Check the Agent address, port, firewall, and service status, then retry.':
      'Agent のアドレス、ポート、ファイアウォール、サービス状態を確認してから再試行してください。',
    'Cluster credential is unavailable': 'クラスター認証情報を利用できません',
    'Collect Now': '今すぐ収集',
    'Collecting...': '収集中...',
    'Data freshness': 'データの鮮度',
    'Displayed metrics come from the last successful sample at {{time}}. Last collection attempt: {{attempt}}.':
      '表示中の指標は {{time}} の最新成功サンプルです。直近の収集試行：{{attempt}}。',
    'Failed {{count}} times in a row': '{{count}} 回連続で失敗',
    Fresh: '最新',
    'Generate a new Agent Token and update the remote Agent configuration.':
      '新しい Agent Token を生成し、リモート Agent の設定を更新してください。',
    'Generate a new Agent Token, update the remote Agent, and test the connection again.':
      '新しい Agent Token を生成してリモート Agent を更新し、接続を再テストしてください。',
    'Last attempt: {{time}}': '直近の試行：{{time}}',
    'Last success: {{time}}': '直近の成功：{{time}}',
    'Last Successful Sample': '最新成功サンプル',
    'Latest telemetry poll failed': '最新のテレメトリ収集に失敗しました',
    'New API did not receive a response in time. Check Agent load and network latency, then retry.':
      'New API が時間内に応答を受信できませんでした。Agent の負荷とネットワーク遅延を確認して再試行してください。',
    'No successful sample': '成功サンプルなし',
    'Open cluster details to review the failure, or retry the collection now.':
      'クラスター詳細で障害を確認するか、今すぐ収集を再試行してください。',
    'Retry now': '今すぐ再試行',
    'Retrying...': '再試行中...',
    Stale: '期限切れ',
    'Telemetry data may be stale': 'テレメトリデータが古い可能性があります',
    'The Agent response is incompatible with this New API version. Check the Agent version and schema.':
      'Agent の応答はこの New API バージョンと互換性がありません。Agent のバージョンとスキーマを確認してください。',
    'The outbound request policy blocks this address. Review the cluster Agent allowlist settings.':
      '送信リクエストポリシーがこのアドレスをブロックしています。クラスター Agent の許可リスト設定を確認してください。',
    'Unsupported telemetry schema': '未対応のテレメトリスキーマ',
  },
  ru: {
    'Agent address is blocked': 'Адрес Agent заблокирован',
    'Agent is unreachable': 'Agent недоступен',
    'Agent rejected the Token': 'Agent отклонил Token',
    'Agent request timed out': 'Истекло время ожидания запроса к Agent',
    'Auto-refreshes every {{seconds}} seconds.':
      'Автообновление каждые {{seconds}} сек.',
    'Check the Agent address, port, firewall, and service status, then retry.':
      'Проверьте адрес и порт Agent, межсетевой экран и состояние сервиса, затем повторите попытку.',
    'Cluster credential is unavailable': 'Учетные данные кластера недоступны',
    'Collect Now': 'Собрать сейчас',
    'Collecting...': 'Идет сбор...',
    'Data freshness': 'Актуальность данных',
    'Displayed metrics come from the last successful sample at {{time}}. Last collection attempt: {{attempt}}.':
      'Показаны метрики из последней успешной выборки в {{time}}. Последняя попытка сбора: {{attempt}}.',
    'Failed {{count}} times in a row': 'Последовательных сбоев: {{count}}',
    Fresh: 'Актуально',
    'Generate a new Agent Token and update the remote Agent configuration.':
      'Создайте новый Agent Token и обновите конфигурацию удаленного Agent.',
    'Generate a new Agent Token, update the remote Agent, and test the connection again.':
      'Создайте новый Agent Token, обновите удаленный Agent и снова проверьте подключение.',
    'Last attempt: {{time}}': 'Последняя попытка: {{time}}',
    'Last success: {{time}}': 'Последний успех: {{time}}',
    'Last Successful Sample': 'Последняя успешная выборка',
    'Latest telemetry poll failed':
      'Последний сбор телеметрии завершился сбоем',
    'New API did not receive a response in time. Check Agent load and network latency, then retry.':
      'New API не получил ответ вовремя. Проверьте нагрузку Agent и задержку сети, затем повторите попытку.',
    'No successful sample': 'Нет успешных выборок',
    'Open cluster details to review the failure, or retry the collection now.':
      'Откройте сведения о кластере для проверки сбоя или повторите сбор сейчас.',
    'Retry now': 'Повторить сейчас',
    'Retrying...': 'Повторная попытка...',
    Stale: 'Устарело',
    'Telemetry data may be stale': 'Данные телеметрии могут быть устаревшими',
    'The Agent response is incompatible with this New API version. Check the Agent version and schema.':
      'Ответ Agent несовместим с этой версией New API. Проверьте версию и схему Agent.',
    'The outbound request policy blocks this address. Review the cluster Agent allowlist settings.':
      'Политика исходящих запросов блокирует этот адрес. Проверьте список разрешенных адресов Agent кластера.',
    'Unsupported telemetry schema': 'Неподдерживаемая схема телеметрии',
  },
  vi: {
    'Agent address is blocked': 'Địa chỉ Agent bị chặn',
    'Agent is unreachable': 'Không thể kết nối Agent',
    'Agent rejected the Token': 'Agent đã từ chối Token',
    'Agent request timed out': 'Yêu cầu Agent đã hết thời gian chờ',
    'Auto-refreshes every {{seconds}} seconds.':
      'Tự động làm mới mỗi {{seconds}} giây.',
    'Check the Agent address, port, firewall, and service status, then retry.':
      'Kiểm tra địa chỉ, cổng, tường lửa và trạng thái dịch vụ Agent, sau đó thử lại.',
    'Cluster credential is unavailable':
      'Thông tin xác thực cụm không khả dụng',
    'Collect Now': 'Thu thập ngay',
    'Collecting...': 'Đang thu thập...',
    'Data freshness': 'Độ mới của dữ liệu',
    'Displayed metrics come from the last successful sample at {{time}}. Last collection attempt: {{attempt}}.':
      'Các chỉ số hiển thị đến từ mẫu thành công gần nhất lúc {{time}}. Lần thu thập gần nhất: {{attempt}}.',
    'Failed {{count}} times in a row': 'Đã lỗi liên tiếp {{count}} lần',
    Fresh: 'Mới',
    'Generate a new Agent Token and update the remote Agent configuration.':
      'Tạo Agent Token mới và cập nhật cấu hình Agent từ xa.',
    'Generate a new Agent Token, update the remote Agent, and test the connection again.':
      'Tạo Agent Token mới, cập nhật Agent từ xa và kiểm tra lại kết nối.',
    'Last attempt: {{time}}': 'Lần thử gần nhất: {{time}}',
    'Last success: {{time}}': 'Thành công gần nhất: {{time}}',
    'Last Successful Sample': 'Mẫu thành công gần nhất',
    'Latest telemetry poll failed': 'Lần thu thập đo từ xa gần nhất thất bại',
    'New API did not receive a response in time. Check Agent load and network latency, then retry.':
      'New API không nhận được phản hồi kịp thời. Kiểm tra tải Agent và độ trễ mạng, sau đó thử lại.',
    'No successful sample': 'Chưa có mẫu thành công',
    'Open cluster details to review the failure, or retry the collection now.':
      'Mở chi tiết cụm để xem lỗi hoặc thử thu thập lại ngay.',
    'Retry now': 'Thử lại ngay',
    'Retrying...': 'Đang thử lại...',
    Stale: 'Đã cũ',
    'Telemetry data may be stale': 'Dữ liệu đo từ xa có thể đã cũ',
    'The Agent response is incompatible with this New API version. Check the Agent version and schema.':
      'Phản hồi của Agent không tương thích với phiên bản New API này. Kiểm tra phiên bản và schema của Agent.',
    'The outbound request policy blocks this address. Review the cluster Agent allowlist settings.':
      'Chính sách yêu cầu đi ra chặn địa chỉ này. Kiểm tra danh sách cho phép Agent của cụm.',
    'Unsupported telemetry schema': 'Schema đo từ xa không được hỗ trợ',
  },
}

const currentLoadEnglish = {
  'Current Load Trends': 'Current Load Trends',
  'Current Model Load Trends': 'Current Model Load Trends',
  'Current Requests': 'Current Requests',
  'Current Requests Trend': 'Current Requests Trend',
  'Current Token Usage': 'Current Token Usage',
  'Current Token Usage Trend': 'Current Token Usage Trend',
  'Current request and token load for this model over the selected time window.':
    'Current request and token load for this model over the selected time window.',
  'Global current request and token load over the selected time window.':
    'Global current request and token load over the selected time window.',
  'Search and filters apply only to the model list below; summary metrics and trends remain global.':
    'Search and filters apply only to the model list below; summary metrics and trends remain global.',
  'Sum of current token usage across reporting clusters.':
    'Sum of current token usage across reporting clusters.',
  'Sum of running and waiting requests across reporting clusters.':
    'Sum of running and waiting requests across reporting clusters.',
  'Trend points sum only valid samples in each bucket; missing data is shown as a gap.':
    'Trend points sum only valid samples in each bucket; missing data is shown as a gap.',
  '{{reporting}}/{{monitored}} monitored clusters reporting':
    '{{reporting}}/{{monitored}} monitored clusters reporting',
}

const currentLoadTranslations = {
  zh: {
    'Current Load Trends': '当前负载趋势',
    'Current Model Load Trends': '当前模型负载趋势',
    'Current Requests': '当前请求数',
    'Current Requests Trend': '当前请求数趋势',
    'Current Token Usage': '当前 Token 占用',
    'Current Token Usage Trend': '当前 Token 占用趋势',
    'Current request and token load for this model over the selected time window.':
      '所选时间窗口内该模型的当前请求与 Token 负载。',
    'Global current request and token load over the selected time window.':
      '所选时间窗口内所有模型和集群的当前请求与 Token 负载。',
    'Search and filters apply only to the model list below; summary metrics and trends remain global.':
      '搜索和筛选仅作用于下方模型列表；汇总指标和趋势始终保持全局口径。',
    'Sum of current token usage across reporting clusters.':
      '汇总有数据集群的当前 Token 占用。',
    'Sum of running and waiting requests across reporting clusters.':
      '汇总有数据集群的运行中与等待中请求。',
    'Trend points sum only valid samples in each bucket; missing data is shown as a gap.':
      '每个时间桶仅汇总有效采样；缺失数据以曲线断点显示。',
    '{{reporting}}/{{monitored}} monitored clusters reporting':
      '{{reporting}}/{{monitored}} 个监控中集群有数据',
  },
  'zh-TW': {
    'Current Load Trends': '目前負載趨勢',
    'Current Model Load Trends': '目前模型負載趨勢',
    'Current Requests': '目前請求數',
    'Current Requests Trend': '目前請求數趨勢',
    'Current Token Usage': '目前 Token 佔用',
    'Current Token Usage Trend': '目前 Token 佔用趨勢',
    'Current request and token load for this model over the selected time window.':
      '所選時間範圍內此模型目前的請求與 Token 負載。',
    'Global current request and token load over the selected time window.':
      '所選時間範圍內所有模型與叢集目前的請求與 Token 負載。',
    'Search and filters apply only to the model list below; summary metrics and trends remain global.':
      '搜尋與篩選僅套用至下方模型清單；摘要指標與趨勢維持全域範圍。',
    'Sum of current token usage across reporting clusters.':
      '加總有資料叢集目前的 Token 佔用。',
    'Sum of running and waiting requests across reporting clusters.':
      '加總有資料叢集執行中與等待中的請求。',
    'Trend points sum only valid samples in each bucket; missing data is shown as a gap.':
      '每個時間桶只加總有效採樣；缺少資料時以曲線斷點顯示。',
    '{{reporting}}/{{monitored}} monitored clusters reporting':
      '{{reporting}}/{{monitored}} 個監控中叢集有資料',
  },
  fr: {
    'Current Load Trends': 'Tendances de charge actuelle',
    'Current Model Load Trends': 'Tendances de charge actuelle du modèle',
    'Current Requests': 'Requêtes actuelles',
    'Current Requests Trend': 'Tendance des requêtes actuelles',
    'Current Token Usage': 'Utilisation actuelle des jetons',
    'Current Token Usage Trend':
      "Tendance de l'utilisation actuelle des jetons",
    'Current request and token load for this model over the selected time window.':
      'Charge actuelle des requêtes et des jetons de ce modèle sur la période sélectionnée.',
    'Global current request and token load over the selected time window.':
      'Charge globale actuelle des requêtes et des jetons sur la période sélectionnée.',
    'Search and filters apply only to the model list below; summary metrics and trends remain global.':
      'La recherche et les filtres ne concernent que la liste des modèles ci-dessous ; les indicateurs et tendances restent globaux.',
    'Sum of current token usage across reporting clusters.':
      "Somme de l'utilisation actuelle des jetons des clusters transmettant des données.",
    'Sum of running and waiting requests across reporting clusters.':
      'Somme des requêtes en cours et en attente des clusters transmettant des données.',
    'Trend points sum only valid samples in each bucket; missing data is shown as a gap.':
      'Chaque point additionne uniquement les échantillons valides ; les données manquantes apparaissent comme une interruption.',
    '{{reporting}}/{{monitored}} monitored clusters reporting':
      '{{reporting}}/{{monitored}} clusters surveillés transmettent des données',
  },
  ja: {
    'Current Load Trends': '現在の負荷トレンド',
    'Current Model Load Trends': '現在のモデル負荷トレンド',
    'Current Requests': '現在のリクエスト数',
    'Current Requests Trend': '現在のリクエスト数の推移',
    'Current Token Usage': '現在のトークン使用量',
    'Current Token Usage Trend': '現在のトークン使用量の推移',
    'Current request and token load for this model over the selected time window.':
      '選択した時間範囲における、このモデルの現在のリクエストとトークン負荷です。',
    'Global current request and token load over the selected time window.':
      '選択した時間範囲における、全体の現在のリクエストとトークン負荷です。',
    'Search and filters apply only to the model list below; summary metrics and trends remain global.':
      '検索とフィルターは下のモデル一覧だけに適用され、概要指標とトレンドは全体値のままです。',
    'Sum of current token usage across reporting clusters.':
      'データを報告しているクラスターの現在のトークン使用量の合計です。',
    'Sum of running and waiting requests across reporting clusters.':
      'データを報告しているクラスターの実行中および待機中リクエストの合計です。',
    'Trend points sum only valid samples in each bucket; missing data is shown as a gap.':
      '各時間バケットでは有効なサンプルのみを合計し、欠損データは線の切れ目で表示します。',
    '{{reporting}}/{{monitored}} monitored clusters reporting':
      '{{reporting}}/{{monitored}} 個の監視中クラスターが報告中',
  },
  ru: {
    'Current Load Trends': 'Тренды текущей нагрузки',
    'Current Model Load Trends': 'Тренды текущей нагрузки модели',
    'Current Requests': 'Текущие запросы',
    'Current Requests Trend': 'Тренд текущих запросов',
    'Current Token Usage': 'Текущее использование токенов',
    'Current Token Usage Trend': 'Тренд текущего использования токенов',
    'Current request and token load for this model over the selected time window.':
      'Текущая нагрузка запросов и токенов этой модели за выбранный период.',
    'Global current request and token load over the selected time window.':
      'Общая текущая нагрузка запросов и токенов за выбранный период.',
    'Search and filters apply only to the model list below; summary metrics and trends remain global.':
      'Поиск и фильтры применяются только к списку моделей ниже; сводные показатели и тренды остаются общими.',
    'Sum of current token usage across reporting clusters.':
      'Сумма текущего использования токенов по кластерам, передающим данные.',
    'Sum of running and waiting requests across reporting clusters.':
      'Сумма выполняющихся и ожидающих запросов по кластерам, передающим данные.',
    'Trend points sum only valid samples in each bucket; missing data is shown as a gap.':
      'В каждом временном интервале суммируются только корректные выборки; пропуски данных отображаются разрывами.',
    '{{reporting}}/{{monitored}} monitored clusters reporting':
      '{{reporting}} из {{monitored}} отслеживаемых кластеров передают данные',
  },
  vi: {
    'Current Load Trends': 'Xu hướng tải hiện tại',
    'Current Model Load Trends': 'Xu hướng tải hiện tại của mô hình',
    'Current Requests': 'Yêu cầu hiện tại',
    'Current Requests Trend': 'Xu hướng yêu cầu hiện tại',
    'Current Token Usage': 'Mức sử dụng token hiện tại',
    'Current Token Usage Trend': 'Xu hướng sử dụng token hiện tại',
    'Current request and token load for this model over the selected time window.':
      'Tải yêu cầu và token hiện tại của mô hình này trong khoảng thời gian đã chọn.',
    'Global current request and token load over the selected time window.':
      'Tải yêu cầu và token hiện tại trên toàn hệ thống trong khoảng thời gian đã chọn.',
    'Search and filters apply only to the model list below; summary metrics and trends remain global.':
      'Tìm kiếm và bộ lọc chỉ áp dụng cho danh sách mô hình bên dưới; số liệu tổng hợp và xu hướng vẫn ở phạm vi toàn hệ thống.',
    'Sum of current token usage across reporting clusters.':
      'Tổng mức sử dụng token hiện tại của các cụm đang báo cáo.',
    'Sum of running and waiting requests across reporting clusters.':
      'Tổng số yêu cầu đang chạy và đang chờ của các cụm đang báo cáo.',
    'Trend points sum only valid samples in each bucket; missing data is shown as a gap.':
      'Mỗi điểm xu hướng chỉ cộng các mẫu hợp lệ trong khoảng; dữ liệu thiếu được hiển thị bằng đoạn ngắt.',
    '{{reporting}}/{{monitored}} monitored clusters reporting':
      '{{reporting}}/{{monitored}} cụm đang giám sát có dữ liệu',
  },
}

const modelAnalyticsExportEnglish = {
  'All models': 'All models',
  'Effective range: {{start}} to {{end}} ({{timezone}})':
    'Effective range: {{start}} to {{end}} ({{timezone}})',
  'Export CSV': 'Export CSV',
  'Export Model Analytics': 'Export Model Analytics',
  'Export request, token, quota, RPM, and TPM data by model and time bucket.':
    'Export request, token, quota, RPM, and TPM data by model and time bucket.',
  'Export Settings': 'Export Settings',
  'Exporting model analytics...': 'Exporting model analytics...',
  'Failed to export model analytics': 'Failed to export model analytics',
  'Failed to load exportable models': 'Failed to load exportable models',
  'Limits: hourly 90 days, daily 2 years, weekly 5 years; up to 200,000 rows.':
    'Limits: hourly 90 days, daily 2 years, weekly 5 years; up to 200,000 rows.',
  'Model analytics exported': 'Model analytics exported',
  'No model data is available in this time range':
    'No model data is available in this time range',
  'Select a model': 'Select a model',
  'The export uses complete natural time buckets in your browser time zone.':
    'The export uses complete natural time buckets in your browser time zone.',
  'The selected export range is too large for this granularity':
    'The selected export range is too large for this granularity',
  'The selected model has no data in this time range':
    'The selected model has no data in this time range',
  'invalid model analytics export request':
    'Invalid model analytics export request',
  'model analytics export exceeds the allowed row count':
    'Model analytics export exceeds the allowed row count',
  'model analytics export range is too large':
    'Model analytics export range is too large',
  'no model analytics data found': 'No model analytics data found',
}

const modelAnalyticsExportTranslations = {
  zh: {
    'All models': '所有模型',
    'Effective range: {{start}} to {{end}} ({{timezone}})':
      '实际导出范围：{{start}} 至 {{end}}（{{timezone}}）',
    'Export CSV': '导出 CSV',
    'Export Model Analytics': '导出模型调用分析',
    'Export request, token, quota, RPM, and TPM data by model and time bucket.':
      '按模型和时间段导出请求数、Token、额度、RPM 和 TPM 数据。',
    'Export Settings': '导出设置',
    'Exporting model analytics...': '正在导出模型调用分析...',
    'Failed to export model analytics': '模型调用分析导出失败',
    'Failed to load exportable models': '加载可导出模型失败',
    'Limits: hourly 90 days, daily 2 years, weekly 5 years; up to 200,000 rows.':
      '范围限制：小时最多 90 天、天最多 2 年、周最多 5 年；最多导出 200,000 行。',
    'Model analytics exported': '模型调用分析已导出',
    'No model data is available in this time range':
      '该时间范围内没有可导出的模型数据',
    'Select a model': '选择模型',
    'The export uses complete natural time buckets in your browser time zone.':
      '导出将按浏览器时区使用完整的自然时间段。',
    'The selected export range is too large for this granularity':
      '所选导出范围超过当前时间粒度的限制',
    'The selected model has no data in this time range':
      '所选模型在该时间范围内没有数据',
    'invalid model analytics export request': '模型调用分析导出请求无效',
    'model analytics export exceeds the allowed row count':
      '模型调用分析导出超过允许的行数',
    'model analytics export range is too large': '模型调用分析导出范围过大',
    'no model analytics data found': '未找到模型调用分析数据',
  },
  'zh-TW': {
    'All models': '所有模型',
    'Effective range: {{start}} to {{end}} ({{timezone}})':
      '實際匯出範圍：{{start}} 至 {{end}}（{{timezone}}）',
    'Export CSV': '匯出 CSV',
    'Export Model Analytics': '匯出模型呼叫分析',
    'Export request, token, quota, RPM, and TPM data by model and time bucket.':
      '依模型與時間區段匯出請求數、Token、額度、RPM 與 TPM 資料。',
    'Export Settings': '匯出設定',
    'Exporting model analytics...': '正在匯出模型呼叫分析...',
    'Failed to export model analytics': '模型呼叫分析匯出失敗',
    'Failed to load exportable models': '載入可匯出模型失敗',
    'Limits: hourly 90 days, daily 2 years, weekly 5 years; up to 200,000 rows.':
      '範圍限制：小時最多 90 天、天最多 2 年、週最多 5 年；最多匯出 200,000 列。',
    'Model analytics exported': '模型呼叫分析已匯出',
    'No model data is available in this time range':
      '此時間範圍內沒有可匯出的模型資料',
    'Select a model': '選擇模型',
    'The export uses complete natural time buckets in your browser time zone.':
      '匯出會依瀏覽器時區使用完整的自然時間區段。',
    'The selected export range is too large for this granularity':
      '所選匯出範圍超過目前時間粒度的限制',
    'The selected model has no data in this time range':
      '所選模型在此時間範圍內沒有資料',
    'invalid model analytics export request': '模型呼叫分析匯出請求無效',
    'model analytics export exceeds the allowed row count':
      '模型呼叫分析匯出超過允許的列數',
    'model analytics export range is too large': '模型呼叫分析匯出範圍過大',
    'no model analytics data found': '找不到模型呼叫分析資料',
  },
  fr: {
    'All models': 'Tous les modèles',
    'Effective range: {{start}} to {{end}} ({{timezone}})':
      'Plage effective : du {{start}} au {{end}} ({{timezone}})',
    'Export CSV': 'Exporter en CSV',
    'Export Model Analytics': 'Exporter l’analyse des modèles',
    'Export request, token, quota, RPM, and TPM data by model and time bucket.':
      'Exportez les requêtes, jetons, quotas, RPM et TPM par modèle et période.',
    'Export Settings': 'Paramètres d’export',
    'Exporting model analytics...': 'Export de l’analyse des modèles...',
    'Failed to export model analytics':
      'Échec de l’export de l’analyse des modèles',
    'Failed to load exportable models':
      'Échec du chargement des modèles exportables',
    'Limits: hourly 90 days, daily 2 years, weekly 5 years; up to 200,000 rows.':
      'Limites : heure 90 jours, jour 2 ans, semaine 5 ans ; 200 000 lignes maximum.',
    'Model analytics exported': 'Analyse des modèles exportée',
    'No model data is available in this time range':
      'Aucune donnée de modèle exportable sur cette période',
    'Select a model': 'Sélectionner un modèle',
    'The export uses complete natural time buckets in your browser time zone.':
      'L’export utilise des périodes calendaires complètes dans le fuseau horaire du navigateur.',
    'The selected export range is too large for this granularity':
      'La plage choisie dépasse la limite de cette granularité',
    'The selected model has no data in this time range':
      'Le modèle choisi n’a aucune donnée sur cette période',
    'invalid model analytics export request':
      'Demande d’export d’analyse des modèles non valide',
    'model analytics export exceeds the allowed row count':
      'L’export d’analyse des modèles dépasse le nombre de lignes autorisé',
    'model analytics export range is too large':
      'La plage d’export d’analyse des modèles est trop grande',
    'no model analytics data found':
      'Aucune donnée d’analyse des modèles trouvée',
  },
  ja: {
    'All models': 'すべてのモデル',
    'Effective range: {{start}} to {{end}} ({{timezone}})':
      '実際の範囲：{{start}} ～ {{end}}（{{timezone}}）',
    'Export CSV': 'CSV をエクスポート',
    'Export Model Analytics': 'モデル分析をエクスポート',
    'Export request, token, quota, RPM, and TPM data by model and time bucket.':
      'モデルと時間帯ごとのリクエスト数、トークン、クォータ、RPM、TPM をエクスポートします。',
    'Export Settings': 'エクスポート設定',
    'Exporting model analytics...': 'モデル分析をエクスポート中...',
    'Failed to export model analytics':
      'モデル分析のエクスポートに失敗しました',
    'Failed to load exportable models':
      'エクスポート可能なモデルの読み込みに失敗しました',
    'Limits: hourly 90 days, daily 2 years, weekly 5 years; up to 200,000 rows.':
      '上限：時間単位は90日、日単位は2年、週単位は5年、最大200,000行。',
    'Model analytics exported': 'モデル分析をエクスポートしました',
    'No model data is available in this time range':
      'この期間にはエクスポート可能なモデルデータがありません',
    'Select a model': 'モデルを選択',
    'The export uses complete natural time buckets in your browser time zone.':
      'ブラウザーのタイムゾーンに基づく完全な暦時間帯でエクスポートします。',
    'The selected export range is too large for this granularity':
      '選択した範囲は、この時間粒度の上限を超えています',
    'The selected model has no data in this time range':
      '選択したモデルには、この期間のデータがありません',
    'invalid model analytics export request':
      'モデル分析のエクスポート要求が無効です',
    'model analytics export exceeds the allowed row count':
      'モデル分析のエクスポートが許容行数を超えています',
    'model analytics export range is too large':
      'モデル分析のエクスポート範囲が大きすぎます',
    'no model analytics data found': 'モデル分析データが見つかりません',
  },
  ru: {
    'All models': 'Все модели',
    'Effective range: {{start}} to {{end}} ({{timezone}})':
      'Фактический диапазон: {{start}} — {{end}} ({{timezone}})',
    'Export CSV': 'Экспорт CSV',
    'Export Model Analytics': 'Экспорт аналитики моделей',
    'Export request, token, quota, RPM, and TPM data by model and time bucket.':
      'Экспорт запросов, токенов, квоты, RPM и TPM по моделям и временным интервалам.',
    'Export Settings': 'Настройки экспорта',
    'Exporting model analytics...': 'Экспорт аналитики моделей...',
    'Failed to export model analytics':
      'Не удалось экспортировать аналитику моделей',
    'Failed to load exportable models':
      'Не удалось загрузить модели для экспорта',
    'Limits: hourly 90 days, daily 2 years, weekly 5 years; up to 200,000 rows.':
      'Ограничения: час — 90 дней, день — 2 года, неделя — 5 лет; до 200 000 строк.',
    'Model analytics exported': 'Аналитика моделей экспортирована',
    'No model data is available in this time range':
      'За этот период нет данных моделей для экспорта',
    'Select a model': 'Выберите модель',
    'The export uses complete natural time buckets in your browser time zone.':
      'При экспорте используются полные календарные интервалы в часовом поясе браузера.',
    'The selected export range is too large for this granularity':
      'Выбранный диапазон превышает предел для этой детализации',
    'The selected model has no data in this time range':
      'Для выбранной модели нет данных за этот период',
    'invalid model analytics export request':
      'Недопустимый запрос экспорта аналитики моделей',
    'model analytics export exceeds the allowed row count':
      'Экспорт аналитики моделей превышает допустимое число строк',
    'model analytics export range is too large':
      'Диапазон экспорта аналитики моделей слишком велик',
    'no model analytics data found': 'Данные аналитики моделей не найдены',
  },
  vi: {
    'All models': 'Tất cả mô hình',
    'Effective range: {{start}} to {{end}} ({{timezone}})':
      'Khoảng thực tế: {{start}} đến {{end}} ({{timezone}})',
    'Export CSV': 'Xuất CSV',
    'Export Model Analytics': 'Xuất phân tích mô hình',
    'Export request, token, quota, RPM, and TPM data by model and time bucket.':
      'Xuất dữ liệu yêu cầu, token, hạn mức, RPM và TPM theo mô hình và khoảng thời gian.',
    'Export Settings': 'Cài đặt xuất',
    'Exporting model analytics...': 'Đang xuất phân tích mô hình...',
    'Failed to export model analytics': 'Không thể xuất phân tích mô hình',
    'Failed to load exportable models': 'Không thể tải các mô hình có thể xuất',
    'Limits: hourly 90 days, daily 2 years, weekly 5 years; up to 200,000 rows.':
      'Giới hạn: theo giờ 90 ngày, theo ngày 2 năm, theo tuần 5 năm; tối đa 200.000 dòng.',
    'Model analytics exported': 'Đã xuất phân tích mô hình',
    'No model data is available in this time range':
      'Không có dữ liệu mô hình để xuất trong khoảng thời gian này',
    'Select a model': 'Chọn mô hình',
    'The export uses complete natural time buckets in your browser time zone.':
      'Bản xuất dùng các khoảng thời gian lịch đầy đủ theo múi giờ của trình duyệt.',
    'The selected export range is too large for this granularity':
      'Khoảng đã chọn vượt quá giới hạn của độ chi tiết này',
    'The selected model has no data in this time range':
      'Mô hình đã chọn không có dữ liệu trong khoảng thời gian này',
    'invalid model analytics export request':
      'Yêu cầu xuất phân tích mô hình không hợp lệ',
    'model analytics export exceeds the allowed row count':
      'Bản xuất phân tích mô hình vượt quá số dòng cho phép',
    'model analytics export range is too large':
      'Khoảng xuất phân tích mô hình quá lớn',
    'no model analytics data found': 'Không tìm thấy dữ liệu phân tích mô hình',
  },
}

const responsesCompatEnglish = {
  'Automatically detects standard Responses, SGLang text compatibility, and Chat Completions fallback.':
    'Automatically detects standard Responses, SGLang text compatibility, and Chat Completions fallback.',
  'Detection sends minimal streaming and non-streaming requests, then caches the compatible upstream format.':
    'Detection sends minimal streaming and non-streaming requests, then caches the compatible upstream format.',
  'Native Responses (text compatibility)':
    'Native Responses (text compatibility)',
}

const responsesCompatTranslations = {
  zh: {
    'Automatically detects standard Responses, SGLang text compatibility, and Chat Completions fallback.':
      '自动检测标准 Responses、SGLang 文本兼容和 Chat Completions 回退。',
    'Detection sends minimal streaming and non-streaming requests, then caches the compatible upstream format.':
      '检测会发送最小化的流式和非流式请求，然后缓存兼容的上游格式。',
    'Native Responses (text compatibility)': '原生 Responses（文本兼容）',
  },
  'zh-TW': {
    'Automatically detects standard Responses, SGLang text compatibility, and Chat Completions fallback.':
      '自動偵測標準 Responses、SGLang 文字相容與 Chat Completions 後備模式。',
    'Detection sends minimal streaming and non-streaming requests, then caches the compatible upstream format.':
      '偵測會傳送最小化的串流與非串流請求，然後快取相容的上游格式。',
    'Native Responses (text compatibility)': '原生 Responses（文字相容）',
  },
  fr: {
    'Automatically detects standard Responses, SGLang text compatibility, and Chat Completions fallback.':
      'Détecte automatiquement le format Responses standard, la compatibilité texte SGLang et le repli Chat Completions.',
    'Detection sends minimal streaming and non-streaming requests, then caches the compatible upstream format.':
      'La détection envoie des requêtes minimales avec et sans streaming, puis met en cache le format amont compatible.',
    'Native Responses (text compatibility)':
      'Responses natif (compatibilité texte)',
  },
  ja: {
    'Automatically detects standard Responses, SGLang text compatibility, and Chat Completions fallback.':
      '標準 Responses、SGLang テキスト互換、Chat Completions フォールバックを自動検出します。',
    'Detection sends minimal streaming and non-streaming requests, then caches the compatible upstream format.':
      '検出で最小限のストリーミングおよび非ストリーミングリクエストを送信し、互換性のある上流形式をキャッシュします。',
    'Native Responses (text compatibility)':
      'ネイティブ Responses（テキスト互換）',
  },
  ru: {
    'Automatically detects standard Responses, SGLang text compatibility, and Chat Completions fallback.':
      'Автоматически определяет Responses, текстовую совместимость SGLang и резервный Chat Completions.',
    'Detection sends minimal streaming and non-streaming requests, then caches the compatible upstream format.':
      'Проверка отправляет минимальные потоковые и непотоковые запросы, затем кэширует совместимый upstream-формат.',
    'Native Responses (text compatibility)':
      'Нативный Responses (совместимость текста)',
  },
  vi: {
    'Automatically detects standard Responses, SGLang text compatibility, and Chat Completions fallback.':
      'Tự động phát hiện Responses chuẩn, tương thích văn bản SGLang và dự phòng Chat Completions.',
    'Detection sends minimal streaming and non-streaming requests, then caches the compatible upstream format.':
      'Quá trình phát hiện gửi các yêu cầu truyền luồng và không truyền luồng tối thiểu, sau đó lưu bộ nhớ đệm định dạng upstream tương thích.',
    'Native Responses (text compatibility)':
      'Responses gốc (tương thích văn bản)',
  },
}

const usageLogExportEnglish = {
  'Exporting usage logs...': 'Exporting usage logs...',
  'Failed to export usage logs': 'Failed to export usage logs',
  'Usage logs exported': 'Usage logs exported',
  'invalid usage log export request': 'Invalid usage log export request',
  'no usage logs found': 'No usage logs found',
  'usage log export exceeds the allowed row count':
    'Usage log export exceeds the allowed row count',
}

const usageLogExportTranslations = {
  zh: {
    'Exporting usage logs...': '正在导出使用日志...',
    'Failed to export usage logs': '使用日志导出失败',
    'Usage logs exported': '使用日志已导出',
    'invalid usage log export request': '使用日志导出请求无效',
    'no usage logs found': '未找到使用日志',
    'usage log export exceeds the allowed row count':
      '使用日志导出超过允许的行数',
  },
  'zh-TW': {
    'Exporting usage logs...': '正在匯出使用記錄...',
    'Failed to export usage logs': '使用記錄匯出失敗',
    'Usage logs exported': '使用記錄已匯出',
    'invalid usage log export request': '使用記錄匯出請求無效',
    'no usage logs found': '找不到使用記錄',
    'usage log export exceeds the allowed row count':
      '使用記錄匯出超過允許的列數',
  },
  fr: {
    'Exporting usage logs...': "Exportation des journaux d'utilisation...",
    'Failed to export usage logs':
      "Échec de l'exportation des journaux d'utilisation",
    'Usage logs exported': "Journaux d'utilisation exportés",
    'invalid usage log export request':
      "Demande d'exportation des journaux d'utilisation non valide",
    'no usage logs found': "Aucun journal d'utilisation trouvé",
    'usage log export exceeds the allowed row count':
      "L'exportation des journaux d'utilisation dépasse le nombre de lignes autorisé",
  },
  ja: {
    'Exporting usage logs...': '使用ログをエクスポートしています...',
    'Failed to export usage logs': '使用ログのエクスポートに失敗しました',
    'Usage logs exported': '使用ログをエクスポートしました',
    'invalid usage log export request': '使用ログのエクスポート要求が無効です',
    'no usage logs found': '使用ログが見つかりません',
    'usage log export exceeds the allowed row count':
      '使用ログのエクスポートが許容行数を超えています',
  },
  ru: {
    'Exporting usage logs...': 'Экспорт журналов использования...',
    'Failed to export usage logs':
      'Не удалось экспортировать журналы использования',
    'Usage logs exported': 'Журналы использования экспортированы',
    'invalid usage log export request':
      'Недопустимый запрос экспорта журналов использования',
    'no usage logs found': 'Журналы использования не найдены',
    'usage log export exceeds the allowed row count':
      'Экспорт журналов использования превышает допустимое число строк',
  },
  vi: {
    'Exporting usage logs...': 'Đang xuất nhật ký sử dụng...',
    'Failed to export usage logs': 'Không thể xuất nhật ký sử dụng',
    'Usage logs exported': 'Đã xuất nhật ký sử dụng',
    'invalid usage log export request':
      'Yêu cầu xuất nhật ký sử dụng không hợp lệ',
    'no usage logs found': 'Không tìm thấy nhật ký sử dụng',
    'usage log export exceeds the allowed row count':
      'Bản xuất nhật ký sử dụng vượt quá số hàng cho phép',
  },
}

const newKeys = {
  en: {
    ...english,
    ...phase2English,
    ...phase3English,
    ...currentLoadEnglish,
    ...modelAnalyticsExportEnglish,
    ...responsesCompatEnglish,
    ...usageLogExportEnglish,
  },
  zh: {
    ...phase2Translations.zh,
    ...phase3Translations.zh,
    ...currentLoadTranslations.zh,
    ...modelAnalyticsExportTranslations.zh,
    ...responsesCompatTranslations.zh,
    ...usageLogExportTranslations.zh,
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
  'zh-TW': {
    ...english,
    ...phase2Translations['zh-TW'],
    ...phase3Translations['zh-TW'],
    ...currentLoadTranslations['zh-TW'],
    ...modelAnalyticsExportTranslations['zh-TW'],
    ...responsesCompatTranslations['zh-TW'],
    ...usageLogExportTranslations['zh-TW'],
  },
  fr: {
    ...english,
    ...phase2Translations.fr,
    ...phase3Translations.fr,
    ...currentLoadTranslations.fr,
    ...modelAnalyticsExportTranslations.fr,
    ...responsesCompatTranslations.fr,
    ...usageLogExportTranslations.fr,
  },
  ja: {
    ...english,
    ...phase2Translations.ja,
    ...phase3Translations.ja,
    ...currentLoadTranslations.ja,
    ...modelAnalyticsExportTranslations.ja,
    ...responsesCompatTranslations.ja,
    ...usageLogExportTranslations.ja,
  },
  ru: {
    ...english,
    ...phase2Translations.ru,
    ...phase3Translations.ru,
    ...currentLoadTranslations.ru,
    ...modelAnalyticsExportTranslations.ru,
    ...responsesCompatTranslations.ru,
    ...usageLogExportTranslations.ru,
  },
  vi: {
    ...english,
    ...phase2Translations.vi,
    ...phase3Translations.vi,
    ...currentLoadTranslations.vi,
    ...modelAnalyticsExportTranslations.vi,
    ...responsesCompatTranslations.vi,
    ...usageLogExportTranslations.vi,
  },
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
