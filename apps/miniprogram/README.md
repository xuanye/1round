# OneRound Mini Program

## TypeScript

Install dependencies:

```bash
npm install
```

Compile once:

```bash
npm run build
```

Watch during development:

```bash
npm run watch
```

Open this directory in WeChat DevTools:

```text
apps/miniprogram
```

The `src/` directory contains source files. Build output is written to `dist/`, which is the Mini Program runtime root and is ignored by Git.

WeChat DevTools should open this folder directly. The project config points `miniprogramRoot` at `dist/`, and the npm scripts provide deterministic local checks.
