# fontcreator233 设计文档

日期：2026-11-17
状态：待批准
仓库：`github.com/neko233-com/fontcreator233`（public）
语言：Go 1.27，跨平台（Windows / macOS / Linux）

## 1. 意图

fontcreator233 是一个**确定性、离线、agent 原生**的字体创造器 CLI。

外部 agent（Claude、GPT、OpenCode 等）负责它最擅长的事：看美术参考图、理解风格、把视觉语言翻译成参数。CLI 不调用视觉模型、不联网、不猜测；它只负责把一份可审查的 **recipe** 变成真正的字体文件、字库和可审阅样张。

产物必须能直接进入实际管线：Unity（含 TextMeshPro / TMP）、Unreal Engine、操作系统字体目录、Web（`@font-face`、WOFF、CSS）。其中 Unity TMP 是深度集成目标：不仅给 TTF，还给可安装的 UPM 包、Editor 导入脚本、TMP Font Asset / fallback / 字符集 / 图集配置，并在本机 Unity 中做真实导入验证。

```text
参考图 / 品牌规范 / 文字描述
        │  agent 的视觉理解
        ▼
style brief（agent 自己写，可留在对话里）
        │  agent 写 recipe.yaml
        ▼
fontcreator233  ──►  .ttf / .woff  ──►  .html 样张（自动打开）
        │
        └─►  JSON 结果 / 覆盖率 / 警告 / SHA-256
```

核心契约：

- agent 看世界，CLI 造字体。
- 相同 recipe + 相同 seed 永远得到字节级相同的字体。
- 所有 agent 需要的信息都有机器可读出口（JSON、JSON Schema、MCP）。
- 人类审阅只需要打开一个 HTML。

## 2. 目标

1. 单二进制 CLI：`fontcreator`，覆盖 Windows / macOS / Linux。
2. 从 recipe 生成可安装的 TrueType 字体（`.ttf`），可选 `.woff`。
3. 内置参数化字形引擎，生成拉丁、希腊、西里尔字形；支持字重、字宽、倾斜、对比度、衬线、端点、粗糙度、x-height、cap-height 等风格参数。
4. 全球化覆盖：CJK（汉字）、日文假名、韩文谚文、符号、标点、货币、箭头等通过内嵌 OFL 基础字库派生，与生成字形合并成一套字体。
5. 生成 `*.specimen.html` 样张，包含多语言排版、字号瀑布、字形网格、样式参数，并在桌面环境自动打开。
6. Agent 原生：
   - `fontcreator describe --json` 输出完整机器契约与 JSON Schema。
   - 所有关键命令支持 `--json` 信封输出。
   - 内置 MCP stdio server（`fontcreator mcp`），可被任意 MCP 客户端直接挂载。
   - 仓库内含 agent skill（`agent-skill/fontcreator233/SKILL.md`）与 `AGENTS.md`，包含“参考图 → 风格参数”的映射指南。
7. 字库能力：
   - `library list` / `library export` 提供内置骨架字库。
   - 支持外部 SVG 字形覆盖目录，agent 可以手绘或生成单字后合并。
8. 自动化测试覆盖 3 套明显不同风格的字体，每套都断言全球化覆盖、结构合法、可复现、样张可用。
9. 目标平台开箱支持：
   - **Unity TMP**：生成 UPM 包 + Editor 导入器，自动创建 TMP Font Asset、配置多图集与 fallback 链、写入字符集文件；可选按 CJK 分片，服务静态图集和移动端。
   - **Unreal Engine**：生成 TTF + 集成清单 + Editor Python 导入脚本，提供 CompositeFont fallback 范围配置。
   - **系统**：标准 TTF，name/OS-2/cmap 正确，给出 Windows/macOS/Linux 安装说明，并用 fontTools/fontconfig 交叉校验。
   - **Web**：TTF + WOFF + 生成 `font.css`（含 `unicode-range`），样张可直接审阅。
10. 提供 `--target` / recipe `output.targets` 选择产物，`--split-cjk N` 可按码位数切分 CJK 子字体，生成 fallback 顺序与 Web `unicode-range`。
11. Unity TMP 产物以**生成即兼容**为目标（Unity 2022.3 + TMP 3.x、Unity 6 + uGUI/TMP）；不要求安装或启动 Unity 做测试，只用文件级断言与 API 审查保证兼容性。
12. **好看的 CLI**：基于 Bubble Tea / Bubbles / Lip Gloss 的 TUI，带总体进度、分阶段进度、当前字形、耗时、ETA、日志与取消；`--plain`、`--json`、非 TTY、CI 自动降级为纯文本。
13. **并行生成**：
    - 单个字体内部按字形/分片并行生成与编码（`--jobs N`，默认 CPU 核数）。
    - 多字体构建并行（`build-all --font-jobs N`），带多行进度面板。
    - 结果顺序固定；`--jobs 1` 与 `--jobs N` 的 SHA-256 必须一致。
14. **样张前端复用本地公共资源**：Vercel/Geist 审美、Geist 字体、TanStack Virtual 虚拟列表、共享 CSS/JS，全部落盘为本地文件，运行时零 CDN；4 万字形网格只渲染可视行。
15. **工作区清理**：版本化 state 清单、构建 staging、自动清理临时/过期/同族旧文件；`fontcreator clean` / `status` 支持 dry-run、按 target/时间筛选、只删自己生成且未被修改的文件。
16. **导入导出**：导入 TTF/OTF 生成 profile/recipe 与 SVG 字库；导入/导出 bundle zip；导出字形、字体、样张、平台包；安全校验与许可提示。
17. 公开到 GitHub，CI 在三大平台跑测试，并上传 3 套产物（含各平台集成包）供审阅。

## 3. 非目标

- CLI 内嵌 LLM、调用视觉模型、联网分析参考图。
- 阿拉伯文、希伯来文、印度系文字、泰文等需要复杂 shaping / RTL 的文字。
- 可变字体（variable font）输出、COLR/CPAL 彩色字体、emoji。
- OpenType GPOS/GSUB 高级排版（任意组合标记定位、连字、复杂 kerning）。
- 浏览器端编辑器、GUI 应用。
- WOFF2 编码（运行时无 Brotli/变形依赖）；输出 WOFF，WOFF2 作为可选外部后处理并在文档说明。
- 生成 Unity `.unitypackage`、TMP `.asset` 二进制/YAML、UE `.uasset` 等版本相关资产；这些由目标平台里的 Editor 脚本读取 TTF/清单后生成。
- Unity/UE 运行时插件、运行时字体热更新、在线字体服务。
- 样张运行时依赖 CDN、npm、React/Next 或任何联网资源；只使用落盘的本地 CSS/JS/字体。
- 复制 Vercel 专有后台代码；只采用公开的 Geist 字体与 Vercel 式设计语言（黑白、细边框、网格、留白）。
- 从任意商业字体重建同品质字库；导入仅生成 profile、受限字形导出与 bundle，不绕过许可。
- 逐字追求商业字体品质；v1 目标是“风格明确、结构正确、全球化可用、可继续迭代”。

## 4. 反目标

- 输出的字体无法被浏览器或操作系统加载。
- 相同 recipe 两次构建结果不同。
- 声称支持某语言但 cmap 缺字、字形为空、advance 为 0。
- 自动打开样张在 CI 或无桌面环境中卡住或失败。
- 把来源不明的商业字体打包进仓库或输出物。
- TUI/进度条污染 stdout，导致 `--json` 不可用。
- 清理命令删除用户文件、未跟踪文件或被用户修改过的生成文件。
- 并行导致输出顺序漂移或 SHA-256 不一致。
- 样张在 `file://` 下因 CORS、ES module 或 CDN 缺失而白屏。

## 5. 约束

- Go 1.27，`CGO_ENABLED=0`。
- 允许的第三方依赖：
  - `golang.org/x/image/font/sfnt`：解析内嵌基础字体、导入字体、测试时校验输出。
  - `golang.org/x/text/unicode/norm`：预组合字符的 NFD 分解（é → e + acute）。
  - `golang.org/x/term`：TTY 检测。
  - `gopkg.in/yaml.v3`：recipe 解析。
  - `github.com/charmbracelet/bubbletea`、`github.com/charmbracelet/bubbles`、`github.com/charmbracelet/lipgloss`：TUI、进度条、样式。
  - 其余用标准库；不引入 ORM、Web 框架或 Node 运行时。
- 字体单位固定 `unitsPerEm = 1000`（v1）。两个内嵌基础字体也是 1000 UPM，合并无需换算。
- 内嵌基础字库（均为 OFL，已静态化/子集化并改名）：
  - `Fontcreator233BaseSC-Regular.ttf`（约 10.3 MB，来自 Noto Sans SC wght=400 实例，CJK/假名/符号）。
  - `Fontcreator233BaseKR-Regular.ttf`（约 2.4 MB，来自 Noto Sans KR wght=400 实例，子集为 Hangul/Jamo/兼容字母）。
  - 两者合计约 12.7 MB，二进制约 15–18 MB。
- 运行时零网络请求。基础字库、样张 CSS/JS、Geist 字体全部通过 `go:embed` 打包。
- 样张前端资产：
  - Geist Sans/Mono（SIL OFL 1.1，来自 `vercel/geist-font`）。
  - `@tanstack/virtual-core`（MIT）预打包为经典 `<script>`（无 ESM，兼容 `file://`）。
  - 自研 `specimen.css` / `specimen.js` / glyph data JS，全部本地相对路径。
- 并行：默认 `jobs = min(runtime.NumCPU(), 16)`；`--jobs` 与 `--font-jobs` 可覆盖。所有并行结果按索引汇总，测试 `--jobs 1` 与 `--jobs 8` 字节一致。
- 工作区：输出目录下 `.fontcreator233/` 保存 state/lock/tmp/history；只清理自己生成且 SHA-256 未变的文件。
- `.gitignore` 必须保证生成物不入库：
  - 忽略 `dist/`、`out/`、`work/`、`build/`、`tmp/`、`.fontcreator233/`、`*.bundle.zip`、本地 Unity 测试项目、Playwright 报告、OS/IDE 文件。
  - 显式保留源资产：`!assets/base/*.ttf`、`!assets/base/*.txt`、`!internal/specimen/webassets/**`（vendor JS/Geist/许可证）。
  - 生成字体、WOFF、样张 HTML、glyph data JS、平台包都不提交；示例只提交 recipe 与小型 SVG。
  - 提交前用 `git status --porcelain` 检查；CI 增加“无生成物被跟踪”的检查（`git ls-files dist work out` 为空）。
