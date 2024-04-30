import { defineConfig } from 'cypress'

export default defineConfig({
  e2e: {
    baseUrl: 'http://localhost:3020',
    specPattern: 'integration/**/*.spec.{ts,tsx}',
    supportFile: 'support/e2e.ts',
    fixturesFolder: 'fixtures',
    videoUploadOnPasses: false
  },
  projectId: 'mcssf4',
  viewportWidth: 1500,
  viewportHeight: 1000,
  retries: {
    runMode: 2,
    openMode: 0
  },
  fixturesFolder: 'fixtures'
})
