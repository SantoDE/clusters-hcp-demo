import { defineAppSetup } from '@slidev/types'

export default defineAppSetup(({ app }) => {
  app.config.globalProperties.$base = import.meta.env.BASE_URL
})