- 构建产物默认写到 `./dist`，路径可用 `--out` 指定。
- Windows 路径、空格、中文文件名都必须工作。
- Unity TMP 集成兼容 TextMeshPro 3.x（Unity 2022.3）与 Unity 6 的 uGUI/TMP 包；Editor 脚本放在 `Editor/` 预定义程序集，不使用 asmdef，避免 TMP 包版本导致的程序集引用失败。
- Unity 集成只在 Editor 中运行，不修改 ProjectSettings，不自动覆盖用户已有 TMP 资产；所有动作通过菜单显式触发，可重复执行。
- 本机虽安装了 Unity 2022.3.51f1 与 6000.6.2f1，但本设计**不要求**启动 Unity 做测试；Unity 集成以生成的 UPM/C# 源码与清单文件级测试为准，兼容性通过 API/版本审查保证。
- 本机 Python 3.14 + fontTools 4.66 可用；fontTools/fontconfig 仅作为开发与 CI 的交叉校验工具，不是运行时依赖。

## 6. 方案对比与选择

### 方案 A：混合引擎（选定）

- 拉丁/希腊/西里尔：自研骨架 + 笔画引擎，参数化生成轮廓。
- CJK/假名/谚文/符号：从内嵌的 OFL 基础字体（Noto Sans SC + Noto Sans KR 的静态子集）派生，做几何变换与合成加粗。
- 优点：风格跨度大、确定性、离线、可测试、全球化覆盖真实。
- 代价：CJK/Hangul 的风格迁移弱于自绘（只做宽度/倾斜/合成加粗），需要打包约 12.7 MB 基础字体。

### 方案 B：全脚本参数化

- 所有文字都从自研骨架生成。
- 优点：完全统一风格、无外部字体。
- 代价：两万多个汉字不可能在 v1 手绘；只能给出极窄的 CJK 覆盖，违背全球化目标。
- 结论：拒绝。

### 方案 C：纯字库改绘（remix）

- 全部字形来自打包的多套 OFL 字体，只做仿射变换与描边调整。
- 优点：轮廓质量高、实现快。
- 代价：风格范围被基础字体锁死，更像“字体混音器”而不是创造器；为了覆盖多种风格要打包大量字体。
- 结论：不作为主路线；CJK 部分采用该思路。

### 方案 D：CLI 内嵌视觉模型

- CLI 自己读图、调用模型、生成字体。
- 优点：单步完成。
- 代价：密钥、费用、非确定性、难测试，且用户已明确选择“agent 自己看完图，CLI 只负责生成”。
- 结论：拒绝。

推荐并选定方案 A。

## 7. 总体架构

```text
cmd: fontcreator
  ├─ cli            命令解析、JSON 信封、打开浏览器
  ├─ recipe         YAML/JSON 解析、默认值、严格校验、JSON Schema
  ├─ style          参数解析：字重→笔画宽度、对比度、缩放、seed
  ├─ geom           点/轮廓/贝塞尔/变换/面积/包围盒/多边形外扩
  ├─ skeleton       内置骨架数据 + 笔画引擎 + 衬线/端点/抖动
  ├─ compose        NFD 预组合字符、希腊/西里尔派生、标记定位
  ├─ library        SVG 字形覆盖、字库导出、覆盖率清单
  ├─ derive         内嵌 SC/KR 基础字库解析、按 Unicode 区间取字形、变换
  ├─ fontbuild      TTF 表构造、cmap、name、OS/2、hmtx、loca、glyf、kern
  ├─ specimen       HTML 样张模板、本地 assets、glyph data
  ├─ integration    跨平台集成清单（schema + 字符集 + fallback 顺序）
  ├─ targets        目标产物打包：unity / unreal / web / system
  ├─ ui             Bubble Tea TUI、plain/json reporter、进度模型
  ├─ parallel       有界 worker pool、确定性索引、进度聚合
  ├─ workspace      state/lock/staging、自动清理、clean/status
  ├─ importexport   TTF profile、SVG glyph 导出、bundle zip
  ├─ agentschema    describe 契约、JSON Schema、agent guide
  └─ mcp            MCP stdio（JSON-RPC 2.0）
```

数据流：

1. `recipe.Load` → 严格校验 → 默认值归一化。
2. `workspace.Begin` → 取输出目录锁 → 自动清理过期 staging/history → 建立 staging 目录。
3. `style.Resolve` → 得到具体数值（stem width、contrast、scale、serif size 等）。
4. `parallel.Map` 生成任务：骨架、NFD 组合、基础字库派生、SVG overrides；每个任务带码位与确定 seed。进度事件节流上报 TUI/plain reporter。
5. 按码位排序合并 glyph 集合：生成字形 > SVG override > 基础字形。
6. `fontbuild.Build` 并行编码 `glyf`/`loca` 分块 → 顺序组装表 → `[]byte` TTF；可选 WOFF。
7. 逐个 split 分片并行构建（如启用）；每个分片独立 cmap。
8. `targets.Build` 按 `output.targets` 生成平台集成包（Unity UPM、UE 清单/脚本、Web CSS、系统说明）。
9. `specimen.Build` 写 HTML、glyph data JS、复制/复用本地 `assets/`（CSS/JS/Geist/TanStack）。
10. `workspace.Commit` → 校验输出 → 移入正式路径 → 清理同族旧文件 → 原子更新 state/history。
11. `cli.Open`（除非 `--no-open` 或 MCP 默认）。
12. TUI 完成页 / plain 汇总 / JSON 结果。

## 8. Recipe 契约

默认格式 YAML；同时接受 JSON。严格模式，未知字段报错（防止 agent 拼错参数却静默成功）。

```yaml
apiVersion: fontcreator233/v1

meta:
  family: Aurora Grotesk
  style: Regular
  version: 1.0.0
  designer: neko233
  description: Geometric sans derived from a neon poster reference.
  license: OFL-1.1          # 使用了基础字库时强制 OFL-1.1
  accent: "#5B8CFF"         # 样张主题色
  sampleText: "Hamburgefonstiv 字体创造器"

metrics:
  ascender: 800
  descender: -200
  xHeight: 0.52             # 0.40–0.60，相对 em
  capHeight: 0.72           # 0.64–0.78
  lineGap: 0

style:
  skeleton: geometric       # geometric | humanist | serif
  weight: 620               # 100–900
  width: 104                # 75–125，百分比
  slant: 0                  # -15–15 度，正数向右倾
  contrast: 0.06            # 0–1
  terminal: round           # flat | round | angled | flare
  serif: none               # none | slab | bracketed | hairline
  serifSize: 0              # 0–1，衬线相对大小；serif!=none 时生效
  overshoot: 0.012          # 圆角字母超出 x/cap 高度的比例
  tracking: 0               # 额外字距，em 百分比，-5–10
  roughness: 0              # 0–1，确定性抖动
  jitterRotation: 0         # 0–2 度，逐字微旋转
  seed: 233

coverage:
  generated: [latin, greek, cyrillic]
  base:
    cjk: common             # none | sample | common | full
    kana: true
    hangul: true
    symbols: true

glyphs:
  overrides: ""             # 相对 recipe 的 SVG 目录；空表示不用

output:
  formats: [ttf, woff]      # 支持 ttf / woff；默认两者
  targets: [universal, web, system, unity]  # universal|web|system|unity|unreal|all
  specimen: true
  open: true                # CLI --no-open 可覆盖
  splitCJK: 0               # >0：按码位数切分 CJK 子字体；0=不切
  tmp:                      # Unity TMP 深度集成参数
    population: dynamic     # dynamic | static
    samplingPointSize: 90
    padding: 9
    atlas: [1024, 1024]
    multiAtlas: true
    renderMode: SDFAA
```

参数语义：

| 参数 | 作用 |
|---|---|
| `weight` | 100–900 → 主笔画宽度约 34–170 units；同时影响 OS/2 `usWeightClass` |
| `width` | 水平缩放与 advance 缩放 |
| `slant` | 绕基线剪切；影响 `post.italicAngle` 与样式名 |
| `contrast` | 笔画宽度随切线角度变化；0=等宽，1=极高对比 |
| `terminal` | 开放笔画端点：平切/圆头/斜切/喇叭口 |
| `serif` | 垂直笔画末端的衬线形状 |
| `overshoot` | 圆/弧超出 x-height、cap-height 的量，避免视觉缩小 |
| `tracking` | 额外字距（em 百分比，-5–10），用于喇叭口、展示字体的呼吸感 |
| `roughness` | 基于 `seed` 的确定性点抖动，形成手绘/有机感 |
| `jitterRotation` | 每字确定性微旋转，形成海报/手工感 |
| `xHeight` / `capHeight` | 分别缩放小写/大写骨架的 y 坐标 |
| `seed` | 所有“随机”效果的唯一来源；确保可复现 |
| `coverage.base.cjk` | `sample`（约 300 字，测试/演示快）/ `common`（BMP 常用区）/ `full`（含扩展区） |
| `output.targets` | 生成哪些平台包；`universal` 只出字体与样张 |
| `output.splitCJK` | >0 时按 CJK 码位切分为多个子字体，供 TMP 静态图集与 Web `unicode-range` 使用 |
| `output.tmp.*` | 传给 Unity Editor 脚本的 TMP Font Asset 默认参数；脚本中仍可覆盖 |

`fontcreator init --template geometric|serif|display` 生成带注释的起始 recipe。

## 9. 字形引擎

### 9.1 骨架数据

