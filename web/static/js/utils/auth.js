/**
 * 认证工具模块
 * 统一处理登录状态检查和跳转
 */
(function() {
  window.AuthUtil = {
    /**
     * 检查是否已登录，未登录则跳转到登录页
     * @returns {string|null} 返回 Cookie 或 null
     */
    requireAuth() {
      const savedCookie = window.Storage.getJSON('qfnu_auth_cookie', '');
      if (!savedCookie) {
        // 未登录，跳转到登录页
        const currentPath = window.location.pathname;
        const redirect = currentPath !== '/login' ? `?redirect=${encodeURIComponent(currentPath)}` : '';
        window.location.href = '/login' + redirect;
        return null;
      }
      return savedCookie;
    },

    /**
     * 获取当前保存的 Cookie
     * @returns {string} Cookie 字符串
     */
    getCookie() {
      return window.Storage.getJSON('qfnu_auth_cookie', '');
    },

    /**
     * 设置 Cookie
     * @param {string} cookie - Cookie 字符串
     */
    setCookie(cookie) {
      window.Storage.setJSON('qfnu_auth_cookie', cookie);
      window.authCookie = cookie;
    },

    /**
     * 清除 Cookie
     */
    clearCookie() {
      window.Storage.remove('qfnu_auth_cookie');
      window.Storage.remove('zhjw_logged_in');
      window.Storage.remove('zhjw_username');
      window.authCookie = '';
    },

    /**
     * 检查登录状态并初始化全局 authCookie
     * @returns {boolean} 是否已登录
     */
    checkAndInit() {
      const cookie = this.getCookie();
      if (cookie) {
        window.authCookie = cookie;
        return true;
      }
      return false;
    },

    /**
     * 验证 Cookie 是否有效
     * @param {string} cookie - Cookie 字符串
     * @returns {Promise<boolean>} 是否有效
     */
    async verifyCookie(cookie) {
      try {
        const response = await axios.get('/api/v1/zhjw/grade', {
          headers: { 'Authorization': cookie }
        });
        return response.data && response.data.code === 200;
      } catch (e) {
        return false;
      }
    }
  };
})();
