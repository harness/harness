import { IssueLevel, RuleType } from '../consts.js'

const viewBoxElems = ['svg' /*, 'pattern', 'symbol'*/]

export const viewBoxRules = {
  name: 'viewBoxRules',
  description: 'Icon rules for SVG viewBox',
  fn: (_, params) => {
    const { inputFile, svgContent, reportIssue } = params
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
        enter: (node, parentNode) => {
          if (viewBoxElems.includes(node.name)) {
            if (node.name === 'svg' && parentNode.type === 'root') {
              if (!node.attributes.viewBox) {
                issues.push({
                  ruleType: RuleType.viewBoxEmpty,
                  attributes: ['<svg'],
                  level: IssueLevel.ERROR
                })
              }

              const nums = node.attributes.viewBox.split(/[ ,]+/g)

              if (nums[0] !== nums[1] || nums[2] !== nums[3]) {
                issues.push({
                  ruleType: RuleType.viewBoxUnbalanced,
                  attributes: [`viewBox="${node.attributes.viewBox}"`],
                  level: IssueLevel.ERROR
                })
              }
            }
          }
        }
      }
    }
  }
}