- 内置约 210 个基础骨架：ASCII 可打印字符、拉丁字母扩展所需基字、希腊字母、西里尔字母、标点、数字、货币、常用符号。
- 每个骨架是中心线路径 + advance + zone（lowercase / uppercase / digit / mark）。
- 骨架使用 `1000 UPM`、baseline=0、x-height=500、cap=700、ascender=740、descender=-200 的设计空间。
- 路径用紧凑 DSL（`M/L/C/Q/Z`，绝对坐标）嵌入 Go 数据，构建时可用 `library export` 导出 SVG/JSON。
- 同一码位可有风格变体：`a` 单层/双层、`g` 单层/双层、`l` 直/带尾。`skeleton` 参数选择变体。

### 9.2 笔画展开

1. 将路径采样为折线（曲线按 0.5 unit 误差自适应细分）。
2. 每个点计算切线、法线、动态宽度：
   `w = stem * (1 - contrast + contrast * (0.12 + 0.88 * sin^4 θ))`
   其中 θ 为切线相对水平方向角度；垂直笔画厚，水平笔画薄。
3. 开放笔画：左右偏移后闭合；端点按 `terminal` 生成平头/圆头/斜头/喇叭口。
4. 闭合笔画（o、O、0 等）：生成外轮廓与内轮廓；按面积判定外/内，强制外轮廓正向、内轮廓反向。
5. 衬线：在标记为 serif 的垂直端点附加 `slab` / `bracketed` / `hairline` 多边形；与笔画重叠无需布尔运算（TrueType 非零环绕填充可以直接渲染重叠同向轮廓）。
6. 抖动：所有点加 `hash(seed, rune, pointIndex)` 的确定性偏移，再叠加逐字微旋转。
7. 输出点列 + 轮廓。所有轮廓最终转换为 quadratic/line 段。

### 9.3 预组合字符

- 使用 NFD：`é → e + U+0301`。
- 基字骨架 + 标记骨架；标记水平居中于基字视觉重心，垂直放在基字上边界之上（考虑 overshoot）。
- `i` / `j` 遇上方标记时切换到无点变体。
- 覆盖拉丁扩展、希腊 tonos、西里尔部分带附加符号字符。
- 组合标记本身也生成零 advance 字形，但不承诺任意文本中的 GPOS 定位。

### 9.4 度量

- 生成字形的 advance 由骨架 advance × `width`。
- 左右侧距按轮廓包围盒修正，保证不重叠（除非斜体/花体有意为之）。
- 字体级 ascender/descender 来自 recipe；`hhea` 与 `OS/2` 同步。
- `.notdef` 为空心矩形，保证任何缺字可见。

## 10. 全球化与基础字库

内嵌两个静态基础字体（均由 `scripts/prepare-base-fonts.py` 从 Google Fonts 的 Noto 可变字体生成，提交产物与许可）：

- `assets/base/Fontcreator233BaseSC-Regular.ttf`：来自 Noto Sans SC wght=400 实例，去除 hint/layout 表，约 10.3 MB；提供 CJK、假名、符号、拉丁扩展、希腊、西里尔补充。
- `assets/base/Fontcreator233BaseKR-Regular.ttf`：来自 Noto Sans KR wght=400 实例，子集为 Hangul Syllables/Jamo/兼容字母，约 2.4 MB。
- 两者按 OFL 要求改名（Reserved Font Name `Source` 不出现），提交 `assets/base/OFL.txt`、`NOTICE.md` 与生成脚本。
- 为什么不用可变字体默认实例：Noto 可变字体默认 wght=100（Thin），直接派生会得到细体；准备脚本固定到 400 并移除 variation 表。

派生流程：

1. 启动时对两个基础字体各 `sfnt.Parse` 一次（`sync.Once`）。
2. 按覆盖率集合遍历 Unicode 区间，用对应基础字体的 `GlyphIndex` 查字；Hangul 走 KR，其余走 SC；跳过生成字形已经占据的码位。
3. `LoadGlyph` 取轮廓（Y 轴向下 → 翻转为字体坐标）。
4. 应用：
   - `width` 水平缩放；
   - `slant` 剪切；
   - 合成加粗：`embolden = clamp((weight-400)/500*30, 0, 32)` units，按轮廓方向外扩（仅加粗，不减细）；
   - 不改变 CJK/Hangul 的 x-height/cap-height（保持方块字比例）。
5. 写入合并字形表。

覆盖率集合：

| 集合 | 内容 | 用途 |
|---|---|---|
| `none` | 不派生 | 纯生成脚本 |
| `sample` | 约 300 个代表字（常用汉字/假名/谚文/符号） | 快速测试、演示 |
| `common` | CJK Unified Ideographs BMP、CJK 标点、假名、谚文、全角、常用符号 | 默认，面向实际使用 |
| `full` | 内嵌 SC/KR 基础字库中**实际存在**的全部码位（含 SC 的 6,582 个 Ext A、54 个 Ext B 与兼容区） | 最大化覆盖，构建较慢；不承诺填满整个 Unicode 区块 |

基础字库实测（2026-11 版本）：

- SC：cmap 30,890；BMP CJK 20,976/20,992（缺 16 个）；Ext A 6,582；Ext B 54；假名 189；Hangul 0。
- KR：Hangul Syllables 11,172；Jamo 256；兼容 Jamo 94；另含部分 CJK/假名，仅作补充。
- `common`/`full` 的计数必须来自实际 cmap，不得写死区块大小。
- 覆盖率报告包含每个请求区间的 `requested`/`found`/`missing` 计数；缺失超过 0.1% 时产生 warning（`full` 不因区块不完整而失败，但必须如实报告）。
- 不声称支持“全部 Unicode”；README/describe 明确“覆盖 = 内嵌基础字库 + 生成脚本实际提供的字形”。

支持断言的语言/文字：

- 英语、法语、德语、西班牙语、葡萄牙语、意大利语（Latin-1 + 扩展）
- 希腊语
- 俄语、乌克兰语、保加利亚语、塞尔维亚语（西里尔核心）
- 简体中文（常用汉字）
- 日语（假名 + 常用汉字来自基础字库）
- 韩语（谚文）
- 通用符号、货币、箭头、数学符号

明确不支持：阿拉伯文、希伯来文、印度系文字、泰文、竖排、复杂连字。

## 11. 字体编译

纯 Go 实现 TTF 写入器，不依赖外部编译器：

- 必需表：`head`、`hhea`、`maxp`、`hmtx`、`cmap`、`loca`、`glyf`、`name`、`OS/2`、`post`。
- 可选：`kern`（内置常用拉丁字对）、`WOFF` 封装（zlib）。
- `cmap`：format 4（BMP）+ format 12（含补充平面时）；平台记录 0/3、3/1、3/10。
- 轮廓：生成字形输出为 line/quadratic；基础字体本身是 TrueType `glyf`（quadratic），保留其二次曲线节点；来自 SVG 的 cubic 自适应展开为折线/二次段。
- `head.checkSumAdjustment` 按规范计算；`loca` 使用 long 格式。
- 名称表：family、subfamily、full、PostScript、version、copyright、designer、description、license、license URL、typographic family/subfamily、sample text。
- `OS/2` v4：weight class、width class、fsSelection、Panose、Unicode range、codepage range、xHeight、capHeight。
- PostScript 名称清理：只保留 `[A-Za-z0-9-]`，长度 ≤ 63。
- 文件名清理：family + style，去掉 Windows 非法字符。

### 11.1 风格命名与 italic 位

- `meta.style` 是作者可见的 style 名；构建时解析为 `resolvedStyle`：
  - `slant == 0` → 使用 `meta.style`（空则 `Regular`）。
  - `slant != 0` 且 style 已含 `Italic`/`Oblique`（不区分大小写）→ 保持不变。
  - `slant != 0` 且 style 为空或 `Regular` → `Italic`。
  - 其他情况 → `<style> Italic`（例如 `Display` → `Display Italic`）。
- `name` 表：ID 1=family，ID 2=resolvedStyle，ID 4=`family + " " + resolvedStyle`，ID 6=清理后的 `family-resolvedStyle`，ID 16=family，ID 17=resolvedStyle。
- `post.italicAngle = -slant`（正 slant 向右倾，对应负 italicAngle）。
- `OS/2.fsSelection` 与 `head.macStyle` 的 ITALIC/BOLD 位由 resolvedStyle 与 weight 决定：slant != 0 必须置 ITALIC；weight >= 700 置 BOLD。
- 文件名与 JSON `style` 使用 `resolvedStyle`；recipe 的 `meta.style` 原值保留在 manifest 的 `requestedStyle` 字段中（可选）。
- 测试：`Regular + slant 8` → `Italic`；`Display + slant 8` → `Display Italic`；`Oblique + slant 8` → `Oblique`；italicAngle = -8。

WOFF：`WOFF` 头 + 逐表 zlib 压缩，供网页使用；HTML 仍以 TTF 为主，避免 WOFF 兼容性风险。

## 12. HTML 样张与自动打开

`fontcreator build` 默认生成样张并打开；`--no-open` 关闭。

文件：`dist/<Family>-<Style>.html`，与字体同目录，`@font-face` 使用相对路径。

页面内容：

1. 顶部：family、style、描述、版本、构建时间、覆盖率徽章、style 参数。
2. Hero：可编辑大字号样张（默认 recipe `sampleText`）。
3. 字号瀑布：12 / 16 / 24 / 36 / 48 / 72 / 96 / 120 px。
4. 多语言区：
   - Latin pangram + 西欧重音
   - Greek
   - Russian
   - 中文（含常用标点）
   - 日本語（仮名 + 漢字）
   - 한국어
   - 符号/货币/箭头
5. 字形网格：按 script 分组，虚拟列表展示全部码位（4 万+ 也能流畅滚动），点击复制字符；支持搜索码位/字符、输入框实时预览。
6. 自定义文本输入、字号滑块、浅色/深色切换。
7. “复制 CSS”按钮。
8. 页脚：fontcreator233 版本、recipe 文件名、SHA-256 前 12 位、许可证说明。

