import { defineConfig } from '@openapi-codegen/cli';
import { generateSchemaTypes } from '@openapi-codegen/typescript';

const OPENAPI_URL = 'http://localhost:32000/openapi.yaml';


export default defineConfig({
  backoffice: {
    from: {
      source: 'url',
      url: OPENAPI_URL,
    },
    outputDir: 'generated',
    to: async (context) => {
      if (!context.openAPIDocument.components) {
        throw new Error(
          `OpenAPI loaded from ${OPENAPI_URL} has no 'components'.`
        );
      }

      await generateSchemaTypes(context, {
        filenamePrefix: 'backoffice',
      });
    },
  },
});
