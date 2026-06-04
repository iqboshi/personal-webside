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

export interface Signal {
  level: string
  label: string
  detail: string
  progress: number
  timestamp: string
}
