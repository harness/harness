const puppeteer = require('puppeteer')
const lighthouse = require('lighthouse')
const reportGenerator = require('lighthouse/lighthouse-core/report/report-generator')
const QA_URL = 'https://qa.harness.io/ng/'
const PROD_URL = 'https://app.harness.io/ng/'
const fs = require('fs')
const lighthouseRunTimes = 3
const acceptableChange = process.env.LIGHT_HOUSE_ACCEPTANCE_CHANGE
  ? parseInt(process.env.LIGHT_HOUSE_ACCEPTANCE_CHANGE)
  : 5
console.log('acceptableChange', acceptableChange)
const PORT = 8041
let url = PROD_URL
let isQA = false
let passWord = ''
let emailId = 'ui_perf_test_prod@mailinator.com'
async function run() {
  if (process.argv[2] === 'qa') {
    isQA = true
    url = QA_URL
    passWord = process.env.PASSWORD
  } else {
    passWord = process.env.LIGHT_HOUSE_SECRET
  }
  if (!url) {
    throw 'Please provide URL as a first argument'
  }

  const getScores = resultSupplied => {
    let json = reportGenerator.generateReport(resultSupplied.lhr, 'json')
    json = JSON.parse(json)
    let scores = {
      Performance: 0,
      Accessibility: 0,
      'Best Practices': 0,
      SEO: 0,
      'Time To Interactive': 0,
      'First ContentFul Paint': 0,
      'First Meaningful Paint': 0
    }
    scores.Performance = parseFloat(json.categories.performance.score) * 100
    scores.Accessibility = parseFloat(json.categories.accessibility.score) * 100
    scores['Best Practices'] = parseFloat(json['categories']['best-practices']['score']) * 100
    scores.SEO = parseFloat(json.categories.seo.score) * 100
    scores['Time To Interactive'] = json.audits.interactive.displayValue
    scores['First Meaningful Paint'] = json.audits['first-meaningful-paint'].displayValue
    scores['First ContentFul Paint'] = json.audits['first-contentful-paint'].displayValue
    console.log(scores)
    return scores
  }

  const runLightHouseNtimes = async (n, passedUrl) => {
    let localResults = []
    for (let i = 0; i < n; i++) {
      console.log(`Running lighthouse on ${passedUrl} for the ${i + 1} time`)
      try {
        const result = await lighthouse(passedUrl, { port: PORT, disableStorageReset: true })
        localResults.push(getScores(result))
      } catch (e) {
        console.log(e)
        process.exit(1)
      }
    }
    return localResults
  }
  const getAverageResult = (listOfResults, attributeName) => {
    let listLength = listOfResults.length
    let returnAvg = 0
    if (listLength) {
      const sum = listOfResults.reduce((tempSum, ele) => {
        tempSum = tempSum + parseFloat(ele[attributeName])
        return tempSum
      }, returnAvg)
      return sum / listLength
    }
    return returnAvg
  }
  const getFilterResults = resultsToBeFilterd => {
    return {
      Performance: getAverageResult(resultsToBeFilterd, 'Performance').toFixed(2),
      Accessibility: getAverageResult(resultsToBeFilterd, 'Accessibility').toFixed(2),
      'Best Practices': getAverageResult(resultsToBeFilterd, 'Best Practices').toFixed(2),
      SEO: getAverageResult(resultsToBeFilterd, 'SEO').toFixed(2),
      'Time To Interactive': `${getAverageResult(resultsToBeFilterd, 'Time To Interactive').toFixed(2)} s`,
      'First Meaningful Paint': `${getAverageResult(resultsToBeFilterd, 'First Meaningful Paint').toFixed(2)} s`,
      'First ContentFul Paint': `${getAverageResult(resultsToBeFilterd, 'First ContentFul Paint').toFixed(2)} s`
    }
  }
  const runLightHouseNtimesAndGetResults = async (numberOfTimes, passedUrl) => {
    const browser = await puppeteer.launch({
      headless: true,
      executablePath: '/usr/bin/google-chrome',
      args: ['--no-sandbox', `--remote-debugging-port=${PORT}`]
    })
    let page = await browser.newPage()
    await page.setDefaultNavigationTimeout(300000) // 5 minutes timeout
    await page.goto(passedUrl)
    const emailInput = await page.$('#email')
    await emailInput.type(emailId)
    const passwordInput = await page.$('#password')
    await passwordInput.type(passWord)
    await page.$eval('input[type="submit"]', form => form.click())
    await page.waitForNavigation()
    await page.waitForXPath("//span[text()='Main Dashboard']")
    let results = await runLightHouseNtimes(numberOfTimes, passedUrl)
    await browser.close()
    return getFilterResults(results)
  }
  const percentageChangeInTwoParams = (dataToBeCompared, benchMarkData, parameter) => {
    const percentageChange = parseFloat(
      ((parseFloat(dataToBeCompared) - parseFloat(benchMarkData)) / parseFloat(benchMarkData)) * 100
    ).toFixed(2)
    console.log(
      `Comparing ${parameter} Benchmark Value:${benchMarkData}, Data to be compared Value: ${dataToBeCompared} precentage change: ${percentageChange}`
    )
    return percentageChange
  }
  let finalResults = await runLightHouseNtimesAndGetResults(lighthouseRunTimes, url)

  console.log(`Scores for the ${url} \n`, finalResults)
  const finalReport = `Lighthouse ran ${lighthouseRunTimes} times on (${url}) and following are the results
  Name | Value
------------ | -------------
Performance | ${finalResults.Performance}/100
SEO | ${finalResults.SEO}/100
Accessibility | ${finalResults.Accessibility}/100
Best Practices | ${finalResults['Best Practices']}/100
First ContentFul Paint | ${finalResults['First ContentFul Paint']}
First Meaningful Paint | ${finalResults['First Meaningful Paint']}
Time To Interactive | ${finalResults['Time To Interactive']}`
  if (!isQA) {
    fs.writeFile('lighthouse.md', finalReport, function (err) {
      if (err) {
        console.log(err)
        process.exit(1)
      }
    })
  } else {
    console.log('Final Report:', finalReport)

    console.log(`Starting benchmark results collection using  ${PROD_URL}`)
    let benchMark = await runLightHouseNtimesAndGetResults(lighthouseRunTimes, PROD_URL)
    console.log(`benchmark results`, benchMark)
    let hasError = false
    let percentChange = percentageChangeInTwoParams(finalResults.Performance, benchMark.Performance, 'Performance')
    if (percentChange < -acceptableChange) {
      console.error(
        `Performance value of ${finalResults.Performance} is  ${percentChange} %  less than expected ${benchMark.Performance}`
      )
      hasError = true
    }
    percentChange = percentageChangeInTwoParams(finalResults.SEO, benchMark.SEO, 'SEO')
    if (percentChange < -acceptableChange) {
      console.error(`SEO value ${finalResults.SEO} is  ${percentChange} %  less than expected ${benchMark.SEO}`)
      hasError = true
    }
    percentChange = percentageChangeInTwoParams(finalResults.Accessibility, benchMark.Accessibility, 'Accessibility')
    if (percentChange < -acceptableChange) {
      console.error(
        `Accessibility value ${finalResults.Accessibility} is  ${percentChange} %  less than expected ${benchMark.Accessibility}`
      )
      hasError = true
    }
    percentChange = percentageChangeInTwoParams(
      finalResults['Best Practices'],
      benchMark['Best Practices'],
      'Best Practices'
    )
    if (percentChange < -acceptableChange) {
      console.error(
        `Best Practices value ${finalResults['Best Practices']} is  ${percentChange} %  less than expected ${benchMark['Best Practices']}`
      )
      hasError = true
    }
    percentChange = percentageChangeInTwoParams(
      finalResults['First ContentFul Paint'],
      benchMark['First ContentFul Paint'],
      'First ContentFul Paint'
    )
    if (percentChange > acceptableChange) {
      console.error(
        `First ContentFul Paint value ${finalResults['First ContentFul Paint']} is  ${percentChange} %  more than expected ${benchMark['First ContentFul Paint']}`
      )
      hasError = true
    }
    percentChange = percentageChangeInTwoParams(
      finalResults['First Meaningful Paint'],
      benchMark['First Meaningful Paint'],
      'First Meaningful Paint'
    )
    if (percentChange > acceptableChange) {
      console.error(
        `First Meaningful Paint value ${finalResults['First Meaningful Paint']} is ${percentChange} %  more than expected ${benchMark['First Meaningful Paint']}`
      )
      hasError = true
    }
    percentChange = percentageChangeInTwoParams(
      finalResults['Time To Interactive'],
      benchMark['Time To Interactive'],
      'Time To Interactive'
    )
    if (percentChange > acceptableChange) {
      console.error(
        `Time To Interactive value ${finalResults['Time To Interactive']} is ${percentChange} %  more than expected ${benchMark['Time To Interactive']}`
      )
      hasError = true
    }
    if (hasError) {
      console.log('Failed in benchmark comparison')
      process.exit(1)
    }
  }
}
run()