### 12.1 本地公共资源

样张不再把所有 CSS/JS 内联成一个巨石文件，而是引用同一目录下的共享资源：

```text
dist/
├─ AuroraGrotesk-Regular.html
├─ AuroraGrotesk-Regular.glyphs.js   # window.FontCreatorData = {...}
└─ assets/
   ├─ specimen.css
   ├─ specimen.js                    # classic script
   ├─ virtual-list.js                # TanStack Virtual core 的 IIFE 打包
   ├─ geist/Geist-Regular.woff2
   ├─ geist/Geist-Medium.woff2
   ├─ geist/GeistMono-Regular.woff2
   └─ licenses/
      ├─ Geist-OFL.txt
      └─ TanStack-Virtual-MIT.txt
```

- 所有资源使用相对路径，运行时不请求 CDN。
- `assets/` 是工作区级共享目录：多个字体共用；内容按版本/hash 管理，只在新版本时替换。
- `assets/versions.json` 记录资源版本与 SHA-256；`workspace` 清理时不会删除仍被引用的资源。
- 可通过 `--assets-dir` 指定外部共享目录（用于多项目复用）；HTML 中写相对路径，必要时复制。
- 许可文件随资源一起输出；`NOTICE.md` 同步列出。

### 12.2 Vercel/Geist 审美

- UI 字体：Geist Sans + Geist Mono（SIL OFL 1.1，来自 `vercel/geist-font`）。
- 视觉语言：高对比黑白、8px 网格、细边框、柔和渐变、圆角卡片、克制的动效；亮/暗色自动跟随。
- 生成字体作为展示主角；Geist 只用于界面文字与代码。
- 不使用 Vercel 专有后台代码；CSS 为本项目原创，遵循公开的设计语言。
- 主题色来自 recipe `meta.accent`，仅用于强调与选中态，不破坏黑白基调。

### 12.3 虚拟列表

- 使用 `@tanstack/virtual-core`（MIT）计算可视窗口；本项目实现轻量 DOM 渲染层。
- 字形数据由 `*.glyphs.js` 提供（`window.FontCreatorData`），包含码位、字符、script、样本分组。
- 网格支持 30,000+ 字形；仅渲染可视行（overscan 2 行）。
- 搜索/筛选时重置滚动并重新计算虚拟项；保持 60fps。
- 资源以经典 `<script>` 加载，不使用 ES module（避免 `file://` CORS 限制）。

### 12.4 file:// 与离线

- 样张必须能直接双击打开：字体、CSS、JS、glyph data 全部相对路径；不使用 `fetch()` 读取本地 JSON。
- 页面加载后检查 `document.fonts.status` 与 `document.fonts.check()`；失败时显示明确错误条，而不是白屏。
- 测试：无网络环境打开样张，断言字体加载、虚拟列表渲染、滚动、搜索、复制。
- 若用户需要单文件分发，提供 `--self-contained-html`（内联字体+CSS+JS），但不是默认路径。

自动打开：

- Windows：`cmd /c start "" <path>`
- macOS：`open <path>`
- Linux：`xdg-open <path>`
- 失败只产生 warning，不让构建失败。
- 测试通过注入 opener 验证命令选择，绝不真的启动浏览器。
- 检测 `CI=true` 时默认不打开（显式 `--open` 仍可覆盖）。

## 13. 平台集成与 Unity TMP 深度支持

### 13.1 产物矩阵

`output.targets` 或 `--target` 控制生成哪些目录：

| target | 产物 | 目的 |
|---|---|---|
| `universal` | `.ttf`、`.woff`、`.html`、`fontcreator233.manifest.json` | 基础交付，任何平台可用 |
| `web` | `web/font.css`、字体、样张 | 浏览器 `@font-face`，`unicode-range` fallback |
| `system` | `system/` 字体副本 + 安装说明 + 校验报告 | Windows/macOS/Linux 安装 |
| `unity` | `unity/` UPM 包 + TMP Editor 导入器 + 字符集 | Unity TextMeshPro 深度集成 |
| `unreal` | `unreal/` 字体 + Editor Python 脚本 + 清单 | UE Font / CompositeFont 导入 |
| `all` | 以上全部 | 发布/演示 |

默认 `targets: [universal, web, system, unity]`；`unreal` 需要显式开启，避免给不相关项目生成冗余文件。

典型目录：

```text
dist/
├─ AuroraGrotesk-Regular.ttf
├─ AuroraGrotesk-Regular.woff
├─ AuroraGrotesk-Regular.html
├─ fontcreator233.manifest.json
├─ web/
│  ├─ font.css
│  ├─ AuroraGrotesk-Regular.woff
│  └─ AuroraGrotesk-Regular.html
├─ system/
│  ├─ AuroraGrotesk-Regular.ttf
│  └─ INSTALL.md
├─ unity/
│  ├─ package.json
│  ├─ README.md
│  ├─ Editor/FontCreator233TMPImporter.cs
│  ├─ Fonts/AuroraGrotesk-Regular.ttf
│  ├─ Fonts/AuroraGrotesk-CJK-01.ttf        # 仅 splitCJK > 0
│  ├─ Charsets/latin.txt
│  ├─ Charsets/greek.txt
│  ├─ Charsets/cyrillic.txt
│  ├─ Charsets/cjk-common.txt
│  └─ fontcreator233.manifest.json
└─ unreal/
   ├─ Scripts/fontcreator233_import.py
   ├─ Fonts/AuroraGrotesk-Regular.ttf
   ├─ fontcreator233.manifest.json
   └─ README.md
```

### 13.2 集成清单

`fontcreator233.manifest.json` 是平台脚本的唯一输入，schema：

```json
{
  "schema": "fontcreator233/integration/v1",
  "generator": "fontcreator233 0.1.0",
  "family": "Aurora Grotesk",
  "style": "Regular",
  "license": "OFL-1.1",
  "licenseFile": "../OFL.txt",
  "fonts": [
    {
      "path": "Fonts/AuroraGrotesk-Regular.ttf",
      "woff": "Fonts/AuroraGrotesk-Regular.woff",
      "sha256": "…",
      "primary": true,
      "scripts": ["latin", "greek", "cyrillic"],
      "ranges": [[32, 126], [160, 591], [1024, 1279]],
      "charset": "Charsets/latin.txt"
    }
  ],
  "fallbacks": ["Fonts/AuroraGrotesk-Regular.ttf", "Fonts/AuroraGrotesk-CJK-01.ttf"],
  "unityTMP": {
    "population": "dynamic",
    "samplingPointSize": 90,
    "padding": 9,
    "atlasWidth": 1024,
    "atlasHeight": 1024,
    "multiAtlas": true,
    "renderMode": "SDFAA",
    "staticCharsets": ["Charsets/cjk-common.txt"]
  },
  "unrealComposite": {
    "default": "Fonts/AuroraGrotesk-Regular.ttf",
    "subFonts": [
      {"path": "Fonts/AuroraGrotesk-CJK-01.ttf", "ranges": [[19968, 40959]]}
    ]
  },
  "web": {
    "css": "font.css",
    "fontDisplay": "swap"
  }
}
```

- 所有路径相对清单文件。
- `ranges` 为闭区间 `[start, end]`（十进制码位）。
- `charset` 是 UTF-8 文本，每行可含多个字符；供 TMP 静态图集和 UE 导入使用。
- `fallbacks` 顺序即运行时查找顺序。
- schema 版本化；未知字段由平台脚本忽略。

### 13.3 Unity TMP 深度集成

UPM 包：

- `package.json`：
  - name：`com.neko233.fontcreator233.generated.<familyslug>`
  - displayName：`<Family> <Style> (fontcreator233)`
  - version：recipe 版本
  - unity：`2022.3`
  - 不声明 TMP 依赖版本，避免 Unity 6 与 2022 的包名差异；README 明确要求安装 TMP（2022：`com.unity.textmeshpro`；Unity 6：uGUI 自带 TMP）。
- `Editor/FontCreator233TMPImporter.cs`：
  - 菜单：`Tools/Fontcreator233/Create TMP Font Assets`、`Create Static TMP Font Assets`、`Wire Fallback Chain`、`Set as Default Font`。
  - 读取清单（`JsonUtility` 兼容的 DTO，避免引入 Newtonsoft）。
  - 从 `Fonts/*.ttf` 找到 Unity `Font` 资产。
  - 对每个字体调用 `TMP_FontAsset.CreateFontAsset(font, samplingPointSize, padding, GlyphRenderMode.SDFAA, atlasWidth, atlasHeight, populationMode, multiAtlas)`。
  - `AtlasPopulationMode.Dynamic`：运行时按需光栅化，适合桌面/主机；默认。
  - `AtlasPopulationMode.Static`：读取 `staticCharsets`，`TryAddCharacters`，报告缺字；若图集满则启用多图集并继续；适合移动端可预测内存。
  - 创建/更新 `Assets/FontCreator233Generated/<Family>/<Style>/` 下的 TMP_FontAsset；把 atlas texture 与 material 作为 sub-asset 写入。
  - 按清单顺序写入 `fallbackFontAssetTable`；主字体在前，CJK 分片依次在后。
  - 幂等：重复执行时更新生成目录内的资产；不触碰目录外资产。若目标已存在且不是本工具生成，弹出确认，不静默覆盖。
  - 不修改 `TMP_Settings.defaultFontAsset`；另有菜单项“Set as Default”需二次确认。
  - 记录 sample text 覆盖检查结果；缺失字形以 `Debug.LogError` 输出具体字符与码位。
- 覆盖检查（内置于 Importer）：导入完成后对清单中的每种文字抽样调用 `HasCharacter`，缺失字形以 `Debug.LogError` 输出具体字符与码位；同时输出图集数量、material 与 fallback 链摘要。
- 静态图集与分片：
  - `output.splitCJK: N`（或 `--split-cjk N`）把 CJK 码位按 N 个一组切分，生成 `Family-CJK-01.ttf`、`Family-CJK-02.ttf`……每个子字体包含完整拉丁/希腊/西里尔基础字形 + 当前 CJK 分片。
  - 每个子字体有独立 cmap；TMP fallback 链按清单顺序查找，不会重叠。
  - Web 侧同样生成 `unicode-range` 拆分，浏览器自动按字符选择文件。
  - 默认不切分（`0`），因为 TMP 动态多图集能处理三万字形；移动端建议 `4096`。
