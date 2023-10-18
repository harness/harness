#!/usr/bin/env node
import fs from 'fs'
import path from 'path'
import sade from 'sade'
import chalk from 'chalk'
import { optimizeAndVerifySVGContent } from '../optimizer/optimizer.js'
import { getTemplate } from '../templates/template.js'
import { IssueLevel, DefaultOptions } from '../consts.js'
import { uniq } from 'lodash-es'
import { halt, highlightContent, pluralIf } from '../utils.js'

sade('svg2icon', true)
  .describe('Generate icon components from a collection of SVG files')
  .example(
    'svg2icon --source=src/icons --dest=src/components --iconset=noir --singleColor --allowedColors black,white --icon --index --lib=react'
  )
  .option('--source', 'Icon input source folder', DefaultOptions.source)
  .option('--dest', 'Component output folder', DefaultOptions.dest)
  .option('--iconset', 'Icon set name', DefaultOptions.iconset)
  .option('--singleColor', 'Enforce single color', DefaultOptions.singleColor)
  .option(
    '--allowedColors',
    'Limit to a single color, chosen from an approved list (separated by commas)',
    DefaultOptions.allowedColors
  )
  .option('--icon', 'Enforce square icon', DefaultOptions.icon)
  .option('--index', 'Generate index file', DefaultOptions.index)
  .option('--lib', 'Component target library (WebC, React, Vue, etc...)', DefaultOptions.lib)
  .action(async ({ source: _source, dest: _dest, iconset, singleColor, allowedColors, icon, index, lib }) => {
    const source = path.normalize(_source)
    const dest = path.normalize(_dest)

    if (!fs.existsSync(source)) {
      halt(`Icons source folder (${chalk.underline.blue(source)}) does not exist.\r\n`)
    }

    const sourceDir = fs.opendirSync(source)
    const iconNames = []
    const template = getTemplate(lib)
    let hasTransformError = false
    const filesCounter = {
      svg: 0,
      component: 0
    }

    // Clean up dest folder to ensure generated components
    // match source svg files
    fs.rmSync(dest, { recursive: true, force: true })
    fs.mkdirSync(dest)

    for await (const file of sourceDir) {
      try {
        const [iconName, ext = ''] = file.name.split('.')
        const inputFile = path.join(source, file.name)
        const outputFile = path.join(dest, template.componentFile(iconName))

        if (ext.toLowerCase() === 'svg') {
          filesCounter.svg++

          const inputSVG = fs.readFileSync(inputFile, 'utf8')
          const outputSVG = optimizeAndVerifySVGContent({
            filename: file.name,
            inputFile,
            svgContent: inputSVG,
            icon,
            singleColor,
            allowedColors: allowedColors.split(',').map(color => color.trim()),
            reportIssue
          })

          if (outputSVG) {
            fs.writeFileSync(
              outputFile,
              template.renderComponent({
                iconset,
                iconName,
                svg: outputSVG
              })
            )
            filesCounter.component++
          } else {
            hasTransformError = true
          }

          if (index) {
            iconNames.push(iconName)
          }
        }
      } catch (err) {
        halt(err)
      }
    }

    if (hasTransformError) {
      halt()
    }

    const indexFile = path.join(dest, template.INDEX_FILE)

    if (iconNames.length) {
      fs.writeFileSync(indexFile, template.renderIndex(iconNames))
    } else {
      fs.rmSync(indexFile, { force: true })
    }

    printReportAndExit({ filesCounter, source })
  })
  .parse(process.argv) // eslint-disable-line no-undef

const metrics = {
  errors: [],
  warnings: []
}

function reportIssue({ inputFile, svgContent, issues }) {
  const hasError = !!issues.find(issue => issue.level === IssueLevel.ERROR)
  console.log(`${hasError ? chalk.bgRed('[ERROR]') : chalk.bgYellow('[WARN]')} ${chalk.blue.underline(inputFile)}\r\n`)
  let contentOutput = svgContent

  for (const issue of issues) {
    const { ruleType, attributes, level } = issue
    const isError = level === IssueLevel.ERROR

    if (isError) {
      console.log(` - ${chalk.red(ruleType)}`)
      metrics.errors.push(inputFile)
    } else {
      console.log(` - ${chalk.yellow(ruleType)}`)
      metrics.warnings.push(inputFile)
    }

    for (const attr of attributes) {
      contentOutput = highlightContent(contentOutput, attr, isError)
    }
  }

  console.log('\r\n' + contentOutput + '\r\n')
}

function printReportAndExit({ filesCounter, source }) {
  if (!filesCounter.svg) {
    halt(`No SVG files found in ${chalk.blue.underline(source)}.\r\n`)
  }

  const { errors, warnings } = metrics
  if (warnings.length) {
    console.log(
      chalk.bold.yellow(`${warnings.length} ${pluralIf('warning', warnings.length)}`) +
        ` found in ${uniq(warnings)
          .map(file => chalk.blue.underline(file))
          .join(', ')}\r\n`
    )
  }

  if (errors.length) {
    console.log(
      `${chalk.red.bold(`❌ [FAILED] ${errors.length}`)} ${pluralIf('error', errors.length)} found in ${uniq(errors)
        .map(file => chalk.blue.underline(file))
        .join(', ')}`
    )
  }

  if (errors.length) {
    console.log()
    halt()
  } else {
    const { svg, component } = filesCounter
    console.log(
      `${chalk.green.bold('✅ [DONE]')} Successfully processed ${chalk.bold.green(svg)} SVG ${pluralIf(
        'file',
        svg
      )}, resulting in the creation of ${chalk.bold.green(component)} ${pluralIf('component', component)}.\r\n`
    )
  }
}
