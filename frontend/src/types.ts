export interface SiteData {
  person: Person
  hero: Hero
  contacts: Contact[]
  navigation: NavItem[]
  metrics: Metric[]
  skills: SkillGroup[]
  experiences: Experience[]
  projects: Project[]
  research: ResearchItem[]
  coding: CodingProfile
  reading: ReadingItem[]
  blogIdeas: BlogItem[]
  techRadar: RadarItem[]
  activity: ActivityDay[]
  serviceStack: StackLayer[]
  readingLinks: ReadingLink[]
  labels: SiteLabels
  uiText: Record<string, string>
}

export interface Article {
  slug: string
  category: string
  date: string
  title: string
  excerpt: string
  tags: string[]
  body?: string[]
  blocks: ArticleBlock[]
}

export interface ArticleBlock {
  type: 'heading' | 'paragraph' | 'list' | 'code' | 'image' | 'callout' | 'links' | string
  level?: number
  text?: string
  items?: string[]
  language?: string
  filename?: string
  code?: string
  src?: string
  alt?: string
  caption?: string
  title?: string
  links?: ArticleLink[]
}

export interface ArticleLink {
  label: string
  href: string
  note?: string
}

export interface Person {
  name: string
  english: string
  title: string
  school: string
  major: string
  location: string
  intentions: string[]
  summary: string
}

export interface Hero {
  eyebrow: string
  headline: string
  subheadline: string
  focus: string[]
}

export interface Contact {
  type: string
  label: string
  value: string
  href?: string
  note?: string
}

export interface NavItem {
  label: string
  to: string
}

export interface Metric {
  label: string
  value: string
  suffix: string
  note: string
}

export interface SkillGroup {
  name: string
  level: number
  description: string
  keywords: string[]
}

export interface Experience {
  company: string
  role: string
  period: string
  place: string
  stack: string[]
  points: string[]
}

export interface Project {
  name: string
  role: string
  period: string
  link?: string
  stack: string[]
  highlights: string[]
  impact: string
}

export interface ResearchItem {
  title: string
  status: string
  tags: string[]
  detail: string
}

export interface CodingProfile {
  headline: string
  metrics: CodingMetric[]
  tracks: CodingTrack[]
}

export interface CodingMetric {
  label: string
  value: string
  unit: string
  trend: string
}

export interface CodingTrack {
  name: string
  progress: number
  note: string
}

export interface ReadingItem {
  topic: string
  cadence: string
  keywords: string[]
  summary: string
}

export interface BlogItem {
  title: string
  type: string
  tags: string[]
  brief: string
}

export interface RadarItem {
  name: string
  value: number
}

export interface ActivityDay {
  day: string
  score: number
  kind: string
}

export interface StackLayer {
  name: string
  items: string[]
}

export interface ReadingLink {
  label: string
  type: string
  href: string
  note: string
}

export interface SiteLabels {
  brandInitial: string
  profileGreeting: string
  articlesEyebrow: string
  articlesTitle: string
  tagsEyebrow: string
  tagsTitle: string
  readingEyebrow: string
  readingTitle: string
}

export interface Signal {
  level: string
  label: string
  detail: string
  progress: number
  timestamp: string
}

export interface Checkin {
  date: string
  note: string
  checkedAt: string
  updatedAt: string
}

export interface TodoTask {
  id: string
  title: string
  done: boolean
  completedAt?: string
  sortOrder: number
}

export interface TodoDayStatus {
  date: string
  tasks: TodoTask[]
  total: number
  completed: number
  allDone: boolean
}

export interface TodoTaskDraft {
  id?: string
  title: string
}

export interface AdminSession {
  authenticated: boolean
  username?: string
  token?: string
}

export interface SiteContent {
  profile: SiteData
  articles: Article[]
  updatedAt: string
}
