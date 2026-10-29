---
name: Mica UI Design System
version: 0.1.0
source: Mica Home
scope: 任意 Web 系统（原生 HTML/CSS，无构建步骤）
themes: light, dark
license: 随源项目
---

# Mica UI 设计规范

> 本文件是**单文件自包含**的设计规范。读到它即可完整复刻界面语言，不需要访问源项目仓库、不需要其他文档。
> 全文的中文说明与英文代码同等重要：代码给实现，规则给判断。

## 给 Agent 的 30 秒摘要

1. 颜色、边框、阴影、圆角、字体**只用 `--mh-*` 令牌**，组件里不写字面色值。
2. 所有组件类名以 `mh-` 开头；令牌名、规则编号（R-xx）不得改名。
3. 视觉骨架：雾面玻璃面板 + 999px 胶囊 + 20/26px 大圆角 + 低饱和柔和阴影 + 青绿强调色。
4. 明暗两套主题由 `html[data-theme="light|dark"]` 驱动，必须两套都验证。
5. 动画只改 `transform` / `opacity`，并且必须在 `prefers-reduced-motion: reduce` 下降级。

## 如何使用本文件

1. **先通读** §1 定位与 §2 规则，再动手。
2. **落盘样式**：把 §4.1 的完整 CSS 保存为项目里的 `mica-ui.css`，在页面 `<head>` 中引入；`<html data-theme="light">` 或 `"dark"` 必须显式设置（可选用 §3.4 的脚本自动跟随系统）。
3. **逐个组件**：按 §4.2 的最小 HTML 结构使用，不要改写类名层级。
4. **交付前自检**：逐条回答 §7 的 8 个问题，任何一条不过都不算完成。

> 禁止只抄某几个组件片段而忽略令牌层和规则层——组件依赖令牌，脱离令牌会立刻退化成普通白卡片。

---

## 1. 设计语言定位

Mica（云母）风格：**冷调雾面玻璃质感**，以低饱和的青绿为唯一强调色，等宽字体承担"工程标注"角色，大圆角胶囊与克制的位移动效共同营造精密、轻盈、可信的界面气质。界面不追求高饱和冲击，而追求层次、通透与秩序。

**像：**

- 精密仪器的操作面板：清晰层级、细线分隔、数值用等宽字对齐。
- 磨砂玻璃叠加在半透明背景上，背景有极淡网格、噪点与缓慢漂移的光斑。
- 标签小而克制（大写、等宽、加字距），内容大而干净。
- 交互反馈是"轻微抬起 + 阴影加深 + 边框转强调色"，不是变色或闪烁。

**不像：**

- 不像霓虹赛博：禁止高饱和撞色、发光描边、强渐变文字。
- 不像 Material 实心色块：禁止大面积纯色卡片、粗重投影、悬浮按钮。
- 不像后台管理模板：禁止密集边框表格风、直角卡片、灰底白块叠罗汉。
- 不像营销落地页：禁止大段彩色插画、跑马灯、夸张缩放动画。

---

## 2. 硬性规则（R-01 ～ R-13）

| 编号 | 级别 | 规则 |
|---|---|---|
| R-01 | MUST | 颜色、边框、阴影、圆角、字体一律引用令牌。组件 CSS 内不得出现主题色彩字面值（`#hex`、`rgb()/rgba()`）。**唯一例外**：令牌定义块本身，以及只表达透明度/蒙版的 `mask-image`、噪点纹理。 |
| R-02 | MUST | 组件类名统一 `mh-` 前缀，采用 `块__元素--修饰符` 结构（如 `.mh-card__head`、`.mh-btn--soft`）。业务类名不得进入组件层。 |
| R-03 | MUST | 强调色底上的文字用 `var(--mh-bg)`，禁止 `#fff`——深色主题下强调色变亮，白字会糊。 |
| R-04 | MUST | 半透明叠层用 `color-mix(in srgb, var(--mh-accent) N%, transparent)`（同理可用于 `--mh-blue`、`--mh-amber`），禁止手写 rgba。 |
| R-05 | NEVER | 不得在组件内写 `@media (prefers-color-scheme)` 第二套配色。主题只由 `html[data-theme]` 切换。 |
| R-06 | NEVER | 不得用 `filter: invert()`、`mix-blend-mode` 取巧反色。 |
| R-07 | MUST | 动画只改 `transform` / `opacity`。持续动画只允许背景光斑（`.mh-blob`）。 |
| R-08 | MUST | 圆角只取以下值：`2px`（装饰短线/小方块）、`5px`（行内码）、`8px`（小图标按钮）、`12px`（提示条）、`13px`（弹层头部 40px 图标）、`14px`（内嵌条目/代码块）、`15px`（图标块）、`20px`（卡片）、`26px`（大面板/弹层）、`999px`（胶囊）。 |
| R-09 | MUST | 页面与区块级间距只取刻度值：`4 / 7 / 10 / 14 / 18 / 22 / 26 / 38 / 44 / 78`（px）；组件内部为对齐图标与文字允许微调，但同一组件内必须一致。 |
| R-10 | MUST | 每个可交互元素必须有可见 `:focus-visible`；纯图标按钮必须有 `aria-label`。 |
| R-11 | MUST | 所有动效必须在 `prefers-reduced-motion: reduce` 下降级（§4.1 末尾已内置）。 |
| R-12 | MUST | 正文与背景对比度 ≥ 4.5:1；`--mh-text-3` 只用于辅助信息，不承载关键内容。 |
| R-13 | MUST | 状态类统一用 `.is-*` 并绑定组件根（如 `.mh-modal.is-open`、`.mh-toast.is-visible`）；禁止使用裸 `.open`、`.active`、`.show` 等全局类名，避免状态类与按钮/其他组件类名冲突。 |

---

## 3. 令牌

### 3.1 命名与层级

- 命名：`--mh-<语义>[-<层级|状态>]`。语义优先——用 `--mh-accent`，不用 `--mh-teal`。
- 层级顺序固定：`bg → surface → text → line → accent`。背景永远最底层，表面浮在其上，文字再浮在表面，线条只做分隔，强调色只做点缀。
- 一个令牌在明暗两套里**语义相同、色值不同**；使用方只记语义，不记颜色。

### 3.2 令牌表

