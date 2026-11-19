# fontcreator233

**为 agent 打造的确定性、离线字体创造器。**

你看参考图，fontcreator233 把你的视觉判断变成真正可安装的字体：TTF/WOFF、好看的本地 HTML 样张，Unity TMP、Unreal、Web、系统产物，以及 Photoshop、剪映/CapCut、Premiere、Figma、Office 等应用的使用包。

![fontcreator233 样张](docs/assets/specimen-dark.png)

```bash
fontcreator init --dir work --template geometric --family "Aurora Grotesk"
fontcreator build work/font.recipe.yaml --out dist --target all
```

运行时零联网，二进制里没有大模型，也不需要安装字体编译器。只需要 Go、一份 OFL 基础字库和一份 recipe。

## 特点

- **Agent 原生**：视觉理解交给 agent；CLI 提供严格 JSON、JSON Schema 和 MCP server。
- **确定性**：固定时间戳、seed 驱动随机、排序输出、SHA-256 校验；同一平台 `--jobs 1` 与 `--jobs 8` 字节一致。
- **全球化**：拉丁/希腊/西里尔参数化生成，CJK/假名/谚文/符号来自内嵌 OFL 基础字库；覆盖率如实报告。
- **真实产物**：TTF、WOFF、HTML 样张、AngelCode BMFont `.fnt` + PNG 图集、Unity UPM + TMP 导入器、Unreal 导入脚本 + CompositeFont 范围、Web CSS、系统安装说明、应用使用包。
- **OpenType**：`liga`、`kern`、`case`、`frac`、`tnum`、`ss01`、`smcp`、`onum`、`zero`、`sups`、`subs`、`ccmp`、`calt` 和 `mark`/`mkmk` 锚点，同时生成 GPOS 与 legacy kern 表。
- **可变字重（可选）**：`output.variable: true` 生成一条真实的 `wght` 可变轴（`fvar` + `gvar`），并写入 `HVAR` 字宽与 `MVAR` 度量变化；样张带实时字重滑杆。
- **上下文替换**：`openType.contextual` 通过真实 GSUB chain-context lookup 把任意已存在的字形序列映射到任意已存在的目标字形。
- **草书连写（可选）**：`output.cursive: true` 生成真实 GPOS cursive attachment lookup（`curs`），为小写字母写入 entry/exit 锚点；样张提供开关与连写示例。
- **彩色字形**：`openType.color` 生成 recipe 驱动的多字形 COLR v0 与多套 CPAL 调色板（circle/square/triangle/star/ring 图层），样张带调色板切换。未配置时附带一个 `U+E000` 示例字形。
- **CLI 好看**：Bubble Tea 进度、ETA、取消；非 TTY/CI/JSON 自动降级。
- **工作区安全**：staging + 原子提交；`clean` 只删暂存/历史或自己生成且 SHA-256 未变的文件。

## 安装

```bash
go install github.com/neko233-com/fontcreator233@latest
fontcreator version
```

或：

```bash
git clone https://github.com/neko233-com/fontcreator233
cd fontcreator233
go build -o fontcreator .
```

Go 1.27+，Windows/macOS/Linux，支持 `CGO_ENABLED=0`。

## Agent 工作流

```bash
fontcreator describe --json
fontcreator init --dir work --template geometric --family "My Typeface"
fontcreator validate work/font.recipe.yaml --json
fontcreator build work/font.recipe.yaml --out dist --target all --json --no-open
fontcreator specimen dist/MyTypeface-Regular.ttf --open
fontcreator inspect dist/MyTypeface-Regular.ttf --json
fontcreator learn path/to/reference.ttf --out work/learned --json
```

参考图 → 风格参数的映射表见 [`AGENTS.md`](AGENTS.md)、[`agent-skill/fontcreator233/SKILL.md`](agent-skill/fontcreator233/SKILL.md) 和 [`docs/agent-workflow.md`](docs/agent-workflow.md)。

