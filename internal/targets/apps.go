package targets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type appGuide struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Platforms string   `json:"platforms"`
	Method    string   `json:"method"`
	Use       string   `json:"use"`
	Notes     []string `json:"notes,omitempty"`
}

var appGuides = []appGuide{
	{
		ID: "photoshop", Name: "Adobe Photoshop", Platforms: "Windows, macOS", Method: "System font",
		Use:   "Install the TTF, restart Photoshop, then choose the family in the Type tool. For Chinese, Japanese, or Korean, set the paragraph composer to World-Ready Layout and pick the matching language.",
		Notes: []string{"Photoshop reads system-installed fonts; it does not load a TTF from the document folder.", "If the family does not appear, quit every Adobe process and reopen Photoshop."},
	},
	{
		ID: "illustrator", Name: "Adobe Illustrator", Platforms: "Windows, macOS", Method: "System font",
		Use:   "Install the TTF, restart Illustrator, then use the Type tool or Character panel. For CJK text, choose World-Ready Layout in the paragraph options.",
		Notes: []string{"Illustrator can outline text before export when handing files to a machine without the font.", "Use the TTF rather than the web WOFF file."},
	},
	{
		ID: "indesign", Name: "Adobe InDesign", Platforms: "Windows, macOS", Method: "System font",
		Use:   "Install the TTF, restart InDesign, and choose the family in the Character panel. Set the paragraph composer to World-Ready Layout for mixed CJK and Latin copy.",
		Notes: []string{"Package the document with fonts when sending it to another machine.", "The generated font has no advanced OpenType layout features; do not claim optical sizing or stylistic sets."},
	},
	{
		ID: "capcut", Name: "剪映 / CapCut", Platforms: "Windows, macOS, iOS, Android", Method: "System font or import",
		Use:   "Desktop reads system-installed fonts: install the TTF, restart CapCut, add text, then choose the family from the font picker. Mobile versions vary; when the picker exposes Import font, select the TTF. Otherwise render the text in the desktop app or use the caption sample as a style reference.",
		Notes: []string{"手机端常见路径是 文本 → 字体 → 导入字体；不同版本的菜单名称可能不同。", "Some mobile builds only expose built-in fonts; keep a PNG or desktop-rendered title as fallback.", "Import caption.srt for a subtitle track, or use caption.ass in tools that accept ASS styles."},
	},
	{
		ID: "premiere", Name: "Adobe Premiere Pro", Platforms: "Windows, macOS", Method: "System font",
		Use:   "Install the TTF, restart Premiere, then choose the family in Essential Graphics, the Text tool, or the Captions panel. The same family is available for titles and subtitle tracks.",
		Notes: []string{"For a handoff, use File > Project Manager and include the font, or render captions to a flattened video.", "Import caption.srt or caption.ass to audition the subtitle styling."},
	},
	{
		ID: "aftereffects", Name: "Adobe After Effects", Platforms: "Windows, macOS", Method: "System font",
		Use:   "Install the TTF, restart After Effects, and choose the family in the Character panel. Refresh fonts after installing if the app was already open.",
		Notes: []string{"For templates, either bundle the TTF or convert the title layer to shapes before delivery.", "Keep the TTF as the source of truth; avoid substituting a fallback silently."},
	},
	{
		ID: "davinci", Name: "DaVinci Resolve", Platforms: "Windows, macOS, Linux", Method: "System font",
		Use:   "Install the TTF, restart Resolve, then choose the family in Edit titles, the Fusion Text+ node, or subtitle tracks.",
		Notes: []string{"On Linux, refresh fontconfig after installing.", "Import caption.srt for a fast subtitle pass."},
	},
	{
		ID: "finalcut", Name: "Final Cut Pro", Platforms: "macOS", Method: "System font",
		Use:   "Install the TTF with Font Book, restart Final Cut Pro, then choose the family in the Titles browser or the Text inspector.",
		Notes: []string{"If the family is missing, validate the font in Font Book and reopen Final Cut.", "For collaboration, share the font separately; Final Cut libraries do not embed it."},
	},
	{
		ID: "figma", Name: "Figma", Platforms: "Web, Windows, macOS", Method: "Local fonts",
		Use:   "Install the TTF, then refresh local fonts in Figma. In the browser, keep the Figma Font Helper running so the desktop font is visible.",
		Notes: []string{"Figma web does not upload the TTF for you.", "For team files, publish the family as a shared brand asset or install it on every editor's machine."},
	},
	{
		ID: "canva", Name: "Canva", Platforms: "Web, mobile", Method: "Brand font or fallback",
		Use:   "If your plan supports brand fonts, upload the TTF as a brand font. Otherwise use a desktop design tool with the installed font and place the rendered artwork into Canva.",
		Notes: []string{"Canva's font upload availability depends on plan and region.", "Do not rely on a visually similar fallback for a final brand deliverable."},
	},
	{
		ID: "office", Name: "Microsoft Office", Platforms: "Windows, macOS", Method: "System font",
		Use:   "Install the TTF, restart Word, PowerPoint, or Excel, then choose the family in the font menu. Set the East Asian font separately when mixing CJK and Latin text.",
		Notes: []string{"PowerPoint on another machine will substitute the font unless it is installed or embedded.", "For PDF delivery, embed fonts in the export settings."},
	},
	{
		ID: "obs", Name: "OBS Studio", Platforms: "Windows, macOS, Linux", Method: "System font",
		Use:   "Install the TTF, restart OBS, add a Text source, and choose the family in the source properties.",
		Notes: []string{"For a portable scene collection, install the font on the streaming machine.", "Use a solid background or outline for legibility over video."},
	},
	{
		ID: "blender", Name: "Blender", Platforms: "Windows, macOS, Linux", Method: "Load font file",
		Use:   "Create a Text object, open the Object Data properties, and load the TTF with the Load Font button. Blender reads the file directly rather than using system fonts.",
		Notes: []string{"Keep the TTF next to the .blend or pack it into the blend file.", "The font is static; do not expect variable-weight axes."},
	},
	{
		ID: "godot", Name: "Godot", Platforms: "Windows, macOS, Linux", Method: "Import asset",
		Use:   "Drop the TTF into the project, let Godot import it as a FontFile, and assign it to a Label, RichTextLabel, or Theme.",
		Notes: []string{"Use the TTF rather than the WOFF file.", "Set fallback fonts for CJK when shipping a trimmed build."},
	},
	{
		ID: "unity", Name: "Unity + TextMeshPro", Platforms: "Unity 2022.3, Unity 6", Method: "Generated UPM package",
		Use:   "Install the generated unity/ package, then run Tools > Fontcreator233 > Create TMP Font Assets. Use the fallback menu for split CJK builds.",
		Notes: []string{"The package is generated and file-tested; it is not a substitute for testing in your Editor.", "Use dynamic multi-atlas for desktop, static atlases for mobile."},
	},
	{
		ID: "unreal", Name: "Unreal Engine", Platforms: "UE 5.x", Method: "Generated import script",
		Use:   "Run unreal/Scripts/fontcreator233_import.py inside the Editor Python environment, then set the composite font ranges from the manifest when the API does not create them automatically.",
		Notes: []string{"The package is generated and file-tested; review it in your project.", "Cook the font assets with the project."},
	},
}

