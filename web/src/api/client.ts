import createClient from 'openapi-fetch'
import type { paths } from '@/api/types.gen'

export const api = createClient<paths>({ baseUrl: '/' })