- 版本兼容：
  - Unity 2022.3 + TMP 3.x；Unity 6000.6 + uGUI/TMP。
  - 脚本放在 `Editor/` 预定义程序集，不写 asmdef，不依赖包 GUID。
  - 脚本使用 `TMPro` 命名空间；未安装 TMP 的项目会得到明确编译错误，README 给出安装步骤。这是显式依赖，不做反射降级。
- 生成即兼容（不做 Unity 测试）：
  - C# 只使用 TMP 3.x 与 Unity 6 共同存在的 API；`TMP_FontAsset.CreateFontAsset`、`AtlasPopulationMode`、`GlyphRenderMode`、`fallbackFontAssetTable`、`TryAddCharacters`、`HasCharacter` 均在两个版本可用。
  - 生成器做静态断言：C# 文件存在、关键 API/菜单字符串存在、package.json 可解析、manifest schema 正确、charset 文件覆盖预期字符。
  - 不安装、不启动 Unity；README 给出用户在 Unity 2022.3 / Unity 6 中手动导入的步骤与预期结果。
  - 若用户环境未安装 TMP，Editor 脚本会得到明确编译错误；README 说明这是显式依赖。

### 13.4 Unreal Engine

- UE 直接导入 TTF 为 Font 资产；CJK 需要 CompositeFont 的 sub-font ranges。
- `unreal/Scripts/fontcreator233_import.py`（Unreal Editor Python）：
  - 读取清单，调用 `unreal.AssetImportTask` + `unreal.FontFactory` 导入每个 TTF 到 `/Game/Fonts/<Family>/`。
  - 若当前 UE Python API 暴露 `CompositeFont` 创建能力，则创建并写入 default/sub-font ranges；否则打印完整的手工配置指引（ranges 来自清单），并明确说明“自动 CompositeFont 需 UE 5.x Python API 支持”。
  - 所有路径可配置；不覆盖已有资产，除非 `--force`。
- README 说明：UE 的字体缓存、CompositeFont fallback、打包时 Cook 字体资产。
- 本机未安装 UE，无法做真实 Editor 验证；测试只覆盖清单、ranges、脚本语法与文本指引。文档明确这一边界。

### 13.5 Web

- `web/font.css`：

  ```css
  @font-face {
    font-family: "Aurora Grotesk";
    src: url("./AuroraGrotesk-Regular.woff2") format("woff2"),
         url("./AuroraGrotesk-Regular.woff") format("woff"),
         url("./AuroraGrotesk-Regular.ttf") format("truetype");
    font-display: swap;
    font-weight: 100 900; /* 仅声明实际静态字重，不冒充可变字体 */
  }
  ```

  - 不生成 WOFF2 文件；若存在同名 `.woff2`（外部后处理），CSS 优先引用。文档给出 `woff2_compress` 可选步骤。
  - 分片时同一 `font-family` 多个 `@font-face`，各自带 `unicode-range`；主文件在前，CJK 分片在后。
- 样张 HTML 与 Web 包共用字体；HTML 以 TTF 为主，避免 WOFF 编码问题。
- 用浏览器检查 `document.fonts.check`、`document.fonts.status` 与 `font-family` 实际加载；开发期用内置浏览器截图复核。

### 13.6 系统

- 输出标准 TTF；name 表包含 family/subfamily/full/PostScript/version/license。
- `system/INSTALL.md`：
  - Windows：右键安装或复制到 `%WINDIR%\Fonts`（需权限），必要时刷新字体缓存。
  - macOS：双击 Font Book，或复制到 `~/Library/Fonts`。
  - Linux：复制到 `~/.local/share/fonts`，运行 `fc-cache -f -v`。
- 校验：
  - 开发/CI 用 `fontTools.ttLib.TTFont` 加载并遍历所有表；用 `fc-scan`（Linux CI 安装 fontconfig）检查 family/style/charset。
  - 测试输出 `fontcreator233 validate-font` 结果与实际字体一致。

### 13.7 CLI 与清单输出

```bash
fontcreator build recipe.yaml --target all --split-cjk 4096
fontcreator build recipe.yaml --target unity --no-open
fontcreator inspect dist/AuroraGrotesk-Regular.ttf --json
```

`build --json` 增加：

```json
{
  "targets": {
    "universal": ["dist/AuroraGrotesk-Regular.ttf"],
    "unity": ["dist/unity/package.json", "dist/unity/Editor/FontCreator233TMPImporter.cs"],
    "web": ["dist/web/font.css"],
    "system": ["dist/system/INSTALL.md"],
    "unreal": []
  },
  "split": {"count": 4, "fonts": ["…-CJK-01.ttf", "…-CJK-02.ttf"]}
}
```

MCP `fontcreator_build` 增加 `targets: string[]` 与 `splitCJK: number` 参数。

## 14. CLI 体验、并行、工作区与导入导出

### 14.1 TUI 与输出模式

使用 Bubble Tea + Bubbles + Lip Gloss 构建交互式构建界面：

```text
┌ fontcreator233 ─ Aurora Grotesk / Regular ───────────────────────────┐
│ coverage common · jobs 8 · out ./dist                                │
│                                                                      │
│ ● recipe         done      12ms                                      │
│ ● latin/greek    ████████████████████ 100%   312 glyphs              │
│ ● cyrillic       ████████████████████ 100%    70 glyphs              │
│ ◐ cjk            ████████████░░░░░░░░  62%  13000/20976  ETA 2.1s    │
│ ○ compile        waiting                                             │
│ ○ targets        waiting                                             │
│                                                                      │
│ now: U+4E2D 中 · workers 8 · elapsed 4.2s                            │
│ q cancel · ? help · v verbose                                        │
└──────────────────────────────────────────────────────────────────────┘
```

- 总体进度 + 分阶段进度 + 当前码位 + workers + ETA + 最近日志。
- 完成页：字体/样张路径、SHA-256、glyph 数、覆盖率、targets、warnings、耗时；`o` 打开样张，`q` 退出。
- 多字体构建（`build-all`）显示每个字体一行进度，底部为聚合进度。
- `q` / Ctrl+C 取消：context 取消 worker、清理 staging、保留旧产物，退出码 130。
- 输出模式自动选择：
  - 交互 TTY → TUI（写 stderr）。
  - 非 TTY / `CI=true` / `--plain` / `NO_COLOR` → 纯文本阶段行，无 ANSI。
  - `--json` → stdout 只输出最终 JSON；进度静默或 `--progress jsonl` 输出到 stderr。
  - `--quiet` → 只输出错误与最终结果。
- 颜色与宽度：`NO_COLOR`、`TERM=dumb`、窄终端降级；Windows 使用 ANSI 支持，必要时启用 virtual terminal。
- 测试：直接测试 Bubble Tea model 的 Update/View，固定 color profile；`--plain` 做 golden；TUI 不实际进入 alt screen。

### 14.2 并行构建

- 通用 `parallel.Map`：有界 worker、context 取消、索引保序、panic 捕获转为 error。
- 单字体内部并行：
  - 生成字形（拉丁/希腊/西里尔、NFD 组合、SVG override、CJK/Hangul 派生）按码位分块。
  - `glyf`/`loca` 分块编码后按索引拼接。
  - split 子字体并行构建。
- 多字体并行：`fontcreator build-all --dir examples --font-jobs 3`，每个字体一个构建任务；总 worker 预算为 `--jobs`，避免超订阅。`build-all` 只获取一次 workspace 锁，所有字体写入同一 staging，最后一次性 commit，避免同一输出目录内的锁竞争与半成品。
- 进度聚合：原子计数 + 每 50ms 或 1% 节流推送 TUI，避免消息风暴。
- 确定性：
  - 结果按任务索引汇总，不依赖完成顺序。
  - 所有 map 迭代先排序；随机只来自 seed。
  - `--jobs 1` 与 `--jobs 8` 的产物 SHA-256 必须一致（测试）。
- `--jobs N`：默认 `min(NumCPU, 16)`；`--jobs 1` 用于调试/复现。`--font-jobs N`：默认 `min(NumCPU, recipe 数)`。
- 内存保护：默认 worker 上限；超大 `full` 覆盖时按分片流式处理，避免同时持有所有轮廓。
- 进度/错误信息包含码位，便于定位。

### 14.3 工作区状态与自动清理

输出目录下维护：

```text
dist/.fontcreator233/
├─ state.json       # 版本化清单：managed files、builds、assets、hash
├─ history.jsonl    # 构建历史（可裁剪）
├─ tmp/             # staging；失败/过期自动清
└─ lock            # 单写者锁（PID + 时间 + host）
```

`state.json` 记录每个生成文件的 family/style/target/buildId/SHA-256/mtime。构建流程：

1. 获取锁；若锁存在且进程已死或超时（30 分钟）则自动回收，否则报“另一个构建正在进行”。
2. 自动清理：
   - `tmp/` 中超过 24 小时的 staging；
   - 超过 30 天或超过 100 条的 history；
   - 同一 family/style 上一次构建留下、而本次不再生成的文件（例如删除的 target、旧 split 分片）；
   - 不再被任何构建引用的旧 assets 版本。
3. 在 staging 中生成全部产物并校验。
4. 将同族旧文件移到备份，移入新文件，失败时回滚。
5. 原子更新 state/history；释放锁。

安全规则：

- 只删除 state 中记录、且当前 SHA-256 与记录一致的文件。
- 用户修改过的生成文件：跳过并 warning，除非 `--force`。
- 未跟踪文件永不删除。
- 拒绝 `..`、绝对路径逃逸、symlink 指向输出目录之外。
- `--no-auto-clean` 关闭自动清理（调试/法务保留）。
- `--keep-history N` 控制历史条数。

