import type { Article } from '../types'
import { staticAssetPath } from '../url'
import { HttpError, requestJSON, requestWithStaticFallback } from './request'

export async function fetchArticles(): Promise<Article[]> {
  return requestWithStaticFallback<Article[]>('/api/articles', 'data/articles.json')
}

export async function fetchArticle(slug: string): Promise<Article | null> {
  const encodedSlug = encodeURIComponent(slug)
  try {
    return await requestJSON<Article>(`/api/articles/${encodedSlug}`)
  } catch (apiError) {
    try {
      return await requestJSON<Article>(staticAssetPath(`data/articles/${encodedSlug}.json`))
    } catch (staticError) {
      if (apiError instanceof HttpError && apiError.status === 404) return null
      throw staticError
    }
  }
}