| 令牌 | 浅色 | 深色 | 用途 |
|---|---|---|---|
| `--mh-bg` | `#EDF1F6` | `#0B1117` | 页面基色 |
| `--mh-bg-1` | `#F4F7FA` | `#101821` | 背景渐变起点（顶部） |
| `--mh-bg-2` | `#E9EFF5` | `#0E1620` | 背景渐变中段 |
| `--mh-bg-3` | `#EFF3F7` | `#111A24` | 背景渐变终点（底部） |
| `--mh-text` | `#0A141F` | `#E9EFF6` | 正文、标题 |
| `--mh-text-2` | `#39485A` | `#A9B7C6` | 次级文字、说明 |
| `--mh-text-3` | `#7B8899` | `#6F7E8D` | 弱提示、元信息、图标 |
| `--mh-line` | `rgba(10,20,31,.09)` | `rgba(255,255,255,.10)` | 分隔线、常规边框 |
| `--mh-line-strong` | `rgba(10,20,31,.14)` | `rgba(255,255,255,.18)` | 更重的边框、虚线框 |
| `--mh-surface` | `rgba(255,255,255,.86)` | `rgba(30,40,52,.78)` | 面板/聚焦底 |
| `--mh-glass-a` | `rgba(255,255,255,.78)` | `rgba(28,38,50,.72)` | 玻璃渐变起点 |
| `--mh-glass-b` | `rgba(255,255,255,.58)` | `rgba(20,28,38,.55)` | 玻璃渐变终点 |
| `--mh-glass-solid` | `rgba(255,255,255,.92)` | `rgba(24,33,44,.90)` | 高不透明度玻璃（弹层） |
| `--mh-card-a` | `rgba(255,255,255,.84)` | `rgba(30,40,52,.78)` | 卡片渐变起点 |
| `--mh-card-b` | `rgba(255,255,255,.46)` | `rgba(22,30,40,.55)` | 卡片渐变终点 |
| `--mh-glass-border` | `rgba(255,255,255,.92)` | `rgba(255,255,255,.10)` | 玻璃边框（浅色靠白边提亮） |
| `--mh-inset` | `rgba(255,255,255,.95)` | `rgba(255,255,255,.06)` | 顶部 1px 内高光 |
| `--mh-hover` | `rgba(255,255,255,.62)` | `rgba(255,255,255,.08)` | 悬停底 |
| `--mh-soft` | `rgba(255,255,255,.55)` | `rgba(255,255,255,.05)` | 按钮/输入框静止底 |
| `--mh-solid` | `#FFFFFF` | `#1A2431` | 不透明浮层底 |
| `--mh-accent` | `#00B39F` | `#34D6C2` | 强调色：填充、圆点、徽章 |
| `--mh-accent-deep` | `#00796B` | `#4FE3D0` | 强调色文字（浅色底上可读） |
| `--mh-accent-line` | `rgba(0,179,159,.45)` | `rgba(52,214,194,.50)` | 强调色边框、焦点环 |
| `--mh-accent-soft` | `rgba(0,179,159,.08)` | `rgba(52,214,194,.12)` | 强调色淡底（行内码、状态胶囊） |
| `--mh-blue` | `#2F6BFF` | `#5C8CFF` | 第二强调色（信息/链接类） |
| `--mh-amber` | `#E86A2C` | `#FF8A4C` | 警示色（删除、错误、提醒） |
| `--mh-scrim` | `rgba(10,20,31,.18)` | `rgba(0,0,0,.50)` | 弹层遮罩 |
| `--mh-shadow` | `0 18px 40px -26px rgba(10,20,31,.45), 0 2px 10px rgba(10,20,31,.05)` | `0 18px 40px -26px rgba(0,0,0,.75), 0 2px 10px rgba(0,0,0,.35)` | 常规阴影 |
| `--mh-shadow-hi` | `0 34px 60px -30px rgba(10,20,31,.5), 0 6px 18px rgba(10,20,31,.07)` | `0 34px 60px -30px rgba(0,0,0,.85), 0 6px 18px rgba(0,0,0,.40)` | 悬浮/弹层阴影 |
| `--mh-grid` | `rgba(10,20,31,.055)` | `rgba(255,255,255,.045)` | 背景网格线 |
| `--mh-noise-opacity` | `.045` | `.05` | 噪点透明度 |
| `--mh-noise-blend` | `multiply` | `overlay` | 噪点混合模式 |
| `--mh-blob-1` | `rgba(0,179,159,.26)` | `rgba(52,214,194,.16)` | 光斑一（青绿） |
| `--mh-blob-2` | `rgba(47,107,255,.18)` | `rgba(92,140,255,.14)` | 光斑二（蓝） |
| `--mh-blob-3` | `rgba(232,106,44,.10)` | `rgba(255,138,76,.08)` | 光斑三（橙） |
| `--mh-code-bg` | `rgba(10,20,31,.92)` | `rgba(0,0,0,.55)` | 代码块底 |
| `--mh-code-text` | `#D7E3EE` | `#D9E4EF` | 代码块文字 |
| `--mh-warn-text` | `#8A3C13` | `#F0A878` | 警示文字 |
| `--mh-warn-bg` | `rgba(232,106,44,.08)` | `rgba(232,106,44,.14)` | 警示底 |
| `--mh-warn-line` | `rgba(232,106,44,.22)` | `rgba(232,106,44,.35)` | 警示边框 |
| `--mh-radius` | `20px` | `20px` | 卡片圆角 |
| `--mh-radius-lg` | `26px` | `26px` | 大面板/弹层圆角 |
| `--mh-ease` | `cubic-bezier(.2,.8,.2,1)` | 同左 | 统一缓动 |
| `--mh-font-d` | `"HarmonyOS Sans SC","PingFang SC","Microsoft YaHei",sans-serif` | 同左 | 标题字体 |
| `--mh-font-b` | `"HarmonyOS Sans SC","PingFang SC","Microsoft YaHei",sans-serif` | 同左 | 正文字体 |
| `--mh-font-m` | `"IBM Plex Mono","SFMono-Regular",Consolas,monospace` | 同左 | 等宽字体 |

### 3.3 令牌定义（可直接复制）

已包含在 §4.1 完整 CSS 开头。若你的系统已有自己的令牌体系，只需把表中语义映射到现有变量，**不要改令牌名**。

### 3.4 主题切换契约

- 主题状态写在根元素：`<html data-theme="light">` 或 `<html data-theme="dark">`。
- 未手动选择时跟随系统；用户选择后持久化到 `localStorage['mica.theme']`，取值为 `light` / `dark`。
- 脚本必须在 `<head>` 中**同步**执行（不能 `defer`），否则深色用户刷新会闪白。
- 需要 JS 感知主题时，监听 `window` 上的 `mh:theme` 事件（`event.detail.theme`），或调用 `window.MicaTheme.get()`。

可落盘的等价实现（与源项目 `theme.js` 行为一致）：

```js
(function () {
  var KEY = 'mica.theme';
  var root = document.documentElement;

  function systemTheme() {
    return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }
  function storedTheme() {
    try {
      var v = localStorage.getItem(KEY);
      return v === 'dark' || v === 'light' ? v : null;
    } catch (e) { return null; }
  }
  function apply(theme) {
    root.dataset.theme = theme;
    root.style.colorScheme = theme;
    document.querySelectorAll('[data-theme-toggle]').forEach(function (btn) {
      var label = theme === 'dark' ? '切换到浅色主题' : '切换到深色主题';
      btn.setAttribute('aria-pressed', String(theme === 'dark'));
      btn.setAttribute('aria-label', label);
      btn.setAttribute('title', label);
    });
    window.dispatchEvent(new CustomEvent('mh:theme', { detail: { theme: theme } }));
  }
  function set(theme) {
    try { localStorage.setItem(KEY, theme); } catch (e) {}
    apply(theme);
  }
  window.MicaTheme = {
    get: function () { return root.dataset.theme || 'light'; },
    set: set,
    toggle: function () { set(window.MicaTheme.get() === 'dark' ? 'light' : 'dark'); }
  };
  apply(storedTheme() || systemTheme());
  document.addEventListener('DOMContentLoaded', function () {
    document.querySelectorAll('[data-theme-toggle]').forEach(function (btn) {
      btn.addEventListener('click', function () { window.MicaTheme.toggle(); });
    });
    apply(window.MicaTheme.get());
  });
})();
```

---

## 4. 组件库

### 4.1 完整 CSS（保存为 `mica-ui.css`）

