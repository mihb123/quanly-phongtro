import { useState } from 'react'
import axios from 'axios'

export type FormError = {
  message: string
  fieldErrors: Record<string, string>
}

export function useFormError() {
  const [formError, setFormError] = useState<FormError | null>(null)

  function handleApiError(error: unknown) {
    if (axios.isAxiosError(error)) {
      const responseData = error.response?.data
      
      // Nếu Backend trả về validation errors
      if (responseData && typeof responseData === 'object' && responseData.errors) {
        setFormError({
          message: responseData.message || 'Dữ liệu không hợp lệ',
          fieldErrors: responseData.errors as Record<string, string>,
        })
        return
      }

      setFormError({
        message: responseData?.message || error.message || 'Có lỗi xảy ra',
        fieldErrors: {},
      })
      return
    }

    setFormError({
      message: 'An unexpected error occurred',
      fieldErrors: {},
    })
  }

  function clearFormError() {
    setFormError(null)
  }

  return { formError, handleApiError, clearFormError }
}
