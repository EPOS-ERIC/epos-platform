const target = process.env.PLATFORM_API_PROXY_TARGET || 'http://localhost:32000';

module.exports = {
  '/v1': {
    target,
    secure: false,
    changeOrigin: true,
  },
};
