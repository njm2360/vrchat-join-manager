import createClient from 'openapi-fetch'
import type { paths } from '@/api/types.gen'

const baseUrl = document.baseURI.replace(/\/$/, '')

export const api = createClient<paths>({ baseUrl })
