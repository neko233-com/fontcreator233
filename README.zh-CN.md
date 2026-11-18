# fontcreator233

**为 agent 打造的确定性、离线字体创造器。**

你看参考图，fontcreator233 把你的视觉判断变成真正可安装的字体：TTF/WOFF、好看的本地 HTML 样张，以及 Unity TMP、Unreal、Web、系统四套产物。

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
- **真实产物**：TTF、WOFF、HTML 样张、Unity UPM + TMP 导入器、Unreal 导入脚本 + CompositeFont 范围、Web CSS、系统安装说明。
- **CLI 好看**：Bubble Tea 进度、ETA、取消；非 TTY/CI/JSON 自动降级。
- **工作区安全**：staging + 原子提交；`clean` 只删自己生成且 SHA-256 未变的文件。

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

工具：`fontcreator_describe`、`fontcreator_validate`、`fontcreator_build`、`fontcreator_inspect`、`fontcreator_specimen`、`fontcreator_library_list`、`fontcreator_status`。

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

## 产物

```text
dist/
├─ AuroraGrotesk-Regular.ttf / .woff
├─ AuroraGrotesk-Regular.html
├─ fontcreator233.manifest.json
├─ web/font.css
├─ system/INSTALL.md
├─ unity/          # UPM + TMP 导入器 + 字符集
└─ unreal/         # Font 导入脚本 + CompositeFont ranges
```

- **Unity TMP**：`Tools/Fontcreator233/Create TMP Font Assets`、`Create Static TMP Font Assets`、`Wire Fallback Chain`。移动端可用 `--split-cjk 4096`。生成器不启动/测试 Unity。
- **Unreal**：在 Editor Python 中运行 `unreal/Scripts/fontcreator233_import.py`；API 不支持时打印 CompositeFont 范围。生成器不启动/测试 UE。
- **Web**：`web/font.css` 全部本地文件；分片字体带 `unicode-range`。不生成 WOFF2。
- **系统**：`system/INSTALL.md` 覆盖 Windows/macOS/Linux。

## 好看、离线的样张

Vercel/Geist 风格：巨大的生成字体 `Aa`、可编辑 type tester（字号/行距/字距）、scale 瀑布、多语言卡片、可搜索/筛选的虚拟化字形网格。

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
fontcreator clean --out dist --dry-run
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
