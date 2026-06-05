<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  ArrowRight,
  ArrowLeft,
  Calendar,
  ChatRound,
  Check,
  CopyDocument,
  Link,
  Message,
  Promotion,
  Star,
} from '@element-plus/icons-vue'
import hljs from 'highlight.js/lib/common'

import { fetchArticle, fetchArticles } from './api/articles'
import { fetchProfile } from './api/profile'
import { fallbackProfile } from './data/fallback'
import type { Article, ArticleBlock, Project, SiteData } from './types'
import { staticAssetPath, stripBasePath, withBasePath } from './url'

interface ClickBubble {
  id: number
  x: number
  y: number
  size: number
  hue: number
}

interface CalendarCell {
  key: string
  day: number
  isCurrentMonth: boolean
  isToday: boolean
  isFuture: boolean
  checked: boolean
}

interface ReaderSection {
  id: string
  title: string
  blocks: ArticleBlock[]
}

interface ReadingLink {
  label: string
  type: string
  href: string
  note: string
}

interface ContactAction {
  type: string
  label: string
  value: string
  href?: string
  note: string
}

interface WorkLink {
  label: string
  href: string
  note: string
  stack: string[]
}

const CHECKIN_STORAGE_KEY = 'mengqing-homepage-checkins'
const today = new Date()
const todayKey = formatDateKey(today)
const weekdayLabels = ['一', '二', '三', '四', '五', '六', '日']

const profile = ref<SiteData>(fallbackProfile)
const articles = ref<Article[]>([])
const selectedArticle = ref<Article | null>(null)
const loading = ref(true)
const activeTopic = ref('全部')
const navHidden = ref(false)
const navFloating = ref(false)
const openPanel = ref<'contact' | 'works' | null>(null)
const clickBubbles = ref<ClickBubble[]>([])
const checkedDays = ref<string[]>([])
const calendarMonth = ref(new Date(today.getFullYear(), today.getMonth(), 1))
const currentPath = ref(stripBasePath(window.location.pathname))

let lastScrollY = 0
let scrollTicking = false
let clickBubbleId = 0
const clickBubbleTimers: number[] = []

const direction = '深度学习视觉方向，主要使用遥感数据。'

const projectLink = computed(() => profile.value.contacts.find((item) => item.type === 'link'))
const workLinks = computed<WorkLink[]>(() => {
  const linkedProjects = profile.value.projects
    .filter((project) => project.link)
    .map(projectToWorkLink)

  if (linkedProjects.length > 0) return linkedProjects

  return projectLink.value?.href
    ? [
        {
          label: projectLink.value.label || '在线作品',
          href: projectLink.value.href,
          note: projectLink.value.value || '项目入口',
          stack: ['Project'],
        },
      ]
    : []
})
const contactActions = computed<ContactAction[]>(() => {
  const mail = profile.value.contacts.find((item) => item.type === 'mail')
  const actions: ContactAction[] = [
    {
      type: 'qq',
      label: 'QQ',
      value: '2358164625',
      href: 'https://qm.qq.com/cgi-bin/qm/qr?k=2358164625',
      note: '点击复制号码',
    },
    {
      type: 'wechat',
      label: '微信',
      value: 'a2358164625',
      note: '点击复制微信号',
    },
  ]

  if (mail) {
    actions.push({
      type: 'mail',
      label: mail.label || '邮箱',
      value: mail.value,
      href: mail.href || `mailto:${mail.value}`,
      note: '点击复制邮箱',
    })
  }

  return actions
})
const readingLinks: ReadingLink[] = [
  {
    label: 'ZYYO/homepage',
    type: 'GitHub',
    href: 'https://github.com/ZYYO666/homepage',
    note: '主页动效与布局参考',
  },
  {
    label: 'iqboshi/platform',
    type: 'GitHub',
    href: 'https://github.com/iqboshi/platform',
    note: '当前项目源码',
  },
  {
    label: 'React Flow Docs',
    type: 'Docs',
    href: 'https://reactflow.dev/',
    note: '流程画布',
  },
]

const siteStats = computed(() => [
  { label: '在线作品', value: `${workLinks.value.length}` },
  { label: '项目帖', value: `${articles.value.length}` },
  { label: '论文', value: '3' },
  { label: '爱好', value: '编程' },
])

