module.exports = {
  apps: [{
    name: 'oneround',
    script: './oneround-server',
    args: ['--config', 'config.yaml'],
    cwd: '/opt/oneround',
    autorestart: true,
    max_restarts: 10,
    restart_delay: 3000,
    max_memory_restart: '256M',
    env: {
      NODE_ENV: 'production',
    },
  }],
};
