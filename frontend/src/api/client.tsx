import axios from 'axios'
import { getDPoPProof } from '@/utils/dpop'

export const apiClient = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true,
})

let currentAccessToken = ''

export const setAccessToken = (token: string) => {
  currentAccessToken = token
}

apiClient.interceptors.request.use(async (config) => {
  const method = config.method || 'GET'
  let url = `${config.baseURL || ''}${config.url || ''}`
  
  const queryIndex = url.indexOf('?')
  if (queryIndex !== -1) {
    url = url.substring(0, queryIndex)
  }
  
  try {
    const proof = await getDPoPProof(method, url, currentAccessToken)
    config.headers['DPoP'] = proof
  } catch (err) {
    console.error('Failed to generate DPoP proof', err)
  }
  
  return config
})
