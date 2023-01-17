import { useState } from 'react'

export function usePageIndex(index = 1) {
  return useState(index)
}
