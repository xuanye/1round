// Minimal Node globals for the test program. @types/node is deliberately NOT
// included in tsconfig.test.json: its ambient setTimeout returns NodeJS.Timeout,
// which conflicts with the WeChat typings' setTimeout(): number when both are
// loaded in the same program (src files must keep compiling against the
// WeChat declarations). Tests therefore use untyped `require()` for Node
// builtins instead of typed `import 'node:*'` statements.
declare function require(id: string): any;
declare namespace require {
  const cache: Record<string, unknown>;
  function resolve(id: string): string;
}
declare const __dirname: string;
declare const process: {
  execPath: string;
  env: Record<string, string | undefined>;
};
