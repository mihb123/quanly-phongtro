export const encryptRSA = async (text: string, spkiBase64: string) => {
  const binaryString = window.atob(spkiBase64)
  const bytes = new Uint8Array(binaryString.length)
  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i)
  }

  const cryptoKey = await window.crypto.subtle.importKey(
    'spki',
    bytes,
    {
      name: 'RSA-OAEP',
      hash: 'SHA-256',
    },
    true,
    ['encrypt'],
  )

  const encryptedBuffer = await window.crypto.subtle.encrypt(
    { name: 'RSA-OAEP' },
    cryptoKey,
    new TextEncoder().encode(text),
  )

  return window.btoa(String.fromCharCode(...new Uint8Array(encryptedBuffer)))
}