```css
/* ==========================================================================
   Mica UI · 单文件组件库
   版本：0.1.0（与 Mica Home v0.1.0 对齐）
   依赖：无。正文字体 HarmonyOS Sans SC 需随包提供两个 woff2；等宽字体走 Google Fonts（见 §5.1）
   结构：令牌 → 基础层 → 排版 → 布局 → 背景 → 面板 → 顶栏 → 表单
        → 按钮 → 芯片 → 卡片 → 图标 → 徽章 → 空状态 → 弹层
        → Toast → 表格 → 代码 → 警示 → 响应式 → 动效降级
   ========================================================================== */

/* ---------- 字体 ---------- */
/* 把 Regular / Bold 两个 woff2 放在与 CSS 同级的 fonts/ 目录 */
@font-face {
  font-family: "HarmonyOS Sans SC";
  src: local("HarmonyOS Sans SC Regular"), local("HarmonyOS Sans SC"),
       url("fonts/HarmonyOS_Sans_SC_Regular.woff2") format("woff2");
  font-weight: 400;
  font-style: normal;
  font-display: swap;
}
@font-face {
  font-family: "HarmonyOS Sans SC";
  src: local("HarmonyOS Sans SC Bold"),
       url("fonts/HarmonyOS_Sans_SC_Bold.woff2") format("woff2");
  font-weight: 700;
  font-style: normal;
  font-display: swap;
}

/* ---------- 令牌：浅色 ---------- */
:root {
  color-scheme: light;

  --mh-bg: #EDF1F6;
  --mh-bg-1: #F4F7FA;
  --mh-bg-2: #E9EFF5;
  --mh-bg-3: #EFF3F7;

  --mh-text: #0A141F;
  --mh-text-2: #39485A;
  --mh-text-3: #7B8899;

  --mh-line: rgba(10, 20, 31, .09);
  --mh-line-strong: rgba(10, 20, 31, .14);

  --mh-surface: rgba(255, 255, 255, .86);
  --mh-glass-a: rgba(255, 255, 255, .78);
  --mh-glass-b: rgba(255, 255, 255, .58);
  --mh-glass-solid: rgba(255, 255, 255, .92);
  --mh-card-a: rgba(255, 255, 255, .84);
  --mh-card-b: rgba(255, 255, 255, .46);
  --mh-glass-border: rgba(255, 255, 255, .92);
  --mh-inset: rgba(255, 255, 255, .95);
  --mh-hover: rgba(255, 255, 255, .62);
  --mh-soft: rgba(255, 255, 255, .55);
  --mh-solid: #FFFFFF;

  --mh-accent: #00B39F;
  --mh-accent-deep: #00796B;
  --mh-accent-line: rgba(0, 179, 159, .45);
  --mh-accent-soft: rgba(0, 179, 159, .08);
  --mh-blue: #2F6BFF;
  --mh-amber: #E86A2C;

  --mh-scrim: rgba(10, 20, 31, .18);
  --mh-shadow: 0 18px 40px -26px rgba(10, 20, 31, .45), 0 2px 10px rgba(10, 20, 31, .05);
  --mh-shadow-hi: 0 34px 60px -30px rgba(10, 20, 31, .5), 0 6px 18px rgba(10, 20, 31, .07);

  --mh-grid: rgba(10, 20, 31, .055);
  --mh-noise-opacity: .045;
  --mh-noise-blend: multiply;
  --mh-blob-1: rgba(0, 179, 159, .26);
  --mh-blob-2: rgba(47, 107, 255, .18);
  --mh-blob-3: rgba(232, 106, 44, .10);

  --mh-code-bg: rgba(10, 20, 31, .92);
  --mh-code-text: #D7E3EE;
  --mh-warn-text: #8A3C13;
  --mh-warn-bg: rgba(232, 106, 44, .08);
  --mh-warn-line: rgba(232, 106, 44, .22);

  --mh-radius: 20px;
  --mh-radius-lg: 26px;
  --mh-ease: cubic-bezier(.2, .8, .2, 1);

  --mh-font-d: "HarmonyOS Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif;
  --mh-font-b: "HarmonyOS Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif;
  --mh-font-m: "IBM Plex Mono", "SFMono-Regular", Consolas, monospace;
}

/* ---------- 令牌：深色 ---------- */
:root[data-theme="dark"] {
  color-scheme: dark;

  --mh-bg: #0B1117;
  --mh-bg-1: #101821;
  --mh-bg-2: #0E1620;
  --mh-bg-3: #111A24;

  --mh-text: #E9EFF6;
  --mh-text-2: #A9B7C6;
  --mh-text-3: #6F7E8D;

  --mh-line: rgba(255, 255, 255, .10);
  --mh-line-strong: rgba(255, 255, 255, .18);

  --mh-surface: rgba(30, 40, 52, .78);
  --mh-glass-a: rgba(28, 38, 50, .72);
  --mh-glass-b: rgba(20, 28, 38, .55);
  --mh-glass-solid: rgba(24, 33, 44, .90);
  --mh-card-a: rgba(30, 40, 52, .78);
  --mh-card-b: rgba(22, 30, 40, .55);
  --mh-glass-border: rgba(255, 255, 255, .10);
  --mh-inset: rgba(255, 255, 255, .06);
  --mh-hover: rgba(255, 255, 255, .08);
  --mh-soft: rgba(255, 255, 255, .05);
  --mh-solid: #1A2431;

  --mh-accent: #34D6C2;
  --mh-accent-deep: #4FE3D0;
  --mh-accent-line: rgba(52, 214, 194, .50);
  --mh-accent-soft: rgba(52, 214, 194, .12);
  --mh-blue: #5C8CFF;
  --mh-amber: #FF8A4C;

  --mh-scrim: rgba(0, 0, 0, .50);
  --mh-shadow: 0 18px 40px -26px rgba(0, 0, 0, .75), 0 2px 10px rgba(0, 0, 0, .35);
  --mh-shadow-hi: 0 34px 60px -30px rgba(0, 0, 0, .85), 0 6px 18px rgba(0, 0, 0, .40);

  --mh-grid: rgba(255, 255, 255, .045);
  --mh-noise-opacity: .05;
  --mh-noise-blend: overlay;
  --mh-blob-1: rgba(52, 214, 194, .16);
  --mh-blob-2: rgba(92, 140, 255, .14);
  --mh-blob-3: rgba(255, 138, 76, .08);

  --mh-code-bg: rgba(0, 0, 0, .55);
  --mh-code-text: #D9E4EF;
  --mh-warn-text: #F0A878;
  --mh-warn-bg: rgba(232, 106, 44, .14);
  --mh-warn-line: rgba(232, 106, 44, .35);
}

/* ---------- 基础层 ---------- */
*,
*::before,
*::after { box-sizing: border-box; }

html { -webkit-text-size-adjust: 100%; }

body {
  margin: 0;
  background: var(--mh-bg);
  color: var(--mh-text);
  font-family: var(--mh-font-b);
  font-size: 15px;
  line-height: 1.6;
  -webkit-font-smoothing: antialiased;
  overflow-x: hidden;
  transition: background-color .35s var(--mh-ease), color .35s var(--mh-ease);
}

::selection { background: color-mix(in srgb, var(--mh-accent) 22%, transparent); }

h1, h2, h3 { margin: 0; font-family: var(--mh-font-d); font-weight: 700; }
p { margin: 0; }
button, input, select, textarea { font-family: inherit; color: inherit; }

.mh-mono { font-family: var(--mh-font-m); font-variant-numeric: tabular-nums; }

/* ---------- 排版 ---------- */
.mh-display {
  font-family: var(--mh-font-d);
  font-size: clamp(42px, 6.2vw, 78px);
  font-weight: 800;
  letter-spacing: -.042em;
  line-height: 1.01;
  text-wrap: balance;
}
.mh-display em { font-style: normal; color: var(--mh-accent-deep); }

.mh-h2 { font-family: var(--mh-font-d); font-size: 26px; font-weight: 700; letter-spacing: -.025em; line-height: 1.2; }
.mh-h3 { font-family: var(--mh-font-d); font-size: 16px; font-weight: 700; letter-spacing: -.012em; line-height: 1.3; }

.mh-lede { margin-top: 20px; max-width: 46ch; color: var(--mh-text-2); font-size: 15.5px; }

.mh-eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  margin-bottom: 20px;
  font-family: var(--mh-font-m);
  font-size: 10.5px;
  letter-spacing: .2em;
  text-transform: uppercase;
  color: var(--mh-accent-deep);
}
.mh-eyebrow i {
  width: 6px;
  height: 6px;
  border-radius: 2px;
  background: var(--mh-accent);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--mh-accent) 16%, transparent);
  animation: mh-pulse 2.6s ease-in-out infinite;
}

.mh-label {
  font-family: var(--mh-font-m);
  font-size: 12px;
  font-weight: 500;
  letter-spacing: .18em;
  text-transform: uppercase;
  color: var(--mh-text-3);
}

.mh-meta { font-size: 12.5px; color: var(--mh-text-3); }

/* ---------- 布局 ---------- */
.mh-container { max-width: 1240px; width: 100%; margin: 0 auto; padding: 0 22px; }
.mh-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(292px, 1fr)); gap: 18px; }
.mh-stack { display: grid; gap: var(--mh-gap, 16px); }
.mh-spacer { flex: 1; }

/* ---------- 背景场景 ---------- */
.mh-scene {
  position: fixed;
  inset: 0;
  z-index: -1;
  overflow: hidden;
  background: linear-gradient(180deg, var(--mh-bg-1) 0%, var(--mh-bg-2) 58%, var(--mh-bg-3) 100%);
}
.mh-scene::before {
  content: "";
  position: absolute;
  inset: -20%;
  background-image:
    linear-gradient(var(--mh-grid) 1px, transparent 1px),
    linear-gradient(90deg, var(--mh-grid) 1px, transparent 1px);
  background-size: 76px 76px;
  -webkit-mask-image: radial-gradient(ellipse at 50% 18%, #000 0%, rgba(0, 0, 0, .55) 45%, transparent 80%);
  mask-image: radial-gradient(ellipse at 50% 18%, #000 0%, rgba(0, 0, 0, .55) 45%, transparent 80%);
}
.mh-scene::after {
  content: "";
  position: absolute;
  inset: 0;
  opacity: var(--mh-noise-opacity);
  pointer-events: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='160' height='160'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.8' numOctaves='3' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='160' height='160' filter='url(%23n)'/%3E%3C/svg%3E");
  mix-blend-mode: var(--mh-noise-blend);
}
.mh-blob {
  position: absolute;
  border-radius: 50%;
  filter: blur(90px);
  opacity: .75;
  animation: mh-drift 46s var(--mh-ease) infinite alternate;
}
.mh-blob--1 { width: 620px; height: 620px; left: -180px; top: -220px; background: var(--mh-blob-1); }
.mh-blob--2 { width: 540px; height: 540px; right: -140px; top: 40px; background: var(--mh-blob-2); animation-duration: 38s; }
.mh-blob--3 { width: 460px; height: 460px; left: 38%; top: 58%; background: var(--mh-blob-3); animation-duration: 52s; }

/* ---------- 玻璃面板 ---------- */
.mh-glass {
  background: linear-gradient(150deg, var(--mh-glass-a), var(--mh-glass-b));
  -webkit-backdrop-filter: blur(24px) saturate(160%);
  backdrop-filter: blur(24px) saturate(160%);
  border: 1px solid var(--mh-glass-border);
  box-shadow: inset 0 1px 0 var(--mh-inset), var(--mh-shadow);
}
.mh-panel { padding: 20px 22px 18px; border-radius: var(--mh-radius-lg); }

/* ---------- 顶栏 ---------- */
.mh-topbar {
  position: sticky;
  top: 14px;
  z-index: 60;
  max-width: 1240px;
  width: calc(100% - 36px);
  margin: 14px auto 0;
  display: flex;
  align-items: center;
  gap: 26px;
  padding: 9px 10px 9px 18px;
  border-radius: 999px;
  animation: mh-drop .9s var(--mh-ease) both;
}
.mh-brand { display: flex; align-items: center; gap: 10px; flex: 0 0 auto; text-decoration: none; color: inherit; }
.mh-brand svg { display: block; }
.mh-brand__name { display: flex; flex-direction: column; line-height: 1.15; }
.mh-brand__name b { font-family: var(--mh-font-d); font-weight: 800; font-size: 14px; letter-spacing: .02em; }
.mh-brand__name span {
  font-family: var(--mh-font-m);
  font-size: 10px;
  letter-spacing: .16em;
  color: var(--mh-text-3);
  text-transform: uppercase;
}

.mh-nav { display: flex; gap: 4px; margin-left: 8px; }
.mh-nav a {
  position: relative;
  padding: 8px 12px;
  border-radius: 999px;
  font-size: 13.5px;
  color: var(--mh-text-2);
  text-decoration: none;
  transition: color .3s, background .3s;
}
.mh-nav a::after {
  content: "";
  position: absolute;
  left: 50%;
  bottom: 5px;
  width: 0;
  height: 2px;
  border-radius: 2px;
  background: var(--mh-accent);
  transform: translateX(-50%);
  transition: width .35s var(--mh-ease);
}
.mh-nav a:hover { color: var(--mh-text); background: var(--mh-hover); }
.mh-nav a:hover::after { width: 16px; }
.mh-nav a[aria-current="page"] { color: var(--mh-text); font-weight: 600; }
.mh-nav a[aria-current="page"]::after { width: 16px; }

.mh-status {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  flex: 0 0 auto;
  padding: 7px 13px;
  border-radius: 999px;
  font-family: var(--mh-font-m);
  font-size: 10.5px;
  letter-spacing: .12em;
  text-transform: uppercase;
  color: var(--mh-accent-deep);
  border: 1px solid var(--mh-accent-line);
  background: var(--mh-accent-soft);
}
.mh-status i { width: 6px; height: 6px; border-radius: 50%; background: var(--mh-accent); animation: mh-blink 1.8s ease-in-out infinite; }

.mh-theme-toggle {
  width: 36px;
  height: 36px;
  flex: 0 0 auto;
  padding: 0;
  border-radius: 50%;
  display: grid;
  place-items: center;
  cursor: pointer;
  border: 1px solid var(--mh-line);
  background: var(--mh-soft);
  color: var(--mh-text-2);
  transition: background .25s, color .25s, border-color .25s, transform .35s var(--mh-ease);
}
.mh-theme-toggle:hover {
  background: var(--mh-surface);
  color: var(--mh-text);
  border-color: var(--mh-accent-line);
  transform: rotate(-14deg);
}
.mh-theme-toggle svg { display: none; }
.mh-theme-toggle .i-sun { display: block; }
:root[data-theme="dark"] .mh-theme-toggle .i-sun { display: none; }
:root[data-theme="dark"] .mh-theme-toggle .i-moon { display: block; }

/* ---------- 表单 ---------- */
.mh-field {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 210px;
  padding: 8px 14px;
  border-radius: 999px;
  background: var(--mh-soft);
  border: 1px solid var(--mh-line);
  transition: width .45s var(--mh-ease), border-color .3s, background .3s;
}
.mh-field:focus-within { width: 280px; border-color: var(--mh-accent-line); background: var(--mh-surface); }
.mh-field svg { flex: 0 0 auto; color: var(--mh-text-3); }
.mh-input { width: 100%; border: 0; outline: 0; background: transparent; font-size: 13px; color: var(--mh-text); }
.mh-input::placeholder { color: var(--mh-text-3); }

.mh-input--boxed {
  padding: 10px 14px;
  border-radius: var(--mh-radius);
  background: var(--mh-soft);
  border: 1px solid var(--mh-line);
  transition: border-color .3s, background .3s;
}
.mh-input--boxed:focus { border-color: var(--mh-accent-line); background: var(--mh-surface); }

/* ---------- 按钮 ---------- */
.mh-btn {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 0;
  cursor: pointer;
  padding: 9px 18px;
  border-radius: 999px;
  font-family: var(--mh-font-b);
  font-size: 12.5px;
  font-weight: 600;
  line-height: 1;
  color: var(--mh-bg);
  background: var(--mh-text);
  transition: background .35s, transform .35s var(--mh-ease), box-shadow .35s;
}
.mh-btn:hover { transform: translateY(-1px); box-shadow: var(--mh-shadow); }
.mh-btn:active { transform: translateY(0) scale(.98); }

.mh-btn--soft { background: var(--mh-soft); color: var(--mh-text-2); border: 1px solid var(--mh-line); }
.mh-btn--soft:hover { background: var(--mh-surface); color: var(--mh-text); border-color: var(--mh-accent-line); }

.mh-btn--accent { background: var(--mh-accent); color: var(--mh-bg); }
.mh-btn--danger {
  background: color-mix(in srgb, var(--mh-amber) 10%, transparent);
  color: var(--mh-warn-text);
  border: 1px solid var(--mh-warn-line);
}
.mh-btn--danger:hover { background: color-mix(in srgb, var(--mh-amber) 16%, transparent); }

.mh-btn.is-done {
  background: var(--mh-soft);
  color: var(--mh-accent-deep);
  border: 1px solid var(--mh-accent-line);
  box-shadow: none;
}
.mh-btn.is-done:hover { transform: none; background: var(--mh-surface); }

.mh-icon-btn {
  width: 34px;
  height: 34px;
  padding: 0;
  border-radius: 8px;
  display: grid;
  place-items: center;
  border: 0;
  background: transparent;
  color: var(--mh-text-3);
  cursor: pointer;
  transition: color .25s, background .25s;
}
.mh-icon-btn:hover {
  color: var(--mh-amber);
  background: color-mix(in srgb, var(--mh-amber) 10%, transparent);
}

/* ---------- 芯片 ---------- */
.mh-chip {
  border: 1px solid var(--mh-line);
  border-radius: 999px;
  padding: 7px 15px;
  font-size: 12.5px;
  color: var(--mh-text-2);
  cursor: pointer;
  background: var(--mh-soft);
  -webkit-backdrop-filter: blur(12px);
  backdrop-filter: blur(12px);
  transition: all .3s var(--mh-ease);
}
.mh-chip:hover { border-color: var(--mh-accent-line); color: var(--mh-text); transform: translateY(-1px); }
.mh-chip[aria-pressed="true"] {
  background: var(--mh-text);
  color: var(--mh-bg);
  border-color: var(--mh-text);
  box-shadow: var(--mh-shadow);
}
.mh-chip[aria-pressed="true"]:hover { transform: none; }

/* ---------- 卡片 ---------- */
.mh-card {
  position: relative;
  isolation: isolate;
  overflow: hidden;
  padding: 20px 20px 18px;
  border-radius: var(--mh-radius);
  background: linear-gradient(152deg, var(--mh-card-a), var(--mh-card-b));
  -webkit-backdrop-filter: blur(22px) saturate(155%);
  backdrop-filter: blur(22px) saturate(155%);
  border: 1px solid var(--mh-glass-border);
  box-shadow: inset 0 1px 0 var(--mh-inset), var(--mh-shadow);
  transition: transform .5s var(--mh-ease), box-shadow .5s var(--mh-ease), border-color .4s;
  will-change: transform;
}
.mh-card::before {
  content: "";
  position: absolute;
  inset: 0;
  z-index: -1;
  pointer-events: none;
  background: radial-gradient(
    260px circle at var(--mx, 50%) var(--my, 0%),
    color-mix(in srgb, var(--mh-accent) 13%, transparent),
    transparent 62%
  );
  opacity: 0;
  transition: opacity .45s;
}
.mh-card:hover { transform: translateY(-9px); box-shadow: inset 0 1px 0 var(--mh-inset), var(--mh-shadow-hi); border-color: var(--mh-accent-line); }
.mh-card:hover::before { opacity: 1; }
.mh-card:focus-within { border-color: var(--mh-accent-line); }

.mh-card__head { display: flex; align-items: center; gap: 13px; }
.mh-card__desc { margin-top: 14px; font-size: 13px; line-height: 1.62; color: var(--mh-text-2); }
.mh-card__foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 16px;
  padding-top: 14px;
  border-top: 1px solid var(--mh-line);
}

/* ---------- 图标块与色调 ---------- */
.mh-icon {
  width: 48px;
  height: 48px;
  flex: 0 0 auto;
  border-radius: 15px;
  display: grid;
  place-items: center;
  background: linear-gradient(150deg, var(--ic1), var(--ic2));
  border: 1px solid var(--mh-glass-border);
  color: var(--ic-ink);
  box-shadow: inset 0 1px 0 var(--mh-inset), var(--mh-shadow);
  transition: transform .5s var(--mh-ease);
}
.mh-card:hover .mh-icon { transform: rotate(-4deg) scale(1.06); }

.mh-tone-a { --ic1: color-mix(in srgb, var(--mh-accent) 20%, transparent); --ic2: color-mix(in srgb, var(--mh-accent) 6%, transparent); --ic-ink: var(--mh-accent-deep); }
.mh-tone-b { --ic1: color-mix(in srgb, var(--mh-blue) 17%, transparent); --ic2: color-mix(in srgb, var(--mh-blue) 5%, transparent); --ic-ink: var(--mh-blue); }
.mh-tone-c { --ic1: color-mix(in srgb, var(--mh-amber) 16%, transparent); --ic2: color-mix(in srgb, var(--mh-amber) 5%, transparent); --ic-ink: var(--mh-amber); }
.mh-tone-d { --ic1: color-mix(in srgb, var(--mh-text-2) 14%, transparent); --ic2: color-mix(in srgb, var(--mh-text-2) 4%, transparent); --ic-ink: var(--mh-text-2); }

/* ---------- 徽章 ---------- */
.mh-badge {
  position: absolute;
  top: 14px;
  right: 14px;
  padding: 4px 9px;
  border-radius: 999px;
  font-family: var(--mh-font-m);
  font-size: 9.5px;
  letter-spacing: .1em;
  text-transform: uppercase;
  color: var(--mh-bg);
  background: var(--mh-accent);
  box-shadow: 0 6px 14px -8px color-mix(in srgb, var(--mh-accent) 90%, transparent);
}

/* ---------- 空状态 ---------- */
.mh-empty {
  grid-column: 1 / -1;
  padding: 56px 24px;
  text-align: center;
  border-radius: var(--mh-radius);
  border: 1px dashed var(--mh-line-strong);
  color: var(--mh-text-3);
  font-size: 13.5px;
  background: var(--mh-soft);
}
.mh-empty b { display: block; margin-bottom: 6px; font-family: var(--mh-font-d); font-size: 17px; font-weight: 700; color: var(--mh-text); }

/* ---------- 弹层 ---------- */
.mh-modal {
  position: fixed;
  inset: 0;
  z-index: 70;
  display: grid;
  place-items: end center;
  pointer-events: none;
  visibility: hidden;
  transition: visibility 0s linear .55s;
}
.mh-modal.is-open { pointer-events: auto; visibility: visible; transition: visibility 0s; }
.mh-modal__scrim {
  position: absolute;
  inset: 0;
  opacity: 0;
  background: var(--mh-scrim);
  -webkit-backdrop-filter: blur(10px);
  backdrop-filter: blur(10px);
  transition: opacity .55s var(--mh-ease);
}
.mh-modal.is-open .mh-modal__scrim { opacity: 1; }

.mh-modal__panel {
  position: relative;
  width: min(1120px, calc(100% - 28px));
  height: min(86vh, 860px);
  margin-bottom: 14px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: var(--mh-radius-lg);
  background: linear-gradient(160deg, var(--mh-glass-solid), var(--mh-surface));
  -webkit-backdrop-filter: blur(30px) saturate(170%);
  backdrop-filter: blur(30px) saturate(170%);
  border: 1px solid var(--mh-glass-border);
  box-shadow: var(--mh-shadow-hi);
  opacity: 0;
  transform: translateY(28px) scale(.985);
  transition: transform .55s var(--mh-ease), opacity .4s var(--mh-ease);
}
.mh-modal.is-open .mh-modal__panel { opacity: 1; transform: none; }

.mh-modal__head { display: flex; align-items: center; gap: 14px; padding: 16px 20px; border-bottom: 1px solid var(--mh-line); }
.mh-modal__head .mh-icon { width: 40px; height: 40px; border-radius: 13px; }
.mh-modal__title { display: flex; flex-direction: column; min-width: 0; line-height: 1.25; }
.mh-modal__title b { font-family: var(--mh-font-d); font-size: 16px; letter-spacing: -.01em; }
.mh-modal__title span {
  font-family: var(--mh-font-m);
  font-size: 10.5px;
  letter-spacing: .1em;
  color: var(--mh-text-3);
  text-transform: uppercase;
}
.mh-modal__close {
  margin-left: auto;
  width: 34px;
  height: 34px;
  padding: 0;
  border-radius: 50%;
  display: grid;
  place-items: center;
  cursor: pointer;
  border: 1px solid var(--mh-line);
  background: var(--mh-soft);
  color: var(--mh-text-2);
  font-size: 16px;
  line-height: 1;
  transition: background .25s, color .25s, transform .35s var(--mh-ease);
}
.mh-modal__close:hover { background: var(--mh-surface); color: var(--mh-text); transform: rotate(90deg); }
.mh-modal__body { flex: 1; padding: 22px; overflow: auto; }

/* ---------- Toast ---------- */
.mh-toast {
  position: fixed;
  left: 50%;
  bottom: 26px;
  z-index: 80;
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 12px 18px 12px 14px;
  border-radius: 999px;
  font-size: 13px;
  color: var(--mh-text);
  border: 1px solid var(--mh-glass-border);
  background: linear-gradient(150deg, var(--mh-glass-solid), var(--mh-glass-a));
  -webkit-backdrop-filter: blur(26px) saturate(170%);
  backdrop-filter: blur(26px) saturate(170%);
  box-shadow: var(--mh-shadow-hi);
  opacity: 0;
  pointer-events: none;
  transform: translate(-50%, 80px);
  transition: transform .6s var(--mh-ease), opacity .4s;
}
.mh-toast.is-visible { opacity: 1; transform: translate(-50%, 0); }
.mh-toast__icon {
  width: 22px;
  height: 22px;
  flex: 0 0 auto;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--mh-accent);
  color: var(--mh-bg);
}

/* ---------- 表格 ---------- */
.mh-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.mh-table th {
  padding: 10px 12px;
  text-align: left;
  font-family: var(--mh-font-m);
  font-size: 10.5px;
  font-weight: 500;
  letter-spacing: .1em;
  text-transform: uppercase;
  color: var(--mh-text-3);
  border-bottom: 1px solid var(--mh-line-strong);
}
.mh-table td { padding: 12px; vertical-align: top; color: var(--mh-text-2); border-bottom: 1px solid var(--mh-line); }
.mh-table tr:last-child td { border-bottom: 0; }

/* ---------- 代码 ---------- */
.mh-code {
  padding: 1px 6px;
  border-radius: 5px;
  font-family: var(--mh-font-m);
  font-size: .88em;
  background: var(--mh-accent-soft);
  color: var(--mh-accent-deep);
}
pre.mh-code {
  display: block;
  margin: 16px 0;
  padding: 16px 18px;
  border-radius: 14px;
  overflow: auto;
  font-size: 12.5px;
  line-height: 1.8;
  background: var(--mh-code-bg);
  color: var(--mh-code-text);
  border: 1px solid var(--mh-line);
}

/* ---------- 警示提示 ---------- */
.mh-note {
  margin-top: 14px;
  padding: 12px 14px;
  border-radius: 12px;
  font-size: 13px;
  color: var(--mh-warn-text);
  background: var(--mh-warn-bg);
  border: 1px solid var(--mh-warn-line);
}
.mh-note b { color: var(--mh-warn-text); }

/* ---------- 焦点可见 ---------- */
a:focus-visible,
button:focus-visible,
input:focus-visible,
select:focus-visible,
textarea:focus-visible,
[tabindex]:focus-visible {
  outline: 2px solid var(--mh-accent-line);
  outline-offset: 2px;
}

/* ---------- 响应式 ---------- */
@media (max-width: 1000px) {
  .mh-nav { display: none; }
}

@media (max-width: 640px) {
  .mh-topbar { gap: 12px; padding: 8px 8px 8px 14px; }
  .mh-field { width: auto; flex: 1; }
  .mh-field:focus-within { width: auto; }
  .mh-status { display: none; }
  .mh-container { padding: 0 18px; }
  .mh-grid { grid-template-columns: 1fr; }
  .mh-card__desc { margin-top: 12px; }
  .mh-modal__panel { width: calc(100% - 16px); height: 92vh; }
  .mh-modal__body { padding: 16px; }
}

/* ---------- 动效关键帧 ---------- */
@keyframes mh-drop {
  from { opacity: 0; transform: translateY(-16px); }
  to { opacity: 1; transform: none; }
}
@keyframes mh-rise {
  from { opacity: 0; transform: translateY(22px); }
  to { opacity: 1; transform: none; }
}
@keyframes mh-blink {
  0%, 100% { opacity: 1; }
  50% { opacity: .25; }
}
@keyframes mh-pulse {
  0%, 100% { box-shadow: 0 0 0 3px color-mix(in srgb, var(--mh-accent) 14%, transparent); }
  50% { box-shadow: 0 0 0 7px color-mix(in srgb, var(--mh-accent) 5%, transparent); }
}
@keyframes mh-drift {
  0% { transform: translate3d(0, 0, 0) scale(1); }
  50% { transform: translate3d(60px, -40px, 0) scale(1.08); }
  100% { transform: translate3d(-40px, 50px, 0) scale(.96); }
}

/* ---------- 动效降级 ---------- */
@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    animation-duration: .001ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: .06s !important;
  }
  body { transition: none; }
}
```

