import { TargetLibrary } from '../consts.js'
import * as reactTemplate from './react.js'

export function getTemplate(lib = TargetLibrary.REACT) {
  switch (lib.toLowerCase()) {
    case TargetLibrary.REACT:
      return reactTemplate
  }
  console.error(`Library "${lib}" not supported`)
  process.exit(1) // eslint-disable-line no-undef
}
