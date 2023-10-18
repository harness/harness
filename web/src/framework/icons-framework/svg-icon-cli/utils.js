import chalk from 'chalk'

export const highlightContent = (str, attr = '', isError) =>
  str.replaceAll(attr, chalk[isError ? 'red' : 'yellow'].bold.underline(attr)).trim()

export const halt = message => {
  if (message) {
    console.log(`${chalk.bgRed('[ERROR]')} ${message}`)
  }
  process.exit(1) // eslint-disable-line no-undef
}

export const pluralIf = (word, count) => `${word}${count > 1 ? 's' : ''}`