### 4.2 组件用法

每个组件给出用途、最小 HTML 与关键状态。类名不可拆分或改名；需要扩展时在其后追加自己的类，不要改组件类。

#### 4.2.1 背景场景 `.mh-scene`

放在 `<body>` 第一个子元素，固定在最底层。页面内容无需额外背景色。

```html
<div class="mh-scene" aria-hidden="true">
  <div class="mh-blob mh-blob--1"></div>
  <div class="mh-blob mh-blob--2"></div>
  <div class="mh-blob mh-blob--3"></div>
</div>
```

- Do：整站只放一个；光斑数量保持 3 个。
- Don't：在 `.mh-scene` 上叠加业务内容；不要提升 `z-index`。

#### 4.2.2 玻璃面板 `.mh-glass` / `.mh-panel`

用于承载一组信息的浮层容器（状态面板、侧栏卡片、统计块）。

```html
<aside class="mh-glass mh-panel">
  <div class="mh-label">System</div>
  <p class="mh-meta">宿主版本 · v0.1.0</p>
</aside>
```

- Do：`mh-glass` 负责质感，`mh-panel` 负责内边距与大圆角，可单独使用前者。
- Don't：在 `.mh-glass` 上再写 `background` 覆盖，会破坏明暗两套的半透明度。

#### 4.2.3 顶栏 `.mh-topbar`