### 14.4 `clean` / `status`

```bash
fontcreator status --out dist --json
fontcreator clean --out dist --dry-run
fontcreator clean --out dist --yes
fontcreator clean --out dist --target unity --yes
fontcreator clean --out dist --older-than 7d --yes
fontcreator clean --out dist --all --yes
```

- `status`：managed 文件数/总大小、各 family/style、历史构建、staging、孤儿文件、被修改文件、可直接回收的空间；`--json` 供 agent 使用。
- `clean` 默认删除：
  - staging/tmp 残留；
  - 过期 history；
  - 不再被引用的旧 assets；
  - 可选 `--target` / `--older-than` 筛选的 managed 产物；
  - `--all` 删除全部 managed 产物与 state（保留用户文件）。
- 默认 dry-run？不：`clean` 是显式命令，执行删除，但 TTY 下要求确认；`--yes` 跳过确认，CI/JSON 模式必须显式 `--yes`。`--dry-run` 永远安全。
- 删除后清理空目录（仅限 managed 目录）。
- 输出 JSON 列出 deleted/skipped/modified 文件与原因。

### 14.5 导入与导出

#### 导入字体（profile + 字库）

```bash
fontcreator import font MyFont.ttf --out work/myfont --extract-glyphs --json
fontcreator import font MyFont.ttf --out work/myfont --unicodes 0041-007A,4E00-9FFF
```

- 解析 TTF/OTF（sfnt），读取：
  - name（family/subfamily/full/PostScript/version/license）；
  - `head.unitsPerEm`（非 1000 时缩放提示）、`hhea`/`OS/2` 的 ascender/descender/lineGap；
  - `OS/2.usWeightClass`、`usWidthClass`、`post.italicAngle`；
  - `OS/2.sxHeight`、`sCapHeight`（缺失时用 `x`、`H` 字形 bounds 估算）；
  - cmap 覆盖、glyph 数、表清单。
- 生成 `profile.json` 与一份 `recipe.yaml` 草稿：
  - weight/width/slant/metrics 直接映射；
  - coverage 按实际 cmap 标注；
  - `style.serif`/`contrast` 用保守启发式并标注 `confidence: low`，由 agent 复核；
  - 许可为 `UNKNOWN` 时禁止把导入字体轮廓打进 bundle，只允许本地实验；README 明确。
- `--extract-glyphs`：把指定码位导出为 SVG（字体单位、y-up、`data-advance`），写入 `<out>/glyphs/`，并在 recipe 中设置 `glyphs.overrides`。
- 不做“从商业字体一键重建同品质字体”的承诺；导入是 remix/审阅工具。

#### 导出字形

```bash
fontcreator export glyphs dist/AuroraGrotesk-Regular.ttf --dir out/glyphs --unicodes 0041-007A
fontcreator export glyphs --recipe examples/aurora-grotesk.recipe.yaml --dir out/glyphs --scripts latin
```

- 从字体或 recipe（内存构建）导出 SVG，文件名 `U+0041_A.svg`，含 `data-advance`。
- 支持 `--unicodes`、`--scripts`、`--limit`；`--json` 输出清单与数量。
- 导出的目录可直接作为 `glyphs.overrides`，形成“导出 → 编辑 → 导回”闭环。

#### Bundle 导入/导出

```bash
fontcreator export bundle examples/aurora-grotesk.recipe.yaml --out dist/aurora.bundle.zip --target all --json
fontcreator import bundle dist/aurora.bundle.zip --out work/aurora --json
fontcreator import bundle dist/aurora.bundle.zip --out work/aurora --build
```

- bundle 是 zip，内含：
  - `bundle.json`（schema/version/recipe/manifest/build summary）
  - `recipe.yaml`
  - `fonts/`、`specimen/`、`targets/`
  - `glyphs/`（可选）
  - `licenses/`、`README.md`
- 导出前可选择重新构建或使用现有产物；默认重新构建以保证一致。
- 导入安全：
  - 拒绝路径穿越、绝对路径、symlink、超过 512 MB 或单文件超过 256 MB 的 zip；
  - 校验 `bundle.json` schema、文件 SHA-256、许可字段；
  - 缺失基础字库许可/OFL 文本时拒绝导入 bundle 中的派生字体；
  - `--build` 用 recipe 重新构建；否则只恢复文件。
- 幂等：导入到非空目录时默认拒绝；`--force` 合并（仍不删用户文件）。
- 测试：导出→导入 roundtrip、zip slip、hash 不匹配、许可缺失、超大文件。

### 14.6 命令总览（新增）

```text
fontcreator build <recipe> [--target ...] [--jobs N] [--no-tui] [--json]
fontcreator build-all --dir examples [--font-jobs N] [--jobs N]
fontcreator clean [--out dist] [--dry-run] [--all] [--yes] [--json]
fontcreator status [--out dist] [--json]
fontcreator import font <ttf> --out <dir> [--extract-glyphs] [--json]
fontcreator import bundle <zip> --out <dir> [--build] [--json]
fontcreator export glyphs <font|--recipe> --dir <dir> [--unicodes ...] [--json]
fontcreator export bundle <recipe> --out <zip> [--target ...] [--json]
```

## 15. Agent 原生集成

### 15.1 CLI JSON 契约

所有关键命令支持 `--json`：

```json
{
  "ok": true,
  "command": "build",
  "font": "dist/AuroraGrotesk-Regular.ttf",
  "woff": "dist/AuroraGrotesk-Regular.woff",
  "specimen": "dist/AuroraGrotesk-Regular.html",
  "family": "Aurora Grotesk",
  "style": "Regular",
  "sha256": "…",
  "bytes": 123456,
  "glyphs": 28640,
  "coverage": {
    "latin": 312,
    "greek": 78,
    "cyrillic": 70,
    "cjk": 20976,
    "kana": 192,
    "hangul": 11172,
    "symbols": 512
  },
  "warnings": ["CJK weight is synthetic"],
  "targets": {
    "universal": ["dist/AuroraGrotesk-Regular.ttf"],
    "unity": ["dist/unity/package.json", "dist/unity/Editor/FontCreator233TMPImporter.cs"],
    "web": ["dist/web/font.css"],
    "system": ["dist/system/INSTALL.md"]
  },
  "durationMs": 4210
}
```

错误：

```json
{
  "ok": false,
  "error": {
    "code": "invalid_recipe",
    "message": "style.weight must be between 100 and 900",
    "details": [{"field": "style.weight", "value": 1200}]
  }
}
```

退出码：0 成功；1 用户/recipe 错误；2 内部错误。所有日志写 stderr，stdout 只放 JSON/结果。

### 15.2 `describe --json`

一次调用返回 agent 需要的全部信息：

- CLI 版本、支持的 apiVersion。
- 命令列表与参数。
- 内联 JSON Schema（recipe）。
- 所有枚举值及含义。
- 参数 → 视觉效果的映射指南。
- 覆盖率选项与语言清单。
- 已支持/不支持的文字系统。
- 3 个示例 recipe 的路径与摘要。
- 字库覆盖与 `library export` 用法。
- 目标平台清单、`--target` 取值、Unity TMP 集成参数与 UE CompositeFont 说明。
- TUI/plain/json 输出模式、`--jobs`/`--font-jobs` 并行参数、`--no-open`/`--no-tui`。
- 工作区 `status`/`clean` 语义、自动清理规则与安全边界。
- 导入/导出命令、bundle schema、字形 SVG 格式与许可警告。
- 许可规则（派生 CJK 时输出必须 OFL）。
- 推荐工作流（看参考图 → 写 recipe → validate → build → specimen → inspect → 迭代）。

`describe --markdown` 输出同样内容的 markdown，供 agent 直接阅读。

### 15.3 MCP stdio

`fontcreator mcp` 实现最小 MCP server（newline-delimited JSON-RPC 2.0，协议版本 `2025-06-18`）：

- `initialize` / `notifications/initialized` / `ping`
- `tools/list`
- `tools/call`

工具：

| 工具 | 输入 | 输出 |
|---|---|---|
| `fontcreator_describe` | `{}` | 契约 JSON |
| `fontcreator_validate` | `{recipe: string, filename?: string}` | 校验结果 |
| `fontcreator_build` | `{recipe: string, filename?: string, outDir: string, targets?: string[], splitCJK?: number, jobs?: number, open?: boolean}` | 构建结果 + 路径 + 覆盖 + 平台包 |
| `fontcreator_inspect` | `{fontPath: string}` | 字体元数据/覆盖/表 |
| `fontcreator_specimen` | `{fontPath: string, outPath?: string, open?: boolean}` | 样张路径 |
| `fontcreator_library_list` | `{}` | 内置字库清单 |
| `fontcreator_status` | `{outDir: string}` | 工作区 managed/stale/可回收空间（只读） |

安全：只接受文件内容与路径，不执行 shell；`open` 默认 false。

### 15.4 Agent skill 与文档

- `AGENTS.md`：项目根部的 agent 入口，包含最短工作流、命令示例、JSON 约定、禁止事项。
- `agent-skill/fontcreator233/SKILL.md`：可被 agent 直接加载的 skill，带 YAML frontmatter。
- `docs/agent-workflow.md`：
  - 如何看参考图：字重、对比度、衬线、宽度、端点、情绪、种子。
  - 3 个“参考图 → style brief → recipe”案例。
  - 迭代循环：最多 3 轮，先结构后审美。
  - 何时使用 SVG override。
  - 禁止：声称不支持的脚本；忽略 warning；在 CI 打开浏览器。
- `describe` 输出的映射指南与文档保持同步（由同一份 Go 数据结构生成，测试断言关键字段存在）。

## 16. 字库能力

### 16.1 内置骨架字库

- `fontcreator library list --json`：按 script 列出内置字形数、变体、可覆盖码位。
- `fontcreator library export --dir out --recipe recipe.yaml`：把当前风格下的所有生成字形导出为 SVG（y-up，viewBox `0 -250 1000 1250`），文件名 `U+0041_A.svg`。
- SVG 带 `data-advance`，便于再导入。

