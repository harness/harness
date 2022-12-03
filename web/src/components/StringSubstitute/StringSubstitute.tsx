import React, { Fragment } from 'react'

type SubstituteVars = Record<string, any> // eslint-disable-line @typescript-eslint/no-explicit-any

function translateExpression(str: string, key: string, vars: SubstituteVars) {
  // Replace simple i18n expression {key|match1:value1,match2:value2}
  // Sample: '{user} wants to merge {number} {number|1:commit,commits} into {target} from {source}'
  // Find `{number|`
  const startIndex = str.indexOf(`{${key}|`)
  const MATCH_ELSE_KEY = '___'

  if (startIndex !== -1) {
    // Find closing `}`
    const endIndex = str.indexOf('}', startIndex)

    if (endIndex !== -1) {
      // Get whole expression of `{number|1:commit,commits}`
      const expression = str.substring(startIndex, endIndex + 1)

      // Build value mapping from expression
      const mapping = expression
        .split('|')[1] // Get `1:commit,commits}`
        .slice(0, -1) // Remove last closing `}`
        .split(',') // ['1:commit', 'commits']
        .reduce((map, token) => {
          // Convert to a map { 1: commit, [MATCH_ELSE_KEY]: commits }
          const [k, v] = token.split(':')
          map[v ? k : MATCH_ELSE_KEY] = v || k
          return map
        }, {} as Record<string, string>)

      const matchedValue = mapping[vars[key]] || mapping[MATCH_ELSE_KEY]

      if (matchedValue) {
        return str.replace(expression, matchedValue)
      }
    }
  }

  return str
}

export const StringSubstitute: React.FC<{
  str: string
  vars: SubstituteVars
}> = ({ str, vars }) => {
  const re = Object.keys(vars)
    .map(key => {
      str = translateExpression(str, key, vars)
      return `{${key}}`
    })
    .join('|')

  return (
    <>
      {str
        .split(new RegExp('(' + re + ')', 'gi'))
        .filter(token => !!(token || '').trim())
        .map((token, index) => (
          <Fragment key={index}>
            {token.startsWith('{') && token.endsWith('}') ? vars[token.substring(1, token.length - 1)] || token : token}
          </Fragment>
        ))}
    </>
  )
}