```html
<header class="mh-topbar mh-glass">
  <a class="mh-brand" href="/">
    <svg width="30" height="30" viewBox="0 0 30 30" fill="none" aria-hidden="true">…</svg>
    <span class="mh-brand__name"><b>Product</b><span>Modular Shell</span></span>
  </a>
  <nav class="mh-nav" aria-label="主导航">
    <a href="/" aria-current="page">仓库</a>
    <a href="/docs">文档</a>
  </nav>
  <div class="mh-spacer"></div>
  <label class="mh-field">
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><circle cx="11" cy="11" r="7"/><path d="M20 20l-3.5-3.5"/></svg>
    <input class="mh-input" type="search" placeholder="搜索…" aria-label="搜索">
  </label>
  <span class="mh-status"><i></i><span>ONLINE</span></span>
  <button class="mh-theme-toggle" data-theme-toggle type="button" aria-pressed="false" aria-label="切换主题">
    <svg class="i-sun" …></svg>
    <svg class="i-moon" …></svg>
  </button>
</header>
```

- 当前页导航项必须带 `aria-current="page"`。
- 主题按钮的 `.i-sun` / `.i-moon` 必须同时存在，显隐由 CSS 控制。
- 窄屏（≤1000px）自动隐藏 `.mh-nav`；≤640px 隐藏 `.mh-status`。