### MCP

```json
{
  "mcpServers": {
    "fontcreator233": {
      "command": "fontcreator",
      "args": ["mcp"]
    }
  }
}
```

工具：`fontcreator_describe`、`fontcreator_validate`、`fontcreator_build`、`fontcreator_inspect`、`fontcreator_learn`、`fontcreator_specimen`、`fontcreator_library_list`、`fontcreator_status`。

### 学习已有字体

```bash
fontcreator learn path/to/reference.ttf --out work/learned --json
```

会解析 name、metrics、cmap、GSUB/GPOS 特性标签、lookup 类型、legacy kern pair、可变轴、彩色表和抽样字形统计，输出 `font-report.json`、`font-report.md`、`profile.json` 和起始 `recipe.yaml`。生成的 recipe 只是起点，不复制原字体；skeleton、contrast、serif、terminal 仍需结合参考图判断。

## 三套示例字体

| Recipe | 风格 |
|---|---|
| [`aurora-grotesk`](examples/aurora-grotesk.recipe.yaml) | 宽几何无衬线、圆端点、weight 620 |
| [`noir-didone`](examples/noir-didone.recipe.yaml) | 高对比 hairline 衬线、8° 斜体 |
| [`botanic-display`](examples/botanic-display.recipe.yaml) | 喇叭端点、有机抖动、seed 233 |

```bash
fontcreator build examples/aurora-grotesk.recipe.yaml --target all --no-open
fontcreator build examples/noir-didone.recipe.yaml --target all --no-open
fontcreator build examples/botanic-display.recipe.yaml --target all --no-open
```

自动化 E2E 会用 `common` 覆盖构建这三套字体，断言多语言覆盖、风格、确定性、样张和所有平台包。

## Recipe 与全球化

