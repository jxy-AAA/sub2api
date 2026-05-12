#!/usr/bin/env node

const userAgent = process.env.npm_config_user_agent || '';
const isPnpm = userAgent.startsWith('pnpm/');

if (!isPnpm) {
  console.error('This project uses pnpm. Run `pnpm install` from frontend/ instead of npm/yarn.');
  process.exit(1);
}