#### 4.2.4 表单 `.mh-field` / `.mh-input`

胶囊搜索框（聚焦展宽 210→280），或独立输入框 `.mh-input--boxed`。

```html
<label class="mh-field">
  <svg …></svg>
  <input class="mh-input" type="search" placeholder="搜索模块、作者或能力…" aria-label="搜索模块">
</label>

<input class="mh-input mh-input--boxed" type="text" placeholder="写点什么…" aria-label="便签内容">
```

- Do：每个输入都有关联 `aria-label` 或真实 `<label>`。
- Don't：把 placeholder 当唯一标签；不要给 `.mh-input` 加边框（边框属于容器）。

#### 4.2.5 按钮 `.mh-btn`

四种语义：默认（反色主按钮）、`--soft`（次要）、`--accent`（强调）、`--danger`（删除类）。另有 `.is-done` 表示"已完成/已启用"的切换态。

```html
<button class="mh-btn" type="button">安装</button>
<button class="mh-btn mh-btn--soft" type="button">取消</button>
<button class="mh-btn mh-btn--accent" type="button">确认</button>
<button class="mh-btn mh-btn--danger" type="button">删除</button>
<button class="mh-btn is-done" type="button">已安装</button>
```

- 默认按钮的配色是"文字色底 + 背景色字"，深色下自动反转，这是本设计语言的标志，不要换成强调色底。
- 纯图标按钮用 `.mh-icon-btn`，必须带 `aria-label`。