const filteredArticles = computed(() => {
  if (activeTopic.value === '全部') return articles.value
  return articles.value.filter((article) => {
    const source = `${article.category} ${article.title} ${article.excerpt} ${article.tags.join(' ')}`
    return source.toLowerCase().includes(activeTopic.value.toLowerCase())
  })
})

const allTags = computed(() => {
  const tags = new Set<string>()
  articles.value.forEach((article) => article.tags.forEach((tag) => tags.add(tag)))
  return Array.from(tags).slice(0, 16)
})

const isPostPage = computed(() => currentPath.value.startsWith('/post/'))
const routePostSlug = computed(() => {
  const match = currentPath.value.match(/^\/post\/([^/?#]+)/)
  return match ? safeDecode(match[1]) : ''
})

const checkedDaySet = computed(() => new Set(checkedDays.value))
const monthTitle = computed(() => `${calendarMonth.value.getFullYear()} 年 ${calendarMonth.value.getMonth() + 1} 月`)
const isTodayChecked = computed(() => checkedDaySet.value.has(todayKey))
const calendarCells = computed(() => buildCalendarCells(calendarMonth.value, checkedDaySet.value))
const monthlyCheckins = computed(() => calendarCells.value.filter((cell) => cell.isCurrentMonth && cell.checked).length)
const isNextMonthDisabled = computed(() => {
  const year = calendarMonth.value.getFullYear()
  const month = calendarMonth.value.getMonth()
  return year > today.getFullYear() || (year === today.getFullYear() && month >= today.getMonth())
})
const selectedArticleIndex = computed(() => {
  const currentArticle = selectedArticle.value
  if (!currentArticle) return -1

  return articles.value.findIndex((article) => article.slug === currentArticle.slug)
})
const previousArticle = computed(() => (selectedArticleIndex.value > 0 ? articles.value[selectedArticleIndex.value - 1] : null))
const nextArticle = computed(() => {
  const nextIndex = selectedArticleIndex.value + 1
  return selectedArticleIndex.value >= 0 && nextIndex < articles.value.length ? articles.value[nextIndex] : null
})
const readerSections = computed<ReaderSection[]>(() => {
  if (!selectedArticle.value) return []

  const sections: ReaderSection[] = []
  let current: ReaderSection | null = null

  articleBlocks(selectedArticle.value).forEach((block, index) => {
    if (block.type === 'heading') {
      current = {
        id: slugFromText(block.text || `section-${index}`),
        title: block.text || `Section ${sections.length + 1}`,
        blocks: [],
      }
      sections.push(current)
      return
    }

    if (!current) {
      current = { id: 'overview', title: '项目概览', blocks: [] }
      sections.push(current)
    }
    current.blocks.push(block)
  })

  return sections
})
const readerWordCount = computed(() => {
  if (!selectedArticle.value) return 0
  return [
    selectedArticle.value.title,
    selectedArticle.value.excerpt,
    ...articleBlocks(selectedArticle.value).flatMap(blockTextParts),
  ].join('').length
})
const readerReadTime = computed(() => `${Math.max(1, Math.ceil(readerWordCount.value / 450))} min`)
const streakDays = computed(() => {
  if (!isTodayChecked.value) return 0

  let streak = 0
  const cursor = new Date(today.getFullYear(), today.getMonth(), today.getDate())
  while (checkedDaySet.value.has(formatDateKey(cursor))) {
    streak += 1
    cursor.setDate(cursor.getDate() - 1)
  }
  return streak
})

function articleBlocks(article: Article): ArticleBlock[] {
  if (Array.isArray(article.blocks) && article.blocks.length > 0) {
    return article.blocks
  }
  return (article.body ?? []).map((text) => ({ type: 'paragraph', text }))
}

function blockTextParts(block: ArticleBlock): string[] {
  const parts = [block.text, block.title, block.caption, block.code, ...(block.items ?? [])]
  block.links?.forEach((item) => parts.push(item.label, item.note))
  return parts.filter((item): item is string => Boolean(item))
}

function slugFromText(value: string) {
  const normalized = value.trim().toLowerCase().replace(/[^\p{L}\p{N}]+/gu, '-').replace(/^-|-$/g, '')
  return normalized || 'section'
}

function sectionBlockKey(section: ReaderSection, block: ArticleBlock, index: number) {
  return `${section.id}-${block.type}-${block.text ?? block.title ?? block.filename ?? index}`
}

function projectToWorkLink(project: Project): WorkLink {
  return {
    label: project.name,
    href: project.link || '#',
    note: project.impact || project.role || project.period || '项目链接',
    stack: project.stack.slice(0, 4),
  }
}

function codeLanguage(language?: string) {
  const value = (language || 'text').trim().toLowerCase()
  const aliases: Record<string, string> = {
    bat: 'dos',
    cmd: 'dos',
    shell: 'bash',
    sh: 'bash',
    ps1: 'powershell',
    text: 'plaintext',
    txt: 'plaintext',
  }
  return aliases[value] || value
}

function highlightedCode(block: ArticleBlock) {
  const code = block.code || ''
  const language = codeLanguage(block.language)
  if (language !== 'plaintext' && hljs.getLanguage(language)) {
    return hljs.highlight(code, { language, ignoreIllegals: true }).value
  }
  return hljs.highlight(code, { language: 'plaintext', ignoreIllegals: true }).value
}

function formatDateKey(date: Date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function buildCalendarCells(monthDate: Date, checkedSet: Set<string>): CalendarCell[] {
  const year = monthDate.getFullYear()
  const month = monthDate.getMonth()
  const firstDay = new Date(year, month, 1)
  const leadingDays = (firstDay.getDay() + 6) % 7
  const monthDays = new Date(year, month + 1, 0).getDate()
  const cellCount = leadingDays + monthDays > 35 ? 42 : 35

  return Array.from({ length: cellCount }, (_, index) => {
    const date = new Date(year, month, index - leadingDays + 1)
    const key = formatDateKey(date)
    return {
      key,
      day: date.getDate(),
      isCurrentMonth: date.getMonth() === month,
      isToday: key === todayKey,
      isFuture: key > todayKey,
      checked: checkedSet.has(key),
    }
  })
}

function loadCheckins() {
  try {
    const raw = window.localStorage.getItem(CHECKIN_STORAGE_KEY)
    if (!raw) return

    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return

    checkedDays.value = parsed
      .filter((item): item is string => typeof item === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(item))
      .slice(-366)
  } catch (error) {
    console.warn(error)
  }
}

function saveCheckins() {
  try {
    window.localStorage.setItem(CHECKIN_STORAGE_KEY, JSON.stringify(checkedDays.value))
  } catch (error) {
    console.warn(error)
  }
}

function checkInToday() {
  if (isTodayChecked.value) {
    ElMessage.info('今天已经签到')
    return
  }

  checkedDays.value = Array.from(new Set([...checkedDays.value, todayKey])).sort()
  saveCheckins()
  ElMessage.success('签到成功')
}

function changeCalendarMonth(offset: number) {
  const next = new Date(calendarMonth.value.getFullYear(), calendarMonth.value.getMonth() + offset, 1)
  const currentMonth = new Date(today.getFullYear(), today.getMonth(), 1)
  if (next > currentMonth) return

  calendarMonth.value = next
}

function postHref(article: Article) {
  return withBasePath(`/post/${encodeURIComponent(article.slug)}/`)
}

function mediaSrc(path?: string) {
  return staticAssetPath(path || '')
}

function safeDecode(value: string) {
  try {
    return decodeURIComponent(value)
  } catch {
    return value
  }
}

function setTopic(topic: string) {
  activeTopic.value = activeTopic.value === topic && topic !== '全部' ? '全部' : topic
}

function togglePanel(panel: 'contact' | 'works') {
  openPanel.value = openPanel.value === panel ? null : panel
}

function closePanel() {
  openPanel.value = null
}

function openArticle(article: Article) {
  window.open(postHref(article), '_blank', 'noopener,noreferrer')
}

function switchArticle(article: Article | null) {
  if (!article) return

  selectedArticle.value = article
  window.history.pushState({}, '', postHref(article))
  currentPath.value = stripBasePath(window.location.pathname)
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function scrollReaderSection(id: string) {
  const target = document.getElementById(`reader-${id}`)
  if (!target) return

  window.scrollTo({
    top: target.getBoundingClientRect().top + window.scrollY - 24,
    behavior: 'smooth',
  })
}

function scrollToSection(target: string) {
  navHidden.value = false
  document.querySelector(target)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function copyValue(value: string, label: string) {
  navigator.clipboard
    .writeText(value)
    .then(() => ElMessage.success(`${label}已复制`))
    .catch(() => ElMessage.warning('复制失败，请手动复制'))
}

function handleContactAction(item: ContactAction) {
  copyValue(item.value, item.label)
}

function openWorkLink(item: WorkLink) {
  window.open(item.href, '_blank', 'noopener,noreferrer')
  closePanel()
}

async function syncSelectedArticleFromRoute() {
  if (!isPostPage.value) {
    selectedArticle.value = null
    return
  }

  const slug = routePostSlug.value
  if (!slug) {
    selectedArticle.value = null
    return
  }

  const cachedArticle = articles.value.find((article) => article.slug === slug)
  if (cachedArticle) {
    selectedArticle.value = cachedArticle
    return
  }

  try {
    selectedArticle.value = await fetchArticle(slug)
  } catch (error) {
    console.warn(error)
    selectedArticle.value = null
    ElMessage.warning('文章数据库接口暂不可用')
  }
}

async function loadInitialData() {
  loading.value = true

  const [profileResult, articlesResult] = await Promise.allSettled([fetchProfile(), fetchArticles()])

  if (profileResult.status === 'fulfilled') {
    profile.value = profileResult.value
  } else {
    console.warn(profileResult.reason)
    ElMessage.warning('资料接口暂不可用，已展示本地默认资料')
  }

  if (articlesResult.status === 'fulfilled') {
    articles.value = articlesResult.value
  } else {
    console.warn(articlesResult.reason)
    articles.value = []
    ElMessage.warning('文章数据库接口暂不可用')
  }

  await syncSelectedArticleFromRoute()
  loading.value = false
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    navHidden.value = false
    closePanel()
  }
}

function syncCurrentPath() {
  currentPath.value = stripBasePath(window.location.pathname)
  void syncSelectedArticleFromRoute()
}

function updateNavOnScroll() {
  const current = window.scrollY
  const delta = current - lastScrollY
  navFloating.value = current > 24

  if (current < 80) {
    navHidden.value = false
  } else if (delta > 8) {
    navHidden.value = true
  } else if (delta < -8) {
    navHidden.value = false
  }

  lastScrollY = current
  scrollTicking = false
}

function handleScroll() {
  if (scrollTicking) return
  scrollTicking = true
  window.requestAnimationFrame(updateNavOnScroll)
}

function spawnClickBubble(event: PointerEvent) {
  if (event.button !== 0) return

  const id = clickBubbleId++
  const size = 22 + Math.round(Math.random() * 16)
  const hue = [202, 165, 218][id % 3]
  clickBubbles.value = [...clickBubbles.value.slice(-8), { id, x: event.clientX, y: event.clientY, size, hue }]

  const timer = window.setTimeout(() => {
    clickBubbles.value = clickBubbles.value.filter((bubble) => bubble.id !== id)
  }, 920)
  clickBubbleTimers.push(timer)
}

onMounted(() => {
  loadInitialData()
  loadCheckins()
  lastScrollY = window.scrollY
  updateNavOnScroll()
  window.addEventListener('keydown', handleKeydown)
  window.addEventListener('popstate', syncCurrentPath)
  window.addEventListener('scroll', handleScroll, { passive: true })
  window.addEventListener('pointerdown', spawnClickBubble, { passive: true })
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('popstate', syncCurrentPath)
  window.removeEventListener('scroll', handleScroll)
  window.removeEventListener('pointerdown', spawnClickBubble)
  clickBubbleTimers.forEach((timer) => window.clearTimeout(timer))
})
</script>

<template>
  <main v-if="isPostPage" class="blog-home post-page" v-loading="loading">
    <div class="bubble-stage" aria-hidden="true">
      <span
        v-for="bubble in clickBubbles"
        :key="bubble.id"
        class="click-bubble"
        :style="{
          left: `${bubble.x}px`,
          top: `${bubble.y}px`,
          width: `${bubble.size}px`,
          height: `${bubble.size}px`,
          '--bubble-hue': bubble.hue,
        }"
      ></span>
    </div>

    <div v-if="selectedArticle" class="reader-shell">
      <aside class="reader-sidebar">
        <a class="reader-back" :href="withBasePath('/')">
          <el-icon><ArrowLeft /></el-icon>
          <span>返回主页</span>
        </a>
        <div class="reader-author">
          <span>ZMQ</span>
          <div>
            <strong>{{ profile.person.name }}</strong>
            <small>{{ selectedArticle.category }} / {{ readerReadTime }}</small>
          </div>
        </div>
        <nav class="reader-toc" aria-label="文章目录">
          <p>目录</p>
          <button
            v-for="section in readerSections"
            :key="section.id"
            type="button"
            @click="scrollReaderSection(section.id)"
          >
            {{ section.title }}
          </button>
        </nav>
      </aside>

      <article class="reader-main">
        <header class="reader-hero">
          <p class="eyebrow">{{ selectedArticle.category }}</p>
          <h1>{{ selectedArticle.title }}</h1>
          <p>{{ selectedArticle.excerpt }}</p>
          <div class="reader-meta">
            <span>{{ selectedArticle.date }}</span>
            <span>{{ readerWordCount }} 字</span>
            <span>{{ readerReadTime }}</span>
          </div>
          <div class="tag-row">
            <el-tag v-for="tag in selectedArticle.tags" :key="tag" size="small">{{ tag }}</el-tag>
          </div>
        </header>

        <section
          v-for="section in readerSections"
          :id="`reader-${section.id}`"
          :key="section.id"
          class="reader-section"
        >
          <h2>{{ section.title }}</h2>
          <template v-for="(block, blockIndex) in section.blocks" :key="sectionBlockKey(section, block, blockIndex)">
            <p v-if="block.type === 'paragraph'" class="reader-paragraph">{{ block.text }}</p>

            <ul v-else-if="block.type === 'list'" class="reader-list">
              <li v-for="item in block.items" :key="item">{{ item }}</li>
            </ul>

            <figure v-else-if="block.type === 'image'" class="reader-media">
              <img
                :src="mediaSrc(block.src)"
                :alt="block.alt || block.caption || selectedArticle.title"
                loading="eager"
                decoding="async"
              />
              <figcaption v-if="block.caption">{{ block.caption }}</figcaption>
            </figure>

            <div v-else-if="block.type === 'code'" class="reader-code-block">
              <div class="reader-code-head">
                <span>{{ block.filename || 'snippet' }}</span>
                <small>{{ block.language || 'text' }}</small>
              </div>
              <pre><code :class="`language-${codeLanguage(block.language)}`" v-html="highlightedCode(block)"></code></pre>
            </div>

            <div v-else-if="block.type === 'callout'" class="reader-callout">
              <strong v-if="block.title">{{ block.title }}</strong>
              <p>{{ block.text }}</p>
            </div>

            <div v-else-if="block.type === 'links'" class="reader-links">
              <strong v-if="block.title">{{ block.title }}</strong>
              <a v-for="item in block.links" :key="item.href" :href="item.href" target="_blank" rel="noreferrer">
                <span>{{ item.label }}</span>
                <small>{{ item.note }}</small>
              </a>
            </div>

            <p v-else-if="block.text" class="reader-paragraph">{{ block.text }}</p>
          </template>
        </section>

        <footer class="reader-footer">
          <div class="reader-note">
            <span>Article</span>
            <strong>{{ selectedArticle.category }} · {{ selectedArticle.date }}</strong>
          </div>
          <div class="reader-pager">
            <button type="button" :disabled="!previousArticle" @click="switchArticle(previousArticle)">
              <span>上一篇</span>
              <strong>{{ previousArticle?.title ?? '没有更早文章' }}</strong>
            </button>
            <button type="button" :disabled="!nextArticle" @click="switchArticle(nextArticle)">
              <span>下一篇</span>
              <strong>{{ nextArticle?.title ?? '没有更新文章' }}</strong>
            </button>
          </div>
        </footer>
      </article>
    </div>

    <section v-else class="post-missing">
      <a class="reader-back" :href="withBasePath('/')">
        <el-icon><ArrowLeft /></el-icon>
        <span>返回主页</span>
      </a>
      <div class="reader-main">
        <p class="eyebrow">Not Found</p>
        <h1>文章不存在</h1>
        <p>这篇内容可能已经改名，回到主页重新打开一篇文章会更稳。</p>
      </div>
    </section>
  </main>

  <main v-else class="blog-home" v-loading="loading">
    <div class="bubble-stage" aria-hidden="true">
      <span
        v-for="bubble in clickBubbles"
        :key="bubble.id"
        class="click-bubble"
        :style="{
          left: `${bubble.x}px`,
          top: `${bubble.y}px`,
          width: `${bubble.size}px`,
          height: `${bubble.size}px`,
          '--bubble-hue': bubble.hue,
        }"
      ></span>
    </div>

    <header class="site-nav" :class="{ 'nav-hidden': navHidden, 'nav-floating': navFloating }">
      <button class="brand" type="button" @click="scrollToSection('#top')">
        <span>Z</span>
        <strong>张孟庆</strong>
      </button>
      <nav aria-label="主页导航">
        <button type="button" @click="scrollToSection('#articles')">项目帖</button>
        <button type="button" @click="scrollToSection('#garden')">签到</button>
        <button type="button" @click="scrollToSection('#about')">关于</button>
      </nav>
    </header>

    <div id="top" class="blog-shell">
      <aside class="left-rail">
        <section id="about" class="profile-card">
          <div class="initial-mark">ZMQ</div>
          <p class="hello">Hello, I am</p>
          <h1>{{ profile.person.name }}</h1>
          <p class="direction">{{ direction }}</p>
          <div class="profile-tags">
            <span>Go</span>
            <span>Vue</span>
            <span>视觉</span>
            <span>PyTorch</span>
          </div>
          <div class="profile-actions">
            <button
              class="profile-action-button primary"
              type="button"
              :aria-expanded="openPanel === 'contact'"
              @click="togglePanel('contact')"
            >
              <el-icon><Message /></el-icon>
              <span>联系我</span>
            </button>
            <button
              class="profile-action-button"
              type="button"
              :aria-expanded="openPanel === 'works'"
              :disabled="workLinks.length === 0"
              @click="togglePanel('works')"
            >
              <el-icon><Link /></el-icon>
              <span>在线作品</span>
            </button>

            <Transition name="action-panel">
              <div v-if="openPanel === 'contact'" class="action-panel contact-panel">
                <button
                  v-for="item in contactActions"
                  :key="item.type"
                  class="contact-item"
                  type="button"
                  @click="handleContactAction(item)"
                >
                  <span class="item-icon">
                    <el-icon v-if="item.type === 'mail'"><Message /></el-icon>
                    <el-icon v-else-if="item.type === 'wechat'"><ChatRound /></el-icon>
                    <el-icon v-else><CopyDocument /></el-icon>
                  </span>
                  <span class="item-copy">
                    <strong>{{ item.label }}</strong>
                    <small>{{ item.value }}</small>
                  </span>
                  <em>{{ item.note }}</em>
                </button>
              </div>
            </Transition>

            <Transition name="action-panel">
              <div v-if="openPanel === 'works'" class="action-panel works-panel">
                <button
                  v-for="item in workLinks"
                  :key="item.href"
                  class="work-item"
                  type="button"
                  @click="openWorkLink(item)"
                >
                  <span class="item-icon">
                    <el-icon><Promotion /></el-icon>
                  </span>
                  <span class="item-copy">
                    <strong>{{ item.label }}</strong>
                  </span>
                  <span class="work-open-icon">
                    <el-icon><ArrowRight /></el-icon>
                  </span>
                </button>
              </div>
            </Transition>
          </div>
        </section>

        <section class="stat-card">
          <article v-for="item in siteStats" :key="item.label">
            <strong>{{ item.value }}</strong>
            <span>{{ item.label }}</span>
          </article>
        </section>

        <section class="reading-card">
          <div class="card-section-heading compact-heading">
            <div>
              <p class="eyebrow">Reading</p>
              <h2>最近在读</h2>
            </div>
            <div class="heading-side">
              <span>{{ readingLinks.length }} 条</span>
            </div>
          </div>
          <div class="reading-links">
            <a v-for="item in readingLinks" :key="item.href" :href="item.href" target="_blank" rel="noreferrer">
              <span>{{ item.type }}</span>
              <strong>{{ item.label }}</strong>
              <small>{{ item.note }}</small>
            </a>
          </div>
        </section>
      </aside>

      <section class="content-flow">
        <section id="articles" class="feed-section">
          <div v-if="filteredArticles.length" class="article-list">
            <a
              v-for="(article, index) in filteredArticles"
              :key="`${article.category}-${article.title}`"
              class="post-card"
              :href="postHref(article)"
              target="_blank"
              rel="noopener noreferrer"
              :aria-label="`新页面打开文章：${article.title}`"
              @click.prevent="openArticle(article)"
            >
              <div v-if="index === 0" class="card-section-heading">
                <div>
                  <p class="eyebrow">Project Posts</p>
                  <h2>我的项目</h2>
                </div>
                <div class="heading-side">
                  <span>{{ filteredArticles.length }} / {{ articles.length }} 篇</span>
                </div>
              </div>
              <div class="post-meta">
                <span>{{ article.category }}</span>
                <time>{{ article.date }}</time>
              </div>
              <h3>{{ article.title }}</h3>
              <p>{{ article.excerpt }}</p>
              <div class="card-bottom">
                <div class="tag-row">
                  <el-tag v-for="tag in article.tags.slice(0, 4)" :key="tag" size="small">{{ tag }}</el-tag>
                </div>
                <span class="card-action">
                  阅读
                  <el-icon><ArrowRight /></el-icon>
                </span>
              </div>
            </a>
          </div>

          <div v-else class="empty-state">
            <div class="card-section-heading">
              <div>
                <p class="eyebrow">Project Posts</p>
                <h2>我的项目</h2>
              </div>
              <div class="heading-side">
                <span>0 / {{ articles.length }} 篇</span>
              </div>
            </div>
            <strong>没有匹配的项目帖</strong>
            <p>换一个标签，或者先看全部项目。</p>
            <button type="button" @click="setTopic('全部')">查看全部</button>
          </div>
        </section>
      </section>

      <aside class="right-rail">
        <section class="mini-card">
          <div class="card-section-heading compact-heading">
            <div>
              <p class="eyebrow">Tags</p>
              <h2>标签</h2>
            </div>
            <div class="heading-side">
              <span>{{ allTags.length }} 个</span>
            </div>
          </div>
          <div class="cloud">
            <button
              class="clear-filter"
              :class="{ active: activeTopic === '全部' }"
              type="button"
              :aria-pressed="activeTopic === '全部'"
              @click="setTopic('全部')"
            >
              全部
            </button>
            <button
              v-for="tag in allTags"
              :key="tag"
              :class="{ active: activeTopic === tag }"
              type="button"
              :aria-pressed="activeTopic === tag"
              @click="setTopic(tag)"
            >
              {{ tag }}
            </button>
          </div>
        </section>

        <section id="garden" class="mini-card checkin-card">
          <div class="card-section-heading compact-heading">
            <div>
              <p class="eyebrow">Check-in</p>
              <h2>签到日历</h2>
            </div>
            <div class="heading-side">
              <span>{{ isTodayChecked ? '已签到' : '今日未签' }}</span>
            </div>
          </div>

          <div class="calendar-nav">
            <button type="button" aria-label="上个月" @click="changeCalendarMonth(-1)">
              <el-icon><ArrowLeft /></el-icon>
            </button>
            <strong>{{ monthTitle }}</strong>
            <button type="button" aria-label="下个月" :disabled="isNextMonthDisabled" @click="changeCalendarMonth(1)">
              <el-icon><ArrowRight /></el-icon>
            </button>
          </div>

          <div class="checkin-overview">
            <div>
              <span>本月</span>
              <strong>{{ monthlyCheckins }} 天</strong>
            </div>
            <div>
              <span>连续</span>
              <strong>{{ streakDays }} 天</strong>
            </div>
          </div>

          <div class="calendar-weekdays" aria-hidden="true">
            <span v-for="weekday in weekdayLabels" :key="weekday">{{ weekday }}</span>
          </div>
          <div class="checkin-calendar" aria-label="签到日历">
            <button
              v-for="cell in calendarCells"
              :key="cell.key"
              class="checkin-day"
              :class="{
                muted: !cell.isCurrentMonth,
                today: cell.isToday,
                checked: cell.checked,
                future: cell.isFuture,
              }"
              :disabled="!cell.isToday || cell.checked"
              :title="`${cell.key}${cell.checked ? ' 已签到' : ''}`"
              type="button"
              :aria-label="`${cell.key}${cell.checked ? ' 已签到' : ' 未签到'}`"
              @click="checkInToday"
            >
              <el-icon v-if="cell.checked"><Check /></el-icon>
              <span v-else>{{ cell.day }}</span>
            </button>
          </div>

          <button class="checkin-action" type="button" :disabled="isTodayChecked" @click="checkInToday">
            <el-icon><Calendar /></el-icon>
            <span>{{ isTodayChecked ? '今天已签到' : '今日签到' }}</span>
          </button>
        </section>
      </aside>
    </div>

    <footer class="site-footer">
      <span>© 2026 {{ profile.person.english }}. Built with Go and Vue.</span>
      <button type="button" @click="scrollToSection('#top')">
        回到顶部
        <el-icon><Star /></el-icon>
      </button>
    </footer>
  </main>
</template>
