# 集群数据导出弹窗布局优化

## 1. 调整目标

集群数据导出弹窗原先沿用较窄的默认宽度。切换到“时间范围”后，日期时间输入框、快捷时间按钮、时区说明和历史数据起始时间集中在有限空间内，中文界面容易出现文本换行过密和输入内容拥挤的问题。

本次调整只优化弹窗布局与响应式行为，不修改导出参数、时间范围校验、接口调用或文件下载逻辑。

## 2. 布局调整

- 桌面端弹窗最大宽度调整为 `2xl`，为日期时间输入和说明文字提供更多空间；
- 弹窗最大高度限制为当前视口高度减去安全边距，内容过高时允许纵向滚动；
- 标题说明增加行高，并为右上角关闭按钮预留空间；
- 开始时间和结束时间只在中等及以上宽度并排显示，较窄窗口自动改为上下排列；
- 时区和最早可用历史时间拆分为两行，避免两段信息挤在同一行；
- 同步调整桌面端内容区和底部操作区的内边距，保持视觉对齐。

## 3. 回归测试

新增导出弹窗布局回归测试，保护以下用户可见行为：

- 桌面端使用加宽后的弹窗；
- 弹窗内容超过视口时可以滚动；
- 日期时间输入在空间不足时保持单列；
- 时区和历史数据起始时间分别显示。

执行：

```bash
cd web
node --experimental-strip-types --test \
  src/features/cluster-status/__tests__/layout.test.ts
./node_modules/.bin/tsgo -b
./node_modules/.bin/oxlint -c .oxlintrc.json \
  src/features/cluster-status/components/cluster-export-dialog.tsx \
  src/features/cluster-status/components/cluster-history-range-fields.tsx \
  src/features/cluster-status/lib/export-dialog-layout.ts \
  src/features/cluster-status/__tests__/layout.test.ts
./node_modules/.bin/rsbuild build
```

以上测试、类型检查、Lint 和生产构建均通过。

## 4. 主要改动文件

- `web/src/features/cluster-status/components/cluster-export-dialog.tsx`
- `web/src/features/cluster-status/components/cluster-history-range-fields.tsx`
- `web/src/features/cluster-status/lib/export-dialog-layout.ts`
- `web/src/features/cluster-status/__tests__/layout.test.ts`

## 5. 数据库与接口影响

本次调整不涉及数据库表结构、历史数据、Redis、后端接口或导出文件格式，无需执行数据库迁移。
