import { useState } from 'react'

export function usePageIndex(index = 0) {
  return useState(index)
}
