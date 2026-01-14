import { RuleType, IssueLevel } from '../consts.js'

const ALLOWED_COLORS = ['none', 'transparent']

export const currentColorRules = {
  name: 'currentColorRules',
  description: 'Mandate the use of "currentColor" exclusively for fill and stroke properties',
  fn: (_, params) => {
    const { allowedColors = [], inputFile, svgContent, reportIssue } = params
    const issues = []

    return {
      root: {
        exit: () => {
          if (issues.length) {
            reportIssue({
              inputFile,
              svgContent,
              issues
            })
          }
        }
      },
      element: {
        enter: ({ type, attributes }) => {
          if (type === 'element') {
            const { fill, stroke } = attributes || {}

            if (fill || stroke) {
              if (fill && fill !== 'currentColor' && !ALLOWED_COLORS.includes(fill)) {
                issues.push({
                  ruleType: RuleType.currentColorFillStroke,
                  attributes: [`fill="${fill}"`],
                  level: allowedColors.includes(fill) ? IssueLevel.WARN : IssueLevel.ERROR
                })
              }

              if (stroke && stroke !== 'currentColor' && !ALLOWED_COLORS.includes(stroke)) {
                issues.push({
                  ruleType: RuleType.currentColorFillStroke,
                  attributes: [`stroke="${stroke}"`],
                  level: allowedColors.includes(stroke) ? IssueLevel.WARN : IssueLevel.ERROR
                })
              }
            }
          }
        }
      }
    }
  }
}