### 16.2 SVG 覆盖

`glyphs.overrides` 指向目录，支持：

- 文件名：`U+4E2D.svg`、`uni4E2D.svg`、`A.svg`。
- `<svg viewBox="0 -250 1000 1250" data-advance="600">`，坐标系为字体单位（y 向上，baseline=0）。
- `d` 支持 `M/L/H/V/C/S/Q/T/Z`（含相对命令）；不支持弧线 `A`，报清晰错误。
- 覆盖在生成与派生之后应用，优先级最高。
- 缺 `data-advance` 时用包围盒 + 50 单位侧距推算，并产生 warning。

## 17. 许可与合规

- 代码：MIT。
- 内嵌 Noto Sans SC/KR 的静态子集：SIL OFL 1.1，提交原始 `OFL.txt`、来源、版本、下载 URL 与生成脚本；改名以遵守 Reserved Font Name。
- 生成字体：
  - 使用基础字形（`coverage.base` 非空）→ 输出**必须是 OFL-1.1**；`meta.license` 若填其他值，校验直接报错。
  - 不使用基础字形 → 可用 recipe 指定许可，默认 OFL-1.1。
- 字体 name 表写入基础字体版权与 OFL 许可链接。
- 不得使用 `Noto` 作为生成字体名称；校验拒绝 `Noto*` family，避免商标混淆。
- 仓库不包含任何来源不明的商业字体。
- README 明确说明：输出字体若派生自 Noto，整体按 OFL 分发。

## 18. 可复现性与性能

- `head.created/modified` 取 `SOURCE_DATE_EPOCH`，否则固定为 `2026-01-01T00:00:00Z`。
- 所有“随机”只来自 `seed` 与码位的确定性 hash。
- 构建顺序固定；并行生成时按码位排序后写表。
- **字节一致的边界**：相同 fontcreator233 版本、相同 OS/arch、相同 Go 版本下，同一 recipe 的 TTF/WOFF 字节一致（`--jobs 1` 与 `--jobs 8` 也一致）。跨 OS/arch/Go 版本不承诺字节一致，只承诺语义/覆盖一致；原因是浮点三角函数与 zlib 的实现差异。CI 在每个平台内部做重复构建哈希测试，不做跨平台哈希对比。
- 目标性能（现代 4–8 核，SSD）：
  - `sample` 覆盖：< 1 s
  - `common` 覆盖（约 4 万字形：CJK + Hangul + 生成脚本）：< 12 s
  - `full` 覆盖：< 30 s
  - `--jobs` 从 1 到 8 应有明显加速；`--jobs 1` 用于复现，不追求速度。
- 基础字体只解析一次；`derive` 用 `sync.Once` 缓存；worker 各自持有 `sfnt.Buffer`。
- 并行编码 `glyf`/`loca`，顺序组装；CV 结果按索引归并。
- 输出目录已存在时通过 workspace 管理：只替换/删除自己生成且未修改的文件，绝不删除用户其他文件。

## 19. 测试策略

### 19.1 单元测试

- recipe：默认值、边界、未知字段、许可规则、文件名清理。
- geom：变换、面积/方向、采样误差、轮廓外扩、抖动确定性。
- skeleton：每个内置码位都能生成非空轮廓、advance > 0（组合标记除外）、无 NaN、点在合理范围。
- compose：é/ñ/Ω tonos/Ё 等 NFD 组合。
- fontbuild：表长度、checksum、cmap format 4/12 编码、name 表 UTF-16BE、OS/2 字段。
- SVG override：路径解析、advance、覆盖优先级、错误提示。
- specimen：HTML 转义、相对路径、主题色、复制 CSS。
- open：三平台命令构造，注入 opener，不真启动。
- integration：清单 schema、路径相对性、sha256、fallback 顺序、charset 编码、Web unicode-range 生成。
- targets：Unity package.json/C# 关键 API/菜单/字符集；UE Python 脚本与 ranges；system INSTALL.md；JSON 输出中的 target 路径。
- split：按 CJK 分片后每个子字体 cmap 不重叠、合起来覆盖全集、fallback 顺序正确、Web CSS 多 `@font-face`。
- ui：Bubble Tea model 的阶段更新/进度/取消/窄终端/NO_COLOR；plain reporter 输出；JSON 模式 stdout 不含进度。
- parallel：保序、取消、panic 转 error、worker 上限、进度节流；`go test -race`。
- workspace：lock 获取/回收、staging 清理、同族 stale 删除、用户文件/修改文件保护、state 原子更新、路径逃逸拒绝。
- importexport：TTF profile 字段、metrics 推断、SVG 导出/导入、bundle 打包/解包、zip slip/超大/篡改/许可缺失拒绝。
- specimen assets：无 `http(s)://` 资源引用、无 ES module、glyph data JS 可解析、虚拟列表初始化、资源 hash/许可文件齐全。

### 19.2 集成测试（核心验收）

`TestBuildThreeStyles` 在临时目录构建 3 套风格：

| 风格 | recipe 特征 | 断言 |
|---|---|---|
| Aurora Grotesk | 几何无衬线、weight 620、width 104、round terminal | weightClass=620；无衬线；国际化覆盖 |
| Noir Didone | 高对比 hairline 衬线、weight 500、slant 8°、width 94 | italicAngle≈-8；serif hairline；对比度高 |
| Botanic Display | Art Nouveau、weight 560、contrast .62、flare、roughness .18、seed 233 | 轮廓与非抖动版本不同；可复现 |

每套字体都断言：

- 能被 `golang.org/x/image/font/sfnt` 解析。
- cmap 覆盖：`A`、`a`、`0`、`é`、`ñ`、`Ω`、`Ж`、`中`、`あ`、`한`、`€`、`→`、`“”`。
- 至少 30,000 个非空字形（`common` 覆盖：SC CJK/假名/符号 + KR Hangul）。
- 必需表齐全；`head.magic` 正确；advance 为正（space 除外）。
- `meta.family`、`meta.style`、版本、许可出现在 name 表。
- 样张存在，包含 `@font-face`、family、中文/日文/韩文样例行。
- 同 recipe 构建两次 SHA-256 相同。
- 3 套字体 SHA-256 互不相同，weightClass/italicAngle/风格签名不同。
- 构建 JSON 中 coverage 计数与实际情况一致。

`TestGlobalCoverage`：对每套字体逐码位检查代表字符集，缺字给出可读错误。

`TestPlatformBundles`：对 3 套字体分别生成 `--target all`，断言：

- `web/font.css` 含 `@font-face`、WOFF/TTF 回退、正确 family、`font-display: swap`。
- `system/INSTALL.md` 存在，字体副本与 universal 字体 SHA-256 一致。
- `unity/package.json` 可解析，`unity/Editor/FontCreator233TMPImporter.cs` 含 `TMP_FontAsset.CreateFontAsset`、fallback 写入、静态字符集路径。
- TMP 字符集文件包含拉丁/希腊/西里尔/常用 CJK 样本。
- `unreal/Scripts/fontcreator233_import.py` 语法可被 Python 解析（`python -m py_compile`，可选；Go 测试只做文本断言），清单 ranges 正确。
- 集成清单 schema 与字体实际 cmap 一致。

`TestSplitCJK`：`--split-cjk 256 --coverage sample` 生成多个子字体；断言每个子字体可独立解析、cmap 不重叠、合并覆盖全集、TMP fallback 顺序与 Web unicode-range 正确。

`TestUnityImporterText`：对生成 C# 做关键字断言（菜单路径、`GlyphRenderMode.SDFAA`、`AtlasPopulationMode.Dynamic/Static`、`fallbackFontAssetTable`、`TryAddCharacters`、`HasCharacter`）。

`TestFontToolsValidation`（可选，环境有 Python/fontTools 时运行）：用 `fontTools.ttLib.TTFont` 加载每套字体，遍历表；失败则测试失败，缺失则 `t.Skip`。

`TestMCP`：通过管道发送 `initialize` / `tools/list` / `tools/call(fontcreator_describe)` / 一次 `fontcreator_build`，断言 JSON-RPC 响应与服务端未崩溃。

`TestCLIJSON`：`validate` / `build` / `inspect` / `library list` / `library export` 的 JSON 信封与退出码；`--json` 时 stdout 必须是纯 JSON。

`TestParallelDeterminism`：同一 recipe 分别 `--jobs 1`、`--jobs 4`、`--jobs 8` 构建；3 个 SHA-256 完全一致；进度事件不回退。

`TestWorkspaceCleanup`：

- 两次构建删除某个 target 后，旧 managed 文件被自动清理。
- 手工创建/修改的文件不被删除，`--force` 才删除修改过的 managed 文件。
- `clean --dry-run` 不改变磁盘；`clean --yes` 删除；`--all` 保留用户文件。
- 伪造过期 lock/tmp 被自动回收；活跃 lock 阻止第二个构建。
- 路径逃逸/symlink 被拒绝。

`TestImportExportRoundtrip`：

- 用生成字体 `import font`，断言 weight/italic/metrics/coverage 与源字体一致。
- `export glyphs` 导出后用 `glyphs.overrides` 重新构建，指定字形轮廓哈希一致。
- `export bundle` → `import bundle` → 重新构建，字体 SHA-256 与 bundle 内清单一致。
- 损坏 zip、zip slip、hash 不匹配、缺 OFL 文本均报错。

`TestSpecimenAssets`：构建后 `assets/` 含 CSS/JS/Geist/TanStack/许可；HTML 引用的所有本地文件存在；没有任何外链脚本/样式/字体；`*.glyphs.js` 包含 `window.FontCreatorData`；HTML 不含 `type="module"`。开发期用内置浏览器以 `file://` 打开样张，检查控制台无错误、字体已加载、虚拟列表只渲染可视行、滚动/搜索正常并截图留档。

`TestBuildAllParallel`：`build-all --dir examples --font-jobs 3` 并行构建 3 套字体，全部成功且与逐个构建的 SHA-256 一致。

### 19.3 测试数据

