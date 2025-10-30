export const RuleType = {
  viewBoxEmpty: 'Empty SVG viewBox Detected: The viewBox attribute cannot be empty.',
  viewBoxUnbalanced: 'Unbalanced SVG viewBox: The dimensions in the viewBox attribute are not in proportion.',
  viewBoxDifferentWidthHeight: 'Inconsistent SVG Dimensions: Width and height do not match.',
  currentColorFillStroke: 'Only "currentColor" allowed for fill and stroke colors.'
}

export const IssueLevel = {
  ERROR: 'ERROR',
  WARN: 'WARN'
}

export const TargetLibrary = {
  REACT: 'react'
}

export const DefaultOptions = {
  source: 'src/icons',
  dest: 'src/components',
  iconset: '',
  singleColor: true,
  allowedColors: '',
  icon: true,
  size: '16',
  strokeWidth: '1',
  index: true,
  lib: TargetLibrary.REACT
}
