# raw·lens 品牌资产

保真的 wire-level HTTP 观察工具的标志系统。全部为**矢量（SVG）**，可无损缩放，可直接用于应用图标、UI 内联与文档。

## 设计释义

- **六边形光圈（aperture）**：摄影镜头的光圈叶片，把视场收束到一个点——对应产品「对准、逐字节看清裸 socket」的职责。
- **中心点**：被对准、被原样留下的**那一个 byte**（the byte, as sent）。
- **字节紫 `0D 0A`**：CRLF 原样保留，是「保真」的灵魂——header 顺序、大小写、重复项一概不动。
- 镜头 + 字节两个意象融合，克制、可拥有，贴合「不经 `net/http` 路由层、逐字节读 socket」的产品原则。

## 调色板

| 名称 | 用途 | HEX |
| :-- | :-- | :-- |
| 光圈绿 Aperture | 主强调色（标记、链接、激活态） | `#34E0A1` |
| 字节紫 Byte | 次强调（CRLF / 十六进制等保真语义点缀） | `#A394F2` |
| 炭黑 Canvas | 深色界面底 / 深底标志底 | `#070A09` |
| 面板 Panel | 卡片 / 面板 | `#0D1513` |
| 象牙 Text | 主文字 | `#CFE3DA` |

**纪律**：一主（绿）一辅（紫），其余皆中性。强调色跨物料反复出现，禁止彩虹色。

## 文件清单

| 文件 | 用途 |
| :-- | :-- |
| `raw-lens-mark.svg` | 主标志（浅底：墨色叶片 + 绿色中心字节） |
| `raw-lens-mark-dark.svg` | 深底版（发光的光圈绿，= 产品内已落地标记） |
| `raw-lens-mark-mono.svg` | 单色版，用 `currentColor` 跟随上下文颜色，适合 UI 内联 / 灰度 |
| `raw-lens-app-icon.svg` | 应用图标（圆角方块，可导出各尺寸 png/icns/ico） |
| `raw-lens-favicon.svg` | favicon（64×64 圆角深底） |
| `raw-lens-wordmark.svg` | 横向字标组合（标记 + raw·lens + 一句话定位） |
| `raw-lens-brand-board.svg` | 品牌标志板（标志 / 构成 / 图标 / 色彩 / 组合 / 主张） |
| `preview.html` | 浏览器内一览全部资产 |

## 使用约定

- **留白**：标志四周至少保留一个「光圈半径」的安全边距。
- **最小尺寸**：图标 ≥ 24px、字标组合 ≥ 160px 宽，更小请只用纯标记。
- **明暗**：浅底用 `raw-lens-mark.svg`，深底用 `raw-lens-mark-dark.svg`，跟随上下文取色用 `raw-lens-mark-mono.svg`。
- **字体**：字标用 `IBM Plex Mono`（回退 `ui-monospace`）渲染——逐字节读 socket 的工具，字也该是等宽的；**正式交付前请把文字转曲（outline）**，避免缺字体时走样。
- **生成应用图标**：以 `raw-lens-app-icon.svg` 为母版导出 png（如 `rsvg-convert -w 512 raw-lens-app-icon.svg -o icon.png`），再按需生成 icns/ico。

## 本地预览

```bash
open brand/preview.html
```
