import { defineConfig } from '@openapi-codegen/cli';
import { generateSchemaTypes } from '@openapi-codegen/typescript';

const OPENAPI_PATH = '../../services/gateway/docs/openapi.yaml';


export default defineConfig({
  backoffice: {
    from: {
      source: 'file',
      relativePath: OPENAPI_PATH,
    },
    outputDir: 'generated',
    to: async (context) => {
      if (!context.openAPIDocument.components) {
        throw new Error(
          `OpenAPI loaded from ${OPENAPI_PATH} has no 'components'.`
        );
      }

      await generateSchemaTypes(context, {
        filenamePrefix: 'backoffice',
      });
    },
  },
});
