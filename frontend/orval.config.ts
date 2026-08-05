import { defineConfig } from 'orval'

// Источник контракта: backend/docs/openapi.yaml (копия medFlowObsidian/enpoints.yaml).
// При изменении API: правим контракт первым, затем `npm run generate:api`.
export default defineConfig({
  medFlowAPI: {
    input: '../backend/docs/openapi.yaml',
    output: {
      target: 'src/api/generated/medFlowAPI.ts',
      schemas: 'src/api/generated',
      mode: 'split',
      client: 'axios-functions',
      mock: {
        generators: [{ type: 'msw' }],
        delay: 200,
      },
      override: {
        mutator: {
          path: 'src/api/axios-instance.ts',
          name: 'customInstance',
        },
      },
    },
  },
})
