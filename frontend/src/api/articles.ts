import type { Article } from '../types'

export async function fetchArticles(): Promise<Article[]> {
  const response = await fetch('/api/articles')
  if (!response.ok) {
    throw new Error(`Articles API failed: ${response.status}`)
  }
  return response.json()
}

export async function fetchArticle(slug: string): Promise<Article | null> {
  const response = await fetch(`/api/articles/${encodeURIComponent(slug)}`)
  if (response.status === 404) {
    return null
  }
  if (!response.ok) {
    throw new Error(`Article API failed: ${response.status}`)
  }
  return response.json()
}