- 3 份示例 recipe（`examples/*.recipe.yaml`）。
- 1 个 SVG override 示例（`examples/glyphs/U+2605.svg`）。
- 测试不依赖网络。
- 大字体产物写入 `t.TempDir()`，不提交。
- `go test -short` 跳过 `full` 覆盖与 3 风格全量构建，仅跑单元测试；CI 跑完整测试。
- Unity 集成不做 Editor 测试；由 `TestUnityImporterText` 与 `TestPlatformBundles` 做文件级断言，用户按 `unity/README.md` 在 Unity 中手动导入。

## 20. 仓库结构

```text
.
├─ main.go
├─ go.mod / go.sum
├─ internal/
│  ├─ cli/
│  ├─ recipe/
│  ├─ style/
│  ├─ geom/
│  ├─ skeleton/
│  ├─ compose/
│  ├─ library/
│  ├─ derive/
│  ├─ fontbuild/
│  ├─ specimen/
│  │  └─ webassets/       # CSS/JS、vendor TanStack、Geist、licenses（go:embed）
│  ├─ integration/
│  ├─ targets/
│  │  ├─ unity/          # package.json/C# 模板、字符集写出
│  │  ├─ unreal/         # Python 脚本与 README 模板
│  │  ├─ web/            # font.css
│  │  └─ system/         # INSTALL.md
│  ├─ ui/
│  ├─ parallel/
│  ├─ workspace/
│  ├─ importexport/
│  ├─ agentschema/
│  └─ mcp/
├─ assets/
│  ├─ base/Fontcreator233BaseSC-Regular.ttf
│  ├─ base/Fontcreator233BaseKR-Regular.ttf
│  └─ base/OFL.txt
├─ examples/
│  ├─ aurora-grotesk.recipe.yaml
│  ├─ noir-didone.recipe.yaml
│  ├─ botanic-display.recipe.yaml
│  └─ glyphs/U+2605.svg
├─ docs/
│  ├─ agent-workflow.md
│  ├─ platforms.md
│  ├─ cli.md
│  ├─ web-specimen.md
│  └─ superpowers/specs/2026-11-17-fontcreator233-design.md
├─ scripts/
│  ├─ prepare-base-fonts.py   # fonttools：实例化/子集化/改名
│  ├─ vendor-web-assets.ps1   # 重新生成 TanStack IIFE、复制 Geist
│  ├─ test-web.mjs            # 可选 Playwright/浏览器校验
│  └─ validate-fonts.py
├─ agent-skill/fontcreator233/SKILL.md
├─ AGENTS.md
├─ README.md
├─ README.zh-CN.md
├─ LICENSE
├─ NOTICE.md
├─ THIRD_PARTY_NOTICES.md
├─ .gitignore
└─ .github/workflows/ci.yml
```

## 21. CI 与 GitHub 发布

- `ci.yml` 矩阵：ubuntu-latest、windows-latest、macos-latest。
- 步骤：checkout → setup-go 1.27 → `gofmt -l` 检查 → `go vet ./...` → `go test ./...` → `go test -race ./internal/parallel ./internal/workspace`（Ubuntu）→ `go build ./...`。
- Ubuntu job 额外：
  - `apt-get install fontconfig`，用 `fc-scan` 校验 3 套字体；
  - `pip install fonttools`，运行 `scripts/validate-fonts.py`；
  - 构建 `--target all` 产物，断言样张无外链、本地 assets 齐全；`actions/upload-artifact` 上传字体、样张（含 assets）、Unity/UE/Web/System 包，保留 14 天；
  - `scripts/test-web.mjs` 可选：若安装 Playwright/Chromium，则以 `file://` 打开样张，验证字体加载、虚拟列表、无控制台错误；否则跳过。
- 测试设置 `CI=true`，验证不会自动打开浏览器。
- CI 增加 `git ls-files dist out work build tmp | test -z`，确保生成物未入库；`.gitignore` 强制保留 `assets/base` 与 `internal/specimen/webassets`。
- 不创建 Unity CI workflow；Unity 集成只做文件级断言。
- 发布：`gh repo create neko233-com/fontcreator233 --public --source . --remote origin --push`。
- 设置 repo description 与 topics：`golang`, `font`, `typography`, `cli`, `mcp`, `ai-agent`, `font-generator`。
- 推送后检查 CI 状态；失败则修复再推。

## 22. 验收标准

用户批准后，以下全部成立才算完成：

1. `go test ./...` 在本机全绿。
2. `go vet ./...`、`gofmt -l` 无问题。
3. `go install` 后可以直接运行 `fontcreator`。
4. 3 条命令各自构建一套风格字体、样张与平台包：

   ```bash
   fontcreator build examples/aurora-grotesk.recipe.yaml --target all
   fontcreator build examples/noir-didone.recipe.yaml --target all
   fontcreator build examples/botanic-display.recipe.yaml --target all
   ```

5. 3 个 `.ttf` 均能在浏览器样张中加载；样张可在桌面自动打开；`font.css` 可直接使用。
6. `fontcreator inspect <font> --json` 显示全球化覆盖与元数据；fontTools 与 `fc-scan` 通过。
7. `fontcreator describe --json` 可被 agent 直接消费；MCP server 通过测试。
8. Unity UPM 生成并通过文件级断言：package.json 可解析、C# 含 TMP 3.x/Unity 6 共同 API、fallback 顺序正确、charset 覆盖拉丁/希腊/西里尔/CJK/假名/谚文；不安装或运行 Unity。
9. `--split-cjk` 产物在 TMP fallback 与 Web `unicode-range` 中顺序正确、cmap 不重叠、合并覆盖全集。
10. UE 包包含可执行导入脚本、manifest 与 CompositeFont ranges 说明；本机无 UE，边界在 README 中写明。
11. GitHub 仓库 public，CI 绿，产物 artifact（含 3 套平台包）可下载。
12. README 中说明目标平台、能力边界、Unity TMP 使用步骤与 OFL 许可。
13. 在真实终端运行 `fontcreator build ...` 显示 Bubble Tea TUI、总体/阶段进度、ETA、取消可用；`CI=true`/管道/`--json` 自动降级且 stdout 干净。
14. `--jobs 1` 与 `--jobs 8` 产物 SHA-256 相同；`build-all --font-jobs 3` 三套字体并行成功。
15. 样张在断网、`file://` 下打开：Geist/TanStack/本地 CSS/JS 全部加载，无控制台错误，虚拟列表流畅滚动 4 万+ 字形；无任何外链资源。
16. `fontcreator status` 显示工作区状态；`clean --dry-run` 不删除；自动清理只删除同族旧 managed 文件；用户/修改文件安全。
17. `import font`、`export glyphs`、`export bundle`、`import bundle` 完成一次 roundtrip，重建字体与清单 SHA-256 一致。
18. `.gitignore` 生效：`dist/`、`work/`、`out/`、生成字体/样张/平台包均不被 `git status` 跟踪；`assets/base` 与 `internal/specimen/webassets` 正常入库。

## 23. 决策记录

- **外部 agent 做视觉理解**：用户明确选择。
- **混合引擎**：自绘拉丁/希腊/西里尔 + 派生 CJK；平衡风格范围与全球化。
- **内嵌基础字体**：SC + KR 两个静态/子集字体，换取离线与真实 CJK/Hangul 覆盖；接受约 12.7 MB 体积。
- **输出 TTF + WOFF**：浏览器兼容性与可安装性优先；不做 WOFF2。
- **MCP + JSON + skill 三件套**：覆盖不同 agent 集成方式，成本可控。
- **确定性优先**：固定时间戳与 seed，测试 SHA-256。
- **OFL 传染性**：派生 CJK 的输出强制 OFL，避免许可风险。
- **3 套测试风格**：几何无衬线 / 高对比 Didone / 新艺术运动，覆盖主要参数空间。
- **Unity TMP 生成即兼容**：UPM + Editor 导入器 + 动态/静态图集 + fallback + 字符集；只做文件级断言，不安装/启动 Unity 测试。
- **平台包由脚本生成，不预生成版本相关资产**：避免 TMP/UE 资产格式与 Unity/UE 版本绑定。
- **默认输出通用 TTF + Web + System + Unity**；UE 包显式开启，减少无关产物。
- **WOFF2 非目标**：保留 WOFF/TTF；需要时由外部 `woff2_compress` 后处理并自动检测。
- **CJK 可切分**：`--split-cjk` 同时服务 TMP 静态图集和 Web `unicode-range`。
- **Bubble Tea TUI**：交互式体验优先，但所有自动化路径都有 plain/json 降级；TUI 只写 stderr。
- **单字体内部并行 + 多字体并行**：有界 worker、索引保序、jobs=1 可复现。
- **本地 Web 资产而非单文件内联**：复用 Geist/TanStack，`file://` 兼容；单文件 HTML 仅作为可选。
- **保守清理**：state 追踪 + SHA 校验 + 只删同族/过期 managed 文件；自动清理默认开启。
- **导入是 remix/审阅工具**：不承诺从商业字体复制同品质字库；bundle 校验许可。

## 24. 留给构建者的决定

- 骨架坐标的具体审美细节（比例、曲线控制点）。
- 衬线、端点、喇叭口的具体多边形形状。
- 样张的视觉设计（排版、色板、动效），只要符合第 12 节。
- 常用 kern 字对清单。
- 覆盖率 `sample` 集合的具体字符选择。
- 性能优化手段（并行度、缓存），只要满足目标。
- TMP Editor 脚本的 UI 文案、菜单分组、生成目录命名、幂等策略的具体实现。
- TMP 静态图集默认采样大小/边距/图集尺寸的调优。
- UE Python 脚本在 CompositeFont API 不可用时的提示格式。
- TUI 的具体布局、颜色、动画与窄终端断点。
- 进度事件节流的具体阈值与 ETA 算法。
- 自动清理的默认时长/历史条数；state schema 的具体字段。
- 导入 profile 的启发式阈值与 confidence 标注。
- 样张虚拟列表的 DOM 结构与渲染批次大小。
- README 的具体措辞与示例顺序。
