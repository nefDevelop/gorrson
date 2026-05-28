<script>
  import { login } from "./api.js";
  import { t } from "./i18n.js";
  import { User, Lock, ChevronRight, Loader2, Sparkles } from "lucide-svelte";
  import { fly, fade } from "svelte/transition";
  import { backOut } from "svelte/easing";

  /** @type {() => void} */
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
      error = "Invalid username or password. Please try again.";
    } finally {
      loading = false;
    }
  }
</script>

<div class="login-view">
  <main
    class="login-container"
    in:fly={{ y: 20, duration: 800, easing: backOut }}
  >
    <div class="login-card">
      <header class="card-header">
        <div class="logo-wrapper">
          <h1 class="logo-text">GO<span class="rss-logo-part">RSS</span>ON</h1>
        </div>
        <p class="subtitle">{$t('login_subtitle')}</p>
      </header>

      {#if error}
        <div class="error-badge" transition:fade={{ duration: 200 }}>
          {error}
        </div>
      {/if}

      <form class="login-form" on:submit|preventDefault={handleSubmit}>
        <div class="input-field">
          <div class="input-icon">
            <User size={18} />
          </div>
          <input
            type="text"
            id="user"
            bind:value={username}
            placeholder="Username"
            required
            autocomplete="username"
          />
        </div>

        <div class="input-field">
          <div class="input-icon">
            <Lock size={18} />
          </div>
          <input
            type="password"
            id="pass"
            bind:value={password}
            placeholder="Password"
            required
            autocomplete="current-password"
          />
        </div>

        <button type="submit" class="submit-btn" disabled={loading}>
          {#if loading}
            <div class="spinner">
              <Loader2 size={20} class="animate-spin" />
            </div>
            <span>Verifying...</span>
          {:else}
            <span>Sign In</span>
            <ChevronRight size={20} />
          {/if}
        </button>
      </form>

      <footer class="card-footer">
        <span class="version">Build 2026.04 • Enterprise Ready</span>
      </footer>
    </div>
  </main>
</div>

<style>
  .login-view {
    height: 100vh;
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--md-background);
    position: relative;
    overflow: hidden;
  }

  /* Animated background blobs */
  .login-container {
    position: relative;
    z-index: 10;
    width: 100%;
    max-width: 420px;
    padding: 20px;
  }

  .login-card {
    background: var(--md-surface);
    border: 1px solid var(--md-outline);
    border-radius: 40px;
    padding: 64px 48px;
    box-shadow: 0 16px 40px var(--shadow-color, rgba(0,0,0,0.1));
    display: flex;
    flex-direction: column;
    gap: 32px;
  }

  .logo-wrapper {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
  }

  .logo-text {
    font: var(--m3-display-large);
    font-size: 55px;
    font-weight: 900;
    letter-spacing: -3px;
    margin: 0;
    background: linear-gradient(135deg, var(--md-primary), #3b82f6);
    background-clip: text;
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }

  .subtitle {
    font: var(--m3-label-small);
    text-transform: uppercase;
    letter-spacing: 1.5px;
    font-weight: 800;
    color: var(--md-on-surface-variant);
    margin: 12px 0 0 0;
    text-align: center;
  }
  .rss-logo-part {
    font-size: 1.12em;
    font-weight: 950;
    margin: 0 -2px;
    display: inline-block;
    filter: drop-shadow(0 0 8px rgba(165, 139, 255, 0.3));
  }

  .login-form {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .input-field {
    position: relative;
    display: flex;
    align-items: center;
  }

  .input-icon {
    position: absolute;
    left: 20px;
    color: var(--md-primary);
    opacity: 0.7;
    pointer-events: none;
  }

  input {
    width: 100%;
    background: var(--md-surface-variant);
    border: 1px solid transparent;
    border-radius: var(--radius-full);
    padding: 16px 24px 16px 52px;
    color: var(--md-on-surface);
    font: var(--m3-body-large);
    outline: none;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  }

  input:focus {
    background: var(--md-surface);
    border-color: var(--md-primary);
    box-shadow: 0 0 0 4px var(--md-primary-container);
  }

  .submit-btn {
    margin-top: 16px;
    height: 56px;
    background: var(--md-primary);
    color: var(--md-on-primary);
    border: none;
    border-radius: var(--radius-full);
    font: var(--m3-label-large);
    font-size: 16px;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 1px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    box-shadow: 0 4px 12px rgba(0,0,0,0.1);
  }

  .submit-btn:hover:not(:disabled) {
    background: var(--md-primary-container);
    color: var(--md-primary);
    transform: translateY(-2px);
    box-shadow: 0 8px 24px var(--md-primary-container);
  }

  .submit-btn:active:not(:disabled) {
    transform: translateY(0);
  }

  .submit-btn:disabled {
    opacity: 0.7;
    cursor: wait;
  }

  .error-badge {
    background: rgba(255, 82, 82, 0.1);
    color: #ff8a8a;
    padding: 12px 16px;
    border-radius: 16px;
    border: 1px solid rgba(255, 82, 82, 0.2);
    font: var(--m3-body-small);
    text-align: center;
  }

  .card-footer {
    display: flex;
    justify-content: center;
    opacity: 0.4;
  }

  .version {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 1px;
    font-weight: 600;
  }

  :global(.animate-spin) {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }

  @media (max-width: 480px) {
    .login-card {
      padding: 40px 24px;
      border-radius: 32px;
    }
  }
</style>
