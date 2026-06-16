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
  const url = buildDPoPUrl(config.baseURL || apiClient.defaults.baseURL || '', config.url || '')
  
  try {
    const proof = await getDPoPProof(method, url, currentAccessToken)
    config.headers['DPoP'] = proof
  } catch (err) {
    console.error('Failed to generate DPoP proof', err)
  }
  
  return config
})

const buildDPoPUrl = (baseURL: string, requestURL: string) => {
  const absoluteURL = new URL(`${baseURL}${requestURL}`, window.location.origin)
  absoluteURL.search = ''
  absoluteURL.hash = ''
  return absoluteURL.toString()
}
