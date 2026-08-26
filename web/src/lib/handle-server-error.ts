import { toast } from 'sonner'
import { ApiError } from '@/lib/api'

/** Error thrown by the generated v3 SDK (non-2xx responses). */
interface SdkHttpError extends Error {
  status: number
  title: string
}

function isSdkHttpError(error: unknown): error is SdkHttpError {
  return (
    error instanceof Error &&
    'status' in error &&
    typeof (error as { status?: unknown }).status === 'number' &&
    'title' in error
  )
}

export function handleServerError(error: unknown) {
  if (import.meta.env.DEV) {
    // eslint-disable-next-line no-console
    console.log(error)
  }

  let errMsg = 'Something went wrong!'

  if (
    error &&
    typeof error === 'object' &&
    'status' in error &&
    Number(error.status) === 204
  ) {
    errMsg = 'No content.'
  }

  if (error instanceof ApiError) {
    const title = (error.body as { title?: string } | null)?.title
    if (typeof title === 'string' && title.length > 0) {
      errMsg = title
    } else {
      errMsg = error.message
    }
  } else if (isSdkHttpError(error) && error.title) {
    errMsg = error.title
  }

  toast.error(errMsg)
}