func buildApps(o Options) ([]string, error) {
	dir := filepath.Join(o.OutDir, "apps")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if len(o.Fonts) == 0 {
		return nil, fmt.Errorf("apps: no fonts")
	}
	fontName := filepath.Base(o.Fonts[0].Path)
	fontRef := "font/" + fontName
	family := o.Family
	style := o.Style
	display := family + " " + style
	fontDir := filepath.Join(dir, "font")
	if err := os.MkdirAll(fontDir, 0o755); err != nil {
		return nil, err
	}
	if err := copyFile(o.Fonts[0].Path, filepath.Join(fontDir, fontName)); err != nil {
		return nil, err
	}

	index := struct {
		Schema   string     `json:"schema"`
		Family   string     `json:"family"`
		Style    string     `json:"style"`
		Font     string     `json:"font"`
		License  string     `json:"license"`
		Readme   string     `json:"readme"`
		Captions []string   `json:"captions"`
		Apps     []appGuide `json:"apps"`
	}{
		Schema:   "fontcreator233/apps/v1",
		Family:   family,
		Style:    style,
		Font:     fontRef,
		License:  o.License,
		Readme:   "README.md",
		Captions: []string{"caption.srt", "caption.ass"},
		Apps:     appGuides,
	}
	indexData, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, "font-usage.json"), append(indexData, '\n'), 0o644); err != nil {
		return nil, err
	}

	var readme strings.Builder
	fmt.Fprintf(&readme, "# %s — app usage\n\n", display)
	fmt.Fprintf(&readme, "The primary font is `%s`. Install it once, then choose it by family name in each app.\n\n", fontRef)
	if o.License == "OFL-1.1" {
		readme.WriteString("License: OFL-1.1. See `../OFL.txt` for the full text.\n\n")
	} else {
		fmt.Fprintf(&readme, "License: %s. Do not redistribute without permission.\n\n", o.License)
	}
	readme.WriteString("## Install once\n\n")
	readme.WriteString("- **Windows:** right-click the TTF, choose Install, then restart the app.\n")
	readme.WriteString("- **macOS:** open the TTF with Font Book, choose Install, then restart the app.\n")
	readme.WriteString("- **Linux:** copy the TTF to `~/.local/share/fonts`, run `fc-cache -f -v`, then restart the app.\n\n")
	readme.WriteString("## App matrix\n\n")
	readme.WriteString("| App | Platforms | Method | Guide |\n|---|---|---|---|\n")
	for _, app := range appGuides {
		fmt.Fprintf(&readme, "| %s | %s | %s | [Open](%s.md) |\n", app.Name, app.Platforms, app.Method, app.ID)
	}
	readme.WriteString("\n## Video captions\n\n")
	readme.WriteString("Use `caption.srt` for a plain subtitle track, or `caption.ass` for tools that accept ASS styling. The sample uses the generated family and a 1920×1080 safe area.\n\n")
	readme.WriteString("## Engine and web packages\n\n")
	readme.WriteString("- Unity: `../unity/` includes the TMP importer, charsets, and fallback chain.\n")
	readme.WriteString("- Unreal: `../unreal/` includes the import script and composite ranges.\n")
	readme.WriteString("- Web: `../web/font.css` includes local TTF/WOFF sources and split `unicode-range` rules.\n")
	readme.WriteString("- System: `../system/INSTALL.md` covers Windows, macOS, and Linux installs.\n\n")
	readme.WriteString("## Honest limits\n\n")
	readme.WriteString("The font is static, with no variable axes, color glyphs, emoji, or complex shaping. Arabic, Hebrew, Indic scripts, Thai, vertical layout, and advanced OpenType layout are not supported. Engine packages are generated and file-tested, not opened in a Unity or Unreal Editor.\n")
	readmePath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readmePath, []byte(readme.String()), 0o644); err != nil {
		return nil, err
	}

	paths := []string{readmePath, filepath.Join(dir, "font-usage.json"), filepath.Join(fontDir, fontName)}
	for _, app := range appGuides {
		doc := appDoc(display, fontRef, app)
		path := filepath.Join(dir, app.ID+".md")
		if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	for name, data := range map[string]string{
		"caption.srt": srtCaption(display),
		"caption.ass": assCaption(family, display),
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func appDoc(display, fontRef string, app appGuide) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s in %s\n\n", display, app.Name)
	fmt.Fprintf(&b, "Font file: `%s`  \nPlatforms: %s  \nMethod: %s\n\n", fontRef, app.Platforms, app.Method)
	b.WriteString("## Use\n\n")
	b.WriteString(app.Use)
	b.WriteString("\n\n## Install\n\n")
	switch app.Method {
	case "System font", "Local fonts":
		b.WriteString("1. Install the TTF with the operating system font installer.\n")
		b.WriteString("2. Fully quit and reopen the app.\n")
		b.WriteString("3. Pick the family by name in the app's font menu.\n")
	case "System font or import":
		b.WriteString("1. **Desktop:** install the TTF with the operating system installer, then restart the app.\n")
		b.WriteString("2. **Mobile:** open a text layer, tap the font picker, and look for `Import font` or a local-font action.\n")
		b.WriteString("3. If the build has no import action, render the title in the desktop app and place the exported PNG or video layer.\n")
	case "Brand font or fallback":
		b.WriteString("1. Open the brand-font upload flow if your plan supports it.\n")
		b.WriteString("2. Upload the TTF and wait for the family to appear.\n")
		b.WriteString("3. If upload is unavailable, use a desktop editor with the TTF and place the rendered artwork.\n")
	case "Load font file", "Import asset":
		fmt.Fprintf(&b, "1. Keep `%s` in the project folder.\n", fontRef)
		b.WriteString("2. Load or import the TTF through the app's font/file picker.\n")
		b.WriteString("3. Assign the font resource to the text object or theme.\n")
	case "Generated UPM package":
		b.WriteString("1. Install `../unity/` as a local or embedded UPM package.\n")
		b.WriteString("2. Run `Tools > Fontcreator233 > Create TMP Font Assets`.\n")
		b.WriteString("3. Use the generated fallback chain for split CJK builds.\n")
	case "Generated import script":
		b.WriteString("1. Open `../unreal/Scripts/fontcreator233_import.py` in the Editor Python environment.\n")
		b.WriteString("2. Run the script and review the imported Font assets.\n")
		b.WriteString("3. Create or verify the composite font ranges from `../unreal/fontcreator233.manifest.json`.\n")
	default:
		b.WriteString("1. Follow the fallback instructions in this guide.\n")
		b.WriteString("2. If the app cannot load a local TTF, render the text in a desktop tool and place the artwork.\n")
	}
	b.WriteString("\n## Notes\n\n")
	for _, note := range app.Notes {
		fmt.Fprintf(&b, "- %s\n", note)
	}
	b.WriteString("\n## Fallback\n\n")
	b.WriteString("If the family does not appear, restart the app and refresh its font cache. If the app cannot load local fonts at all, use a desktop editor with the installed TTF and place the rendered result into the final project.\n")
	return b.String()
}

func srtCaption(display string) string {
	return fmt.Sprintf("1\n00:00:00,000 --> 00:00:03,000\n%s\n\n2\n00:00:03,200 --> 00:00:06,400\nEvery frame, clearly.\n", display)
}

func assCaption(family, display string) string {
	fontName := strings.ReplaceAll(family, ",", " ")
	fontName = strings.TrimSpace(fontName)
	if fontName == "" {
		fontName = "Specimen"
	}
	return fmt.Sprintf(`[Script Info]
Title: %s caption sample
ScriptType: v4.00+
PlayResX: 1920
PlayResY: 1080
WrapStyle: 2
ScaledBorderAndShadow: yes

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,%s,72,&H00FFFFFF,&H000000FF,&H00101010,&H64000000,-1,0,0,0,100,100,0,0,1,3,0,2,80,80,80,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
Dialogue: 0,0:00:00.00,0:00:03.00,Default,,0,0,0,,%s
Dialogue: 0,0:00:03.20,0:00:06.40,Default,,0,0,0,,Every frame, clearly.
Dialogue: 0,0:00:03.20,0:00:06.40,Default,,0,0,0,,每一帧，都清晰有力。
`, display, fontName, display)
}
