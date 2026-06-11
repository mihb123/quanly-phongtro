import { get, set } from 'idb-keyval'
import { SignJWT, exportJWK, calculateJwkThumbprint } from 'jose'

const DPOP_PRIVATE_KEY = 'dpop_private_key'
const DPOP_PUBLIC_KEY = 'dpop_public_key'
const DPOP_JKT = 'dpop_jkt'

let keyPairPromise: Promise<void> | null = null

/**
 * Ensures a DPoP key pair exists in IndexedDB.
 * Generates a new ECDSA P-256 key pair if it doesn't exist.
 */
export async function ensureDPoPKeyPair(): Promise<void> {
  if (keyPairPromise) {
    return keyPairPromise
  }

  keyPairPromise = (async () => {
    try {
      let privateKey = await get<CryptoKey>(DPOP_PRIVATE_KEY)
      let publicKey = await get<CryptoKey>(DPOP_PUBLIC_KEY)

      if (!privateKey || !publicKey) {
        const keyPair = await window.crypto.subtle.generateKey(
          {
            name: 'ECDSA',
            namedCurve: 'P-256',
          },
          false, // non-extractable private key
          ['sign', 'verify']
        )

        privateKey = keyPair.privateKey
        publicKey = keyPair.publicKey

        const jwk = await exportJWK(publicKey)
        const jkt = await calculateJwkThumbprint(jwk)

        await set(DPOP_PRIVATE_KEY, privateKey)
        await set(DPOP_PUBLIC_KEY, publicKey)
        await set(DPOP_JKT, jkt)
      }
    } finally {
      // Clear the promise so future calls check IDB again
      keyPairPromise = null
    }
  })()

  return keyPairPromise
}

/**
 * Returns the JWK Thumbprint (jkt) of the current public key.
 */
export async function getJKT(): Promise<string> {
  const jkt = await get<string>(DPOP_JKT)
  if (!jkt) {
    await ensureDPoPKeyPair()
    return (await get<string>(DPOP_JKT)) as string
  }
  return jkt
}

/**
 * Generates a DPoP Proof JWT for a specific HTTP method and URL.
 * Optionally includes the Access Token Hash (ath) if accessToken is provided.
 */
export async function getDPoPProof(
  method: string,
  url: string,
  accessToken?: string
): Promise<string> {
  await ensureDPoPKeyPair()

  const privateKey = await get<CryptoKey>(DPOP_PRIVATE_KEY)
  const publicKey = await get<CryptoKey>(DPOP_PUBLIC_KEY)

  if (!privateKey || !publicKey) {
    throw new Error('DPoP key pair not found')
  }

  const jwk = await exportJWK(publicKey)
  
  // Create a unique random jti
  const jti = crypto.randomUUID()

  // Build the payload
  const payload: { htm: string; htu: string; jti: string; ath?: string } = {
    htm: method.toUpperCase(),
    htu: url,
    jti: jti,
  }

  if (accessToken) {
    // Calculate access token hash (ath)
    const encoder = new TextEncoder()
    const data = encoder.encode(accessToken)
    const hashBuffer = await crypto.subtle.digest('SHA-256', data)
    
    // Base64URL encode the hash
    const hashArray = Array.from(new Uint8Array(hashBuffer))
    const hashString = String.fromCharCode.apply(null, hashArray)
    const ath = btoa(hashString)
      .replace(/\+/g, '-')
      .replace(/\//g, '_')
      .replace(/=+$/, '')
      
    payload.ath = ath
  }

  // Sign the JWT
  const jwt = await new SignJWT(payload)
    .setProtectedHeader({
      alg: 'ES256',
      typ: 'dpop+jwt',
      jwk: jwk, // Embed public key in header
    })
    .setIssuedAt()
    .sign(privateKey)

  return jwt
}
