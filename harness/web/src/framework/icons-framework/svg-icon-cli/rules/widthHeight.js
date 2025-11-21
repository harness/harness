import { IssueLevel, RuleType } from '../consts.js'

export const widthHeightRules = {
  name: 'widthHeightRules',
  description: 'Validation Rule: Confirm matching SVG width and height attributes to accommodate square icon style.',
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
          if (node.name === 'svg' && parentNode.type === 'root') {
            if (node.attributes.width !== node.attributes.height) {
              issues.push({
                ruleType: RuleType.viewBoxDifferentWidthHeight,
                attributes: [`width="${node.attributes.width}"`, `height="${node.attributes.height}"`],
                level: IssueLevel.ERROR
              })
            }
          }
        }
      }
    }
  }
}