完整字段见 [`README.md`](README.md#recipe) 和 `fontcreator describe --json`。

- 拉丁/希腊/西里尔：参数化骨架 + NFD 组合。
- 中文：内嵌 Noto SC 静态子集（实际 cmap）。
- 日文：假名 + 常用汉字来自内嵌基础。
- 韩文：内嵌 Noto KR Hangul 子集。
- 符号/货币/箭头/标点：基础字库能提供的部分。

`full` 表示“内嵌字库中实际存在的全部字形”，不是整个 Unicode。构建 JSON 的 `coverageReport` 会给出 requested/found/missing，并产生 warning。

不支持：阿拉伯文、希伯来文、印度系文字、泰文、竖排、复杂 shaping。

生成的 OpenType 覆盖 `liga`、`kern`、`case`、`frac`、`tnum`、`ss01`、`smcp`、`onum`、`zero`、`sups`、`subs`、`ccmp`、基础 top-center `mark` 锚点、`mkmk` 堆叠标记、recipe 定义的 `calt` 精确序列规则，以及可选生成的 `curs` cursive attachment（`output.cursive: true`，在真实 GPOS type 3 上为小写字母写入 entry/exit 锚点）；可选生成一条 `wght` 可变轴（`fvar` + `gvar`），并写入 `HVAR` 字宽与 `MVAR` 度量变化，CJK/韩文/假名保持静态，不生成额外轴。`openType.color` 生成多字形、多调色板的 COLR/CPAL；`output.formats` 可生成真正的 WOFF 2.0（Brotli 压缩）。不生成 SVG/sbix/CBDT 彩色表或 emoji；不生成 contextual class、backtrack/lookahead、`cswh` 等高级 GSUB/GPOS；`openType.contextual` 只支持精确序列规则。`learn` 可以解析并报告其他字体中的这些特性。

## 产物

```text
dist/
├─ AuroraGrotesk-Regular.ttf / .woff
├─ AuroraGrotesk-Regular.html
├─ fontcreator233.manifest.json
├─ web/font.css
├─ system/INSTALL.md
├─ apps/           # Photoshop、剪映/CapCut、Premiere、Figma、Office…
├─ unity/          # UPM + TMP 导入器 + 字符集
└─ unreal/         # Font 导入脚本 + CompositeFont ranges
```

- **应用场景包**：`apps/` 包含 Photoshop、Illustrator、InDesign、剪映/CapCut、Premiere、After Effects、DaVinci、Final Cut、Figma、Canva、Office、OBS、Blender、Godot、Unity、Unreal 的说明，以及 `caption.srt`/`caption.ass` 字幕样例。桌面软件安装系统字体；Blender/Godot 直接加载 TTF；移动端剪映按版本可能支持导入或需要渲染图替代。
- **Unity TMP**：`Tools/Fontcreator233/Create TMP Font Assets`、`Create Static TMP Font Assets`、`Wire Fallback Chain`。移动端可用 `--split-cjk 4096`。生成器不启动/测试 Unity。
- **Unreal**：在 Editor Python 中运行 `unreal/Scripts/fontcreator233_import.py`；API 不支持时打印 CompositeFont 范围。生成器不启动/测试 UE。
- **Web**：`web/font.css` 全部本地文件；分片字体带 `unicode-range`。默认生成 WOFF 2.0（Brotli 压缩），无需外部后处理。
- **系统**：`system/INSTALL.md` 覆盖 Windows/macOS/Linux。

![fontcreator233 应用场景包](docs/assets/apps-dark.png)

## 好看、离线的样张

Vercel/Geist 风格：巨大的生成字体 `Aa`、可编辑 type tester（字号/行距/字距）、OpenType 特性开关（liga/kern/frac/case/tnum/ss01）与 smcp/onum/zero/sups/subs/ccmp/mark/mkmk 样张、scale 瀑布、多语言卡片、应用/字幕/海报预览、可搜索/筛选的虚拟化字形网格。

![OpenType 特性面板](docs/assets/opentype-dark.png)

- Geist Sans/Mono（OFL）做 UI。
- TanStack Virtual（MIT）打包为经典脚本，`file://` 直接可开。
- 无 CDN、无 ES module、无 `fetch()`。
- 支持深浅色、recipe 强调色和 `--self-contained-html` 单文件。

详见 [`docs/web-specimen.md`](docs/web-specimen.md)。

## CLI 与清理

```bash
fontcreator build <recipe> [--target all] [--split-cjk N] [--jobs N] [--json]
fontcreator build-all --dir examples [--font-jobs N]
fontcreator validate <recipe> --json
fontcreator inspect <font> --json
fontcreator learn <font> --out work/learned --json
fontcreator specimen <font> --open
fontcreator describe --json | --markdown
fontcreator library list|export
fontcreator import font|bundle
fontcreator export glyphs|bundle
fontcreator status / clean
fontcreator mcp
```

完整命令见 [`docs/cli.md`](docs/cli.md)。

```bash
fontcreator status --out dist --json
fontcreator clean --out dist --dry-run          # 只清理暂存/历史
fontcreator clean --out dist --all --dry-run    # 预览全部产物
fontcreator clean --out dist --all --yes
fontcreator clean --out dist --target unity --yes
```

`clean` 默认拒绝删除未跟踪或被修改的文件，除非 `--force`。

## 测试

```bash
go test ./...
go test -short ./...
go test -race ./internal/parallel ./internal/workspace
```

覆盖几何、recipe 校验、字形生成、TTF/WOFF 编码、cmap/name/OS2、基础字库派生、NFD 组合、并行确定性、工作区清理安全、SVG 往返、bundle、MCP、CLI JSON，以及三套风格的全平台 E2E。CI 还会用 fontTools/fontconfig 交叉校验。

## 许可

代码 MIT。内嵌基础字体及派生字体 SIL OFL 1.1，见 [`NOTICE.md`](NOTICE.md)。Geist（OFL）与 TanStack Virtual（MIT）见 [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md)。
