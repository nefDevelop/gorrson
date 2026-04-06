<script>
  import { login } from './api.js';
  import { t } from './i18n.js';
  
  export let onLoginSuccess;
  
  let username = "";
  let password = "";
  let error = "";
  let loading = false;

  async function handleSubmit() {
    try {
      loading = true;
      error = "";
      await login(username, password);
      onLoginSuccess();
    } catch (e) {
      error = "Invalid username or password";
    } finally {
      loading = false;
    }
  }
</script>

<div class="login-container">
  <form class="login-card" on:submit|preventDefault={handleSubmit}>
    <h1 class="logo">GORRSON</h1>
    <p class="subtitle">RSS High-Performance Engine</p>
    
    {#if error}
      <div class="error-badge">{error}</div>
    {/if}

    <div class="input-group">
      <label for="user">Username</label>
      <input type="text" id="user" bind:value={username} required />
    </div>

    <div class="input-group">
      <label for="pass">Password</label>
      <input type="password" id="pass" bind:value={password} required />
    </div>

    <button type="submit" disabled={loading}>
      {loading ? "Authenticating..." : "Login"}
    </button>
  </form>
</div>

<style>
  .login-container {
    height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--md-background);
  }
  .login-card {
    background: var(--md-surface);
    padding: 40px;
    border-radius: var(--radius-xl);
    border: 1px solid var(--md-outline);
    width: 100%;
    max-width: 400px;
    text-align: center;
  }
  .logo { font: var(--m3-headline-large); color: var(--md-primary); margin-bottom: 8px; }
  .subtitle { font: var(--m3-body-small); opacity: 0.6; margin-bottom: 32px; }
  
  .input-group { text-align: left; margin-bottom: 20px; }
  .input-group label { display: block; font: var(--m3-label-medium); margin-bottom: 8px; opacity: 0.8; }
  .input-group input {
    width: 100%;
    padding: 12px;
    border-radius: var(--radius-s);
    background: var(--md-surface-variant);
    border: 1px solid var(--md-outline);
    color: var(--md-on-surface);
    font: var(--m3-body-medium);
  }
  
  button {
    width: 100%;
    padding: 14px;
    background: var(--md-primary);
    color: var(--md-on-primary);
    border: none;
    border-radius: var(--radius-m);
    font: var(--m3-label-large);
    font-weight: 700;
    cursor: pointer;
    margin-top: 10px;
  }
  button:disabled { opacity: 0.5; }
  .error-badge { background: #ff525222; color: #ff5252; padding: 10px; border-radius: var(--radius-s); margin-bottom: 20px; font: var(--m3-label-small); }
</style>