#### 4.2.6 芯片 `.mh-chip`

用于筛选、分类切换，状态由 `aria-pressed` 表达，不另加 `active` 类。

```html
<div class="mh-stack" style="--mh-gap:7px;grid-auto-flow:column">
  <button class="mh-chip" type="button" aria-pressed="true">全部</button>
  <button class="mh-chip" type="button" aria-pressed="false">已安装</button>
</div>
```

- Do：单行横向排布、文字短（≤6 字）。
- Don't：用芯片承载主操作；选中态不要自己写颜色。

#### 4.2.7 卡片 `.mh-card`

```html
<article class="mh-card" tabindex="0" style="--d:60ms">
  <span class="mh-badge">RUNNING</span>
  <div class="mh-card__head">
    <div class="mh-icon mh-tone-a"><svg …></svg></div>
    <div>
      <h3 class="mh-h3">模块名称</h3>
      <div class="mh-meta"><span class="mh-mono">v1.0.0</span> · 分类</div>
    </div>
  </div>
  <p class="mh-card__desc">一句话说明这个模块做什么。</p>
  <div class="mh-card__foot">
    <span class="mh-meta mh-mono">权限 2 项</span>
    <button class="mh-btn" type="button">安装</button>
  </div>
</article>
```

- 可点击卡片用 `tabindex="0"`，并在 JS 中响应 `Enter` / `Space`。
- 入场所需的 stagger：给卡片写内联 `--d`（如 60ms、115ms…，步长 55ms），并加 `animation: mh-rise .8s var(--mh-ease) var(--d, 0ms) both`。
- Do：卡片内最多一个主按钮。
- Don't：卡片背景直接写白色；不要用 `transform` 之外的方式做 hover 位移。

#### 4.2.8 图标 `.mh-icon` + 色调

四档色调代表不同能力域：`mh-tone-a`（青绿，核心）、`mh-tone-b`（蓝，信息）、`mh-tone-c`（橙，警示/实验）、`mh-tone-d`（中性，工具）。

```html
<div class="mh-icon mh-tone-a">
  <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor"
       stroke-width="1.55" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <circle cx="12" cy="12" r="8.5"/>
  </svg>
</div>
```

图标规范：24×24 viewBox、`stroke-width:1.55`、`stroke-linecap/join:round`、`fill:none`、颜色继承 `currentColor`。

#### 4.2.9 徽章 `.mh-badge`

只用于卡片右上角的短状态（≤10 个字符，大写）。`position:absolute` 依赖父级为定位容器（`.mh-card` 已满足）。

```html
<span class="mh-badge">RUNNING</span>
```

#### 4.2.10 空状态 `.mh-empty`

```html
<div class="mh-empty">
  <b>这里还没有内容</b>
  添加第一项后，它会出现在这里。
</div>
```

放在网格容器内时自动横跨整行（`grid-column:1/-1`）。

#### 4.2.11 弹层 `.mh-modal`

```html
<div class="mh-modal" id="modal" aria-hidden="true">
  <div class="mh-modal__scrim"></div>
  <section class="mh-modal__panel" role="dialog" aria-modal="true" aria-labelledby="modal-title">
    <header class="mh-modal__head">
      <div class="mh-icon mh-tone-a"><svg …></svg></div>
      <div class="mh-modal__title">
        <b id="modal-title">面板标题</b>
        <span>v1.0.0 · 作者</span>
      </div>
      <button class="mh-modal__close" type="button" aria-label="关闭">✕</button>
    </header>
    <div class="mh-modal__body">…</div>
  </section>
</div>
```

打开/关闭只切 `.is-open` 与 `aria-hidden`；`Esc` 关闭，点遮罩关闭。必须实现焦点管理：打开时把焦点移入面板，关闭时归还触发元素。

#### 4.2.12 Toast `.mh-toast`

```html
<div class="mh-toast" role="status" aria-live="polite">
  <span class="mh-toast__icon"><svg …></svg></span>
  <span>已保存</span>
</div>
```

显示时加 `.is-visible`，2.6s 后移除；同一时刻只保留一个（复用同一个节点）。

#### 4.2.13 表格 `.mh-table`

```html
<table class="mh-table">
  <thead><tr><th>字段</th><th>说明</th></tr></thead>
  <tbody>
    <tr><td><code class="mh-code">id</code></td><td>唯一标识</td></tr>
    <tr><td><code class="mh-code">version</code></td><td>语义化版本</td></tr>
  </tbody>
</table>
```

- Do：表头文字自动大写等宽；数值列加 `.mh-mono`。
- Don't：给表格加外边框或斑马纹。

#### 4.2.14 代码 `.mh-code`

行内用 `<code class="mh-code">`，块级用 `<pre class="mh-code"><code>…</code></pre>`。块级自带深色底，明暗两套通用。

#### 4.2.15 警示 `.mh-note`

```html
<div class="mh-note"><b>注意</b>：这一步会覆盖原数据。</div>
```

只用于提醒与错误说明，不用于普通信息（普通信息用 `.mh-meta`）。

### 4.3 卡片指针光晕（可选增强）

`.mh-card::before` 读取 `--mx` / `--my`。无 JS 时默认 `50% 0`，光晕仍成立。需要跟随指针时加这 6 行：

```js
document.addEventListener('pointermove', function (e) {
  var card = e.target.closest('.mh-card');
  if (!card) return;
  var r = card.getBoundingClientRect();
  card.style.setProperty('--mx', ((e.clientX - r.left) / r.width * 100) + '%');
  card.style.setProperty('--my', ((e.clientY - r.top) / r.height * 100) + '%');
});
```

---

## 5. 排版与间距

### 5.1 字体

