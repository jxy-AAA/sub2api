import tu9 from '../assets/apihub-guide/tu9.png'
import tu10 from '../assets/apihub-guide/tu10.png'
import tu11 from '../assets/apihub-guide/tu11.png'
import { SANDOG_API_BASE_URL, SANDOG_DEFAULT_MODEL } from '@/constants/sandog'

export interface ApihubGuideImageMap { tu9: string; tu10: string; tu11: string }
export interface ApihubGuideSidebarItem { text: string; href: string }
export interface ApihubGuideSidebarGroup { title: string; items: ApihubGuideSidebarItem[] }
export interface ApihubGuideHeading { level: number; id: string; text: string }
export interface ApihubGuideContentBlock { id: string; title: string; level: number; html: string }
export interface ApihubGuideArticleBlock { type: 'html'; html: string }
export interface ApihubGuideArticleSection { id: string; title: string; level: 2 | 3 | 4; blocks: ApihubGuideArticleBlock[] }
export interface ApihubGuideArticleMeta { providerName: string; sourcePath: string }
export interface ApihubGuideArticle {
  title: string
  kicker?: string
  summary?: string
  meta?: ApihubGuideArticleMeta
  sections: ApihubGuideArticleSection[]
}
export interface ApihubGuideDocument {
  title: string
  description: string
  providerName: string
  article: ApihubGuideArticle
  articleHtml: string
  images: ApihubGuideImageMap
  sidebarGroups: ApihubGuideSidebarGroup[]
  headings: ApihubGuideHeading[]
  toc: ApihubGuideHeading[]
  contentBlocks: ApihubGuideContentBlock[]
}

const apiBaseURL = SANDOG_API_BASE_URL
const providerName = 'APIHub'

export const apihubGuideImages: ApihubGuideImageMap = { tu9, tu10, tu11 }

export const apihubGuideSidebarGroups: ApihubGuideSidebarGroup[] = [
  {
    title: '快速开始',
    items: [
      { text: '欢迎使用', href: '#welcome' },
      { text: '中转站是什么', href: '#what-is-relay' },
      { text: '价格说明', href: '#pricing' },
    ],
  },
  {
    title: '使用指南',
    items: [
      { text: '注册账号', href: '#register-account' },
      { text: '创建专属 Key', href: '#create-key' },
      { text: '修改令牌设置', href: '#token-settings' },
      { text: '模型选择', href: '#model-selection' },
      { text: '充值', href: '#recharge' },
    ],
  },
  {
    title: 'Node.js 环境安装',
    items: [
      { text: 'Windows 平台', href: '#node-windows' },
      { text: 'macOS 平台', href: '#node-macos' },
      { text: 'Linux 平台', href: '#node-linux' },
    ],
  },
  {
    title: '快速配置工具',
    items: [
      { text: 'CC-Switch 配置工具', href: '#cc-switch-config' },
      { text: 'Claude Code Hub', href: '#claude-code-hub' },
    ],
  },
  {
    title: '第三方应用',
    items: [
      { text: 'Hapi 远程控制', href: '#hapi-remote-control' },
      { text: 'Hapi 进阶：优选 IP 配置', href: '#hapi-advanced-ip' },
      { text: 'Alma 客户端', href: '#alma-client' },
      { text: 'CherryStudio 客户端', href: '#cherrystudio-client' },
      { text: 'OpenCode', href: '#opencode' },
    ],
  },
  {
    title: '其他',
    items: [
      { text: '疑难杂症', href: '#troubleshooting' },
      { text: '常见问题', href: '#faq' },
      { text: '友情链接', href: '#links' },
    ],
  },
]

export const apihubGuideSidebarSections = apihubGuideSidebarGroups

