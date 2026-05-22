import typescriptEslint from "@typescript-eslint/eslint-plugin";
import tsParser from "@typescript-eslint/parser";

export default [
  {
    files: ["**/*.ts", "**/*.tsx", "**/*.js", "**/*.jsx"],
    languageOptions: {
      parser: tsParser,
    },
    plugins: {
      "@typescript-eslint": typescriptEslint,
    },
    rules: {
      "complexity": ["error", 8],
      "max-params": ["error", 4],
      "max-lines-per-function": ["error", { "max": 50, "skipBlankLines": true, "skipComments": true }],
      "max-lines": ["error", { "max": 300, "skipBlankLines": true, "skipComments": true }],
      "@typescript-eslint/no-explicit-any": "warn"
    }
  }
];
