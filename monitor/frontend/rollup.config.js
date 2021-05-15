import resolve from "@rollup/plugin-node-resolve";
import commonjs from "@rollup/plugin-commonjs";
import { babel } from "@rollup/plugin-babel";
import replace from "@rollup/plugin-replace";
import html from "@web/rollup-plugin-html";

import { terser } from "rollup-plugin-terser";

// `npm run build` -> `production` is true
// `npm run dev` -> `production` is false
const production = !process.env.ROLLUP_WATCH;

const fileName = `[${production ? "hash" : "name"}].js`;
const assetName = `[${production ? "hash" : "name"}][extname]`;

export default {
  preserveEntrySignatures: false,
  treeshake: production,
  input: "./index.html",
  output: {
    entryFileNames: fileName,
    chunkFileNames: fileName,
    assetFileNames: assetName,
    format: "es",
    dir: "public",
    plugins: [],
    sourcemap: true,
  },
  plugins: [
    replace({
      "process.env.NODE_ENV": JSON.stringify(production ? "production" : "development"),
      preventAssignment: true,
    }),
    resolve({
      extensions: ['.mjs', '.js', '.jsx', '.json'],
    }),
    commonjs(),
    production && terser(), // minify, but only in production
    babel({
      babelHelpers: "bundled",
      compact: true,
      presets: [
        [
          require.resolve("@babel/preset-env"),
          {
            targets: [
              "last 3 Chrome major versions",
              "last 3 ChromeAndroid major versions",
              "last 3 Firefox major versions",
              "last 3 Edge major versions",
              "last 3 Safari major versions",
              "last 3 iOS major versions",
            ],
            useBuiltIns: false,
            shippedProposals: true,
            modules: false,
            bugfixes: true,
          },
        ],
        "@babel/preset-react"
      ],
      plugins: [
        require.resolve("@babel/plugin-syntax-dynamic-import"),
        require.resolve("@babel/plugin-syntax-import-meta"),
      ]
    }),
    html({
      publicPath: "/frontend/public/",
      extractAssets: true,
    }),
  ]
};