export const apihubGuideHeadings: ApihubGuideHeading[] = [
  { level: 1, id: 'codex-guide', text: 'CodeX 部署指南' },
  { level: 2, id: 'quick-navigation', text: '快速导航' },
  { level: 2, id: 'cc-switch-config', text: '使用 CC-Switch 快速配置（推荐）' },
  { level: 3, id: 'config-steps', text: '配置步骤' },
  { level: 2, id: 'manual-config', text: '手动命令行配置' },
  { level: 3, id: 'node-windows', text: 'Windows 平台' },
  { level: 3, id: 'node-macos', text: 'macOS 平台' },
  { level: 3, id: 'node-linux', text: 'Linux 平台' },
  { level: 2, id: 'faq', text: '常见问题' },
  { level: 2, id: 'welcome', text: '欢迎使用' },
  { level: 2, id: 'what-is-relay', text: '中转站是什么' },
  { level: 2, id: 'pricing', text: '价格说明' },
  { level: 2, id: 'register-account', text: '注册账号' },
  { level: 2, id: 'create-key', text: '创建专属 Key' },
  { level: 2, id: 'token-settings', text: '修改令牌设置' },
  { level: 2, id: 'model-selection', text: '模型选择' },
  { level: 2, id: 'recharge', text: '充值' },
  { level: 2, id: 'claude-code-hub', text: 'Claude Code Hub' },
  { level: 2, id: 'hapi-remote-control', text: 'Hapi 远程控制' },
  { level: 2, id: 'hapi-advanced-ip', text: 'Hapi 进阶：优选 IP 配置' },
  { level: 2, id: 'alma-client', text: 'Alma 客户端' },
  { level: 2, id: 'cherrystudio-client', text: 'CherryStudio 客户端' },
  { level: 2, id: 'opencode', text: 'OpenCode' },
  { level: 2, id: 'troubleshooting', text: '疑难杂症' },
  { level: 2, id: 'links', text: '友情链接' },
]

export const apihubGuideToc: ApihubGuideHeading[] = apihubGuideHeadings.filter((heading) => heading.level > 1)

function codeBlock(code: string, language = 'bash') {
  return `<div class="language-${language} vp-adaptive-theme"><button title="复制代码" class="copy">复制</button><span class="lang">${language}</span><pre><code>${code}</code></pre></div>`
}

const codexConfig = `model_provider = "APIHub"
model = "${SANDOG_DEFAULT_MODEL}"
model_reasoning_effort = "xhigh"
disable_response_storage = true
approval_policy = "on-request"
sandbox_mode = "danger-full-access"
model_supports_reasoning_summaries = true

[model_providers.APIHub]
name = "apihub"
base_url = "${apiBaseURL}"
wire_api = "responses"
requires_openai_auth = true`

const authJson = `{
  "OPENAI_API_KEY": "请替换为你在 APIHub 创建的 CodeX 专属 Key"
}`