两族分工：正文与标题都用 **HarmonyOS Sans SC**（Regular 400 / Bold 700，两个 woff2 自托管），标签/数字/代码用 **IBM Plex Mono**（Google Fonts）。中文回退到 `PingFang SC` / `Microsoft YaHei`。

把两个 woff2 放在 CSS 同级的 `fonts/` 目录，`@font-face` 已包含在 §4.1 开头；页面只需为等宽字体加一行：

```html
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;500&display=swap" rel="stylesheet">
```

HarmonyOS Sans SC 可免费商用；若项目已有该字体文件，直接替换 `fonts/` 下同名文件即可。本机已安装该字体时 `local()` 优先，不会重复下载。

### 5.2 字号层级

| 层级 | 字号/行高 | 字重 | 字距 | 用途 |
|---|---|---|---|---|
| Display | `clamp(42px, 6.2vw, 78px)` / 1.01 | 800 | -0.042em | 页面主标题，`text-wrap:balance` |
| H2 | `26px` / 1.2 | 700 | -0.025em | 区块标题 |
| H3 | `16px` / 1.3 | 700 | -0.012em | 卡片标题、条目标题 |
| 正文 | `15px` / 1.6 | 400 | 0 | 默认正文 |
| 紧凑正文 | `13.5px` / 1.6 | 400 | 0 | 卡片描述、列表内容 |
| 说明 | `13px` / 1.62 | 400 | 0 | 注释、辅助说明 |
| 元信息 | `12.5px` | 400 | 0 | 版本号、作者、次级信息 |
| 弱提示 | `11px` | 400 | .02em | 尺寸、权限数等极弱信息 |
| 等宽标签 | `10.5px` | 500 | .14–.2em，大写 | eyebrow、状态胶囊、导航副标题 |
| 徽章 | `9.5px` | 500 | .1em，大写 | 卡片角标 |
| 按钮 | `12.5px` / 1 | 600 | 0 | 所有按钮 |

### 5.3 排版规则

- 标题一律负字距（越大越负），中文标题也用标题族，不要用系统默认粗体。
- 英文小标签一律等宽 + 大写 + 正字距，这是"工程标注"气质的来源。
- 数字（版本、计数、时间、权限项）用 `font-variant-numeric: tabular-nums`（`.mh-mono` 已包含）。
- 首屏说明文字最大 `46ch`；主标题用 `text-wrap:balance`，避免两行长短悬殊。
- 一个视口内字号层级不超过 4 档。

### 5.4 间距与容器

- 区块级刻度：`4 / 7 / 10 / 14 / 18 / 22 / 26 / 38 / 44 / 78`。组件内部为对齐图标/文字允许小步微调（如 `8 / 9 / 11 / 13`），但不要引入新的层级节奏。
- 页面容器：最大宽 `1240px`，桌面左右 `22px`，≤640px 为 `18px`。
- 顶栏：`position:sticky; top:14px`，宽 `calc(100% - 36px)`，内边距 `9px 10px 9px 18px`。
- 区块纵向节奏：Hero 顶部 `78px`、区块间距 `44px`、卡片内边距 `20px`、卡片底栏上边距 `16px`。

### 5.5 栅格与断点

| 场景 | 规则 |
|---|---|
| 卡片列表 | `repeat(auto-fill, minmax(292px, 1fr))`，gap `18px` |
| Hero 双栏 | `1.4fr .9fr`，gap `44px`，`align-items:end` |
| ≤1000px | Hero 转单列；隐藏顶部导航 |
| ≤640px | 卡片单列；隐藏状态胶囊；容器 18px；弹层 `92vh` |

### 5.6 层级（z-index）

| 层 | 值 |
|---|---|
| 背景场景 | `-1` |
| 顶栏 | `60` |
| 弹层 | `70` |
| Toast | `80` |

---

## 6. 动效

### 6.1 时间与缓动

统一缓动：`--mh-ease: cubic-bezier(.2, .8, .2, 1)`。

| 场景 | 时长 |
|---|---|
| 悬停底色/文字变化 | 250–350ms |
| 卡片抬升、阴影、图标旋转 | 500ms |
| 弹层进出（transform） | 550ms |
| 弹层遮罩透明度 | 550ms |
| 入场动画 | 800–950ms，卡片 stagger 55ms |
| 背景光斑漂移 | 38–52s，`infinite alternate` |
| Toast 进出 | 600ms（位移）/ 400ms（透明度） |

### 6.2 交互位移约定

| 元素 | 反馈 |
|---|---|
| 按钮 hover | `translateY(-1px)` + 阴影 |
| 按钮 active | `scale(.98)` |
| 卡片 hover | `translateY(-9px)` + `--mh-shadow-hi` + 边框转强调色 |
| 卡片图标 hover | `rotate(-4deg) scale(1.06)` |
| 主题切换按钮 hover | `rotate(-14deg)` |
| 弹层关闭按钮 hover | `rotate(90deg)` |
| 芯片 hover | `translateY(-1px)` |
| 导航下划线 | 宽度 0 → 16px |

### 6.3 性能与降级

- 只动 `transform` / `opacity`；不要动画 `top/left/width/height/box-shadow`（阴影变化用两档预设值切换）。
- 持续动画只允许背景光斑；列表入场动画只播一次。
- `prefers-reduced-motion: reduce` 下所有动画与过渡近似关闭（§4.1 已内置）。
- 不在滚动容器内使用 `will-change`（仅卡片这种 hover 元素使用，且已在 CSS 中限定）。

---

## 7. 自检清单（交付前逐条回答）

1. **硬编码检查**：令牌块之外是否还有 `#hex` / `rgb()` / `rgba()` 的主题色值？（命令：`grep -nE '#[0-9a-fA-F]{3,6}|rgba?\(' mica-ui.css`，命中只应在令牌块与蒙版）
2. **深色检查**：切到 `data-theme="dark"` 后是否出现白色块、白边、白字不可读？边框是否仍可见？
3. **对比度**：正文与背景 ≥ 4.5:1；`--mh-text-3` 是否只用于辅助信息？
4. **焦点**：每个交互元素 Tab 到时是否都有可见焦点环？焦点环是否 `2px` + `offset 2px`？
5. **键盘**：Tab 顺序是否合理？卡片能否 Enter/Space 触发？弹层能否 Esc 关闭？
6. **动效降级**：系统开启"减少动态效果"后是否无持续动画与长过渡？
7. **阶梯值**：圆角是否只用了 2/5/8/12/13/14/15/20/26/999（px）？页面与区块间距是否只用了 4/7/10/14/18/22/26/38/44/78？
8. **字体**：HarmonyOS Sans SC 的 Regular/Bold 是否加载成功？中文回退是否正常（断网时仍可读）？等宽字体是否只用于标签/数字/代码？
9. **状态类冲突**：是否所有状态都是 `.is-*` 且挂在组件根上？全局搜索 `.open`、`.active`、`.show`，确认没有跨组件复用；打开弹层时容器不应意外继承按钮的深色背景或胶囊圆角。

---

## 8. 溯源与同步

- 本文件是 **Mica Home v0.1.0** 设计语言的快照，令牌与组件提炼自 `web/theme.css`、`web/index.html`、`web/docs-modules.html`、`web/docs-theme.html`、`modules/notes/frontend/index.js`。
- 源项目宿主页面使用裸类名（`.topbar`、`.mod` 等），本规范统一改为 `mh-` 命名空间；两者视觉等价，但**不要**把宿主裸类名当作公共 API。
- 源项目令牌若变更，需要重新生成本文件的 §3.2 表与 §4.1 令牌块；两处必须逐条一致。
- 允许的本地化调整：品牌名、Logo、导航项、业务文案、业务图标。除此之外的令牌、圆角、间距、动效曲线、组件结构不应改动——一旦改动就不再是同一套设计语言。
