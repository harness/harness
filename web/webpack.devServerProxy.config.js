require('dotenv').config()

const baseUrl = process.env.BASE_URL ?? 'https://qa.harness.io/gateway'
const targetLocalHost = JSON.parse(process.env.TARGET_LOCALHOST || 'true')

const DEV = process.env.NODE_ENV === 'development'

if (DEV) {
  console.table({ baseUrl, targetLocalHost })
}

module.exports = {
  '/api': {
    target: targetLocalHost ? 'http://localhost:3000' : baseUrl
  }
}