export const apihubGuideArticle: ApihubGuideArticle = {
  title: 'CodeX 部署指南',
  kicker: 'APIHub Docs',
  summary: '企业级 AI 编码助手 - 完整部署手册',
  meta: { providerName, sourcePath: '/deploy/codex' },
  sections: [
    {
      id: 'quick-navigation',
      title: '快速导航',
      level: 2,
      blocks: [{ type: 'html', html: `<p>CodeX 是基于 GPT-5 架构的下一代智能编程助手，为开发者提供代码生成、理解与优化能力。</p><p>部署路径：系统环境配置 ➜ CLI 工具安装 ➜ API 集成 ➜ 开发环境就绪</p><table><thead><tr><th>资源</th><th>地址</th></tr></thead><tbody><tr><td>官方文档</td><td><a href="https://developers.openai.com/codex/" target="_blank" rel="noreferrer">developers.openai.com/codex</a></td></tr></tbody></table><div class="custom-block warning"><p class="custom-block-title">前置要求</p><p>请先完成 Node.js 环境安装和 CC-Switch 工具安装。</p></div>` }],
    },
    {
      id: 'cc-switch-config',
      title: '使用 CC-Switch 快速配置（推荐）',
      level: 2,
      blocks: [{ type: 'html', html: `<div class="custom-block warning"><p class="custom-block-title">前置条件</p><p>使用 CC-Switch 配置 CodeX 之前，请确保已通过 npm 全局安装 CodeX 工具。</p></div>${codeBlock(`npm install -g @openai/codex@latest
codex --version`)}<p>推荐使用 CC-Switch 快速配置工具进行图形化配置，无需手写命令行文件。</p>` }],
    },
    {
      id: 'config-steps',
      title: '配置步骤',
      level: 3,
      blocks: [{ type: 'html', html: `<h4>1. 启动 CC-Switch 并切换到 Codex 标签</h4><ol><li>打开 CC-Switch 应用程序</li><li>点击顶部的「Codex」标签页</li><li>点击右上角橙色「+」按钮添加新配置</li></ol><p><img src="${tu9}" alt="CC-Switch Codex 标签页"></p><h4>2. 填写 CodeX 提供商配置</h4><ol><li>提供商名称：建议填写 APIHub</li><li>Base URL?<code>${apiBaseURL}</code></li><li>API Key?粘贴你从 APIHub 平台获取的 CodeX 专用令牌（codex 令牌组）</li><li>Model?<code>${SANDOG_DEFAULT_MODEL}</code></li><li>其他配置：按需调整推理强度、网络访问等参数</li><li>点击「保存」按钮</li></ol><p><img src="${tu10}" alt="CC-Switch 添加 CodeX 配置"></p><p><img src="${tu11}" alt="CC-Switch CodeX 配置详情"></p><div class="custom-block tip"><p class="custom-block-title">提示</p><ul><li>CC-Switch 会自动创建 <code>~/.codex/config.toml</code> 和 <code>auth.json</code> 文件</li><li>可以添加多个提供商配置，随时切换</li><li>切换配置后，关闭并重启 CodeX 即可生效</li></ul></div><h4>3. 启用配置并使用</h4><ol><li>在配置列表中找到刚创建的 APIHub 配置</li><li>点击配置右侧的「当前使用」按钮</li><li>配置会被标记为「当前使用」状态</li><li>重启 CodeX，新配置即可生效</li></ol><h4>4. 系统托盘快速切换</h4><p>CC-Switch 支持通过系统托盘快速切换 CodeX 配置：</p><ul><li>右键点击系统托盘中的 CC-Switch 图标</li><li>在菜单中选择 Codex 分类</li><li>直接选择要切换到的配置</li><li>配置立即生效，无需打开主界面</li></ul><div class="custom-block warning"><p class="custom-block-title">注意事项</p><ul><li>务必从 APIHub 平台创建「codex」令牌组的专用密钥</li><li>CodeX 令牌与 Claude Code 令牌不通用</li><li>切换配置后需要重启 CodeX 才能生效</li><li>可在 CC-Switch 中测试 API 端点速度</li></ul></div>` }],
    },
    {
      id: 'manual-config',
      title: '手动命令行配置',
      level: 2,
      blocks: [{ type: 'html', html: `<p>如果不使用 CC-Switch，可以按下面的步骤手动配置 CodeX。</p>` }],
    },
    {
      id: 'node-windows',
      title: 'Windows 平台',
      level: 3,
      blocks: [{ type: 'html', html: `<h4>第一步：部署 CodeX 命令行工具</h4><p>以管理员权限启动命令提示符或 PowerShell，执行：</p>${codeBlock(`npm install -g @openai/codex@latest
codex --version`, 'powershell')}<h4>第二步：集成 APIHub API 服务</h4><ol><li>访问 APIHub 开发者控制台</li><li>完成账户注册或执行登录操作</li><li>导航至「API 密钥管理」模块</li><li>创建新密钥时，务必选择「codex」令牌组</li><li>安全保存生成的 API Key</li></ol><div class="custom-block warning"><p class="custom-block-title">安全提醒</p><p>CodeX 要求使用独立的令牌组配置，与 Claude Code 令牌体系完全隔离。</p></div><p>构建配置目录结构：</p>${codeBlock(`mkdir %USERPROFILE%\.codex
cd %USERPROFILE%\.codex`, 'powershell')}<p>编写配置文件：<code>config.toml</code></p>${codeBlock(codexConfig, 'toml')}<p>编写认证文件：<code>auth.json</code></p>${codeBlock(authJson, 'json')}<h4>第三步：初始化工作空间</h4>${codeBlock(`mkdir my-codex-project
cd my-codex-project
codex`, 'powershell')}` }],
    },
    {
      id: 'node-macos',
      title: 'macOS 平台',
      level: 3,
      blocks: [{ type: 'html', html: `<h4>部署 CodeX 工具</h4>${codeBlock(`npm install -g @openai/codex@latest
codex --version`)}<h4>集成 API 服务</h4><p>构建配置目录：</p>${codeBlock(`mkdir -p ~/.codex
cd ~/.codex`)}<p>编写 <code>config.toml</code> 配置：</p>${codeBlock(`cat > config.toml << 'EOF'
${codexConfig}
EOF`)}<p>编写 <code>auth.json</code> 认证配置：</p>${codeBlock(`cat > auth.json << 'EOF'
${authJson}
EOF`)}<h4>初始化工作空间</h4>${codeBlock(`mkdir my-codex-project
cd my-codex-project
codex`)}` }],
    },
    {
      id: 'node-linux',
      title: 'Linux 平台',
      level: 3,
      blocks: [{ type: 'html', html: `<h4>部署 CodeX 工具</h4>${codeBlock(`sudo npm install -g @openai/codex@latest
codex --version`)}<h4>集成 API 服务</h4><p>构建配置目录：</p>${codeBlock(`mkdir -p ~/.codex
cd ~/.codex`)}<p>编写 <code>config.toml</code> 配置：</p>${codeBlock(`cat > config.toml << 'EOF'
${codexConfig}
EOF`)}<p>编写 <code>auth.json</code> 认证配置：</p>${codeBlock(`cat > auth.json << 'EOF'
${authJson}
EOF`)}<h4>初始化工作空间</h4>${codeBlock(`mkdir my-codex-project
cd my-codex-project
codex`)}` }],
    },
    {
      id: 'faq',
      title: '常见问题',
      level: 2,
      blocks: [{ type: 'html', html: `<h3>CodeX 和 Claude Code 的令牌不通用？</h3><p>是的，两者使用不同的令牌组：</p><ul><li>Claude Code?使用 Claude Code 令牌组</li><li>CodeX?使用 <code>codex</code> 令牌组</li></ul><p>请在 APIHub 平台创建对应的专用令牌。</p><h3>配置文件放在哪里？</h3><ul><li>Windows?<code>%USERPROFILE%\.codex\</code></li><li>macOS/Linux?<code>~/.codex/</code></li></ul><h3>更多问题</h3><p>请查看常见问题或在 APIHub 站内获取帮助。</p>` }],
    },
    { id: 'welcome', title: '欢迎使用', level: 2, blocks: [{ type: 'html', html: `<p>这里是 APIHub 文档入门。如果你只需要配置 CodeX，请从本页上方的部署指南开始。</p>` }] },
    { id: 'what-is-relay', title: '中转站是什么', level: 2, blocks: [{ type: 'html', html: `<p>中转站将客户端请求统一转发到可用上游模型服务，客户端只需配置 Base URL 和 API Key。</p>` }] },
    { id: 'pricing', title: '价格说明', level: 2, blocks: [{ type: 'html', html: `<p>不同模型和分组的计费以 APIHub 后台展示为准。</p>` }] },
    { id: 'register-account', title: '注册账号', level: 2, blocks: [{ type: 'html', html: `<p>请先完成 APIHub 账号注册和登录，再创建 CodeX 专用密钥。</p>` }] },
    { id: 'create-key', title: '创建专属 Key', level: 2, blocks: [{ type: 'html', html: `<p>进入 API 密钥管理页创建新密钥，并务必选择 codex 令牌组。</p>` }] },
    { id: 'token-settings', title: '修改令牌设置', level: 2, blocks: [{ type: 'html', html: `<p>可按需调整密钥名称、分组、限额、IP 限制和启停状态。</p>` }] },
    { id: 'model-selection', title: '模型选择', level: 2, blocks: [{ type: 'html', html: `<p>CodeX 示例配置使用 <code>${SANDOG_DEFAULT_MODEL}</code>，也可按后台可用模型调整。</p>` }] },
    { id: 'recharge', title: '充值', level: 2, blocks: [{ type: 'html', html: `<p>长时间使用 CodeX 前请确认 APIHub 余额充足。</p>` }] },
    { id: 'claude-code-hub', title: 'Claude Code Hub', level: 2, blocks: [{ type: 'html', html: `<p>Claude Code Hub 可用于管理 Claude Code 相关配置，接入时同样使用 APIHub 的 Base URL 和专用 Key。</p>` }] },
    { id: 'hapi-remote-control', title: 'Hapi 远程控制', level: 2, blocks: [{ type: 'html', html: `<p>Hapi 远程控制中将接口地址设置为 <code>${apiBaseURL}</code>，密钥使用 APIHub 专用 Key。</p>` }] },
    { id: 'hapi-advanced-ip', title: 'Hapi 进阶：优选 IP 配置', level: 2, blocks: [{ type: 'html', html: `<p>如果为 Key 开启 IP 限制，请确保客户端出口 IP 已加入 APIHub 白名单。</p>` }] },
    { id: 'alma-client', title: 'Alma 客户端', level: 2, blocks: [{ type: 'html', html: `<p>Alma 客户端选择 OpenAI 兼容服务，填写 APIHub 接口地址和专用 Key。</p>` }] },
    { id: 'cherrystudio-client', title: 'CherryStudio 客户端', level: 2, blocks: [{ type: 'html', html: `<p>CherryStudio 中添加自定义 OpenAI 兼容供应商，名称建议填写 APIHub。</p>` }] },
    { id: 'opencode', title: 'OpenCode', level: 2, blocks: [{ type: 'html', html: `<p>OpenCode 接入方式与其他 OpenAI 兼容客户端类似：配置 APIHub 接口地址、Key 和模型名。</p>` }] },
    { id: 'troubleshooting', title: '疑难杂症', level: 2, blocks: [{ type: 'html', html: `<ul><li>无权限：检查 Key 是否启用以及是否选择 codex 令牌组。</li><li>余额不足：进入 APIHub 控制台确认余额或充值状态。</li></ul>` }] },
    { id: 'links', title: '友情链接', level: 2, blocks: [{ type: 'html', html: `<ul><li><a href="https://developers.openai.com/codex/" target="_blank" rel="noreferrer">OpenAI CodeX 官方文档</a></li><li><a href="https://nodejs.org/" target="_blank" rel="noreferrer">Node.js 官方网站</a></li></ul>` }] },
  ],
}

export const apihubGuideContentBlocks: ApihubGuideContentBlock[] = apihubGuideArticle.sections.map((section) => ({
  id: section.id,
  title: section.title,
  level: section.level,
  html: section.blocks.map((block) => block.html).join('\n'),
}))

export const apihubGuideArticleHtml = apihubGuideContentBlocks.map((block) => block.html).join('\n')

export const apihubGuideDocument: ApihubGuideDocument = {
  title: 'apihub使用教程',
  description: 'APIHub CodeX 中文使用教程，覆盖注册、密钥、模型、充值、Node.js、CodeX CLI 和第三方客户端配置。',
  providerName,
  article: apihubGuideArticle,
  articleHtml: apihubGuideArticleHtml,
  images: apihubGuideImages,
  sidebarGroups: apihubGuideSidebarGroups,
  headings: apihubGuideHeadings,
  toc: apihubGuideToc,
  contentBlocks: apihubGuideContentBlocks,
}
