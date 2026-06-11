module.exports = {
  testEnvironment: 'jsdom',
  setupFilesAfterFramework: ['./src/setupTests.js'],
  collectCoverageFrom: ['src/**/*.{js,jsx}', '!src/main.jsx'],
  coverageThreshold: {
    global: {
      lines: 80,
      statements: 80,
      branches: 80,
      functions: 80,
    },
  },
};