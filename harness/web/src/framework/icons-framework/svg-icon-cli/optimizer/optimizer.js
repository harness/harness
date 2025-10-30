import { optimize } from 'svgo'
import { viewBoxRules } from '../rules/viewBox.js'
import { widthHeightRules } from '../rules/widthHeight.js'
import { currentColorRules } from '../rules/currentColor.js'

export const optimizeAndVerifySVGContent = ({
  filename,
  inputFile,
  svgContent,
  icon,
  singleColor,
  allowedColors = [],
  reportIssue
}) => {
  const params = { inputFile, svgContent, reportIssue }

  return optimize(svgContent, {
    path: filename,
    multipass: true,
    plugins: [
      ...(icon
        ? [viewBoxRules, widthHeightRules].map(rule => ({
            params,
            ...runOnce(rule)
          }))
        : []),
      ...(singleColor
        ? [
            {
              params: { ...params, allowedColors },
              ...runOnce(currentColorRules)
            }
          ]
        : []),
      {
        name: 'preset-default',
        params: {
          overrides: {
            removeViewBox: false
          }
        }
      },
      'removeStyleElement',
      'removeScriptElement',
      'removeXMLNS'
    ]
  }).data
}

// Execute custom SVGO plugins (rules) only once. This is because the rules should
// solely target the original SVG content, excluding any transformed versions
// that might be processed by other plugins.
const runOnce = plugin => ({
  ...plugin,
  fn: (ast, params, info) => (info.multipassCount ? {} : plugin.fn(ast, params, info))
})
