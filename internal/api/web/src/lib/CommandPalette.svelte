<script>
  import { onMount } from 'svelte';
  import { Zap } from 'lucide-svelte';
  export let searchQuery = "";
  export let placeholder = "Search (⌘K or /)";
  
  let show = false;
  let inputEl;

  function handleKeydown(e) {
    if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
      e.preventDefault();
      show = !show;
    }
    if (e.key === '/') {
      if (document.activeElement.tagName !== 'INPUT' && document.activeElement.tagName !== 'TEXTAREA') {
        e.preventDefault();
        show = true;
      }
    }
    if (e.key === 'Escape') {
      show = false;
    }
  }

  $: if (show && inputEl) {
    setTimeout(() => {
      if (inputEl) inputEl.focus();
    }, 50);
  }

  onMount(() => {
    window.addEventListener('keydown', handleKeydown);
    return () => window.removeEventListener('keydown', handleKeydown);
  });
</script>

{#if show}
  <div class="command-palette-overlay" on:click|self={() => show = false} role="presentation">
    <div class="command-palette glass">
      <span class="icon">
        <Zap size={18} fill="currentColor" />
      </span>
      <input 
        type="text" 
        {placeholder}
        bind:value={searchQuery}
        bind:this={inputEl}
      />
      <span class="kbd">ESC</span>
    </div>
  </div>
{/if}

<style>
  .command-palette-overlay {
    position: fixed;
    inset: 0;
    z-index: 2000;
    background: rgba(0,0,0,0.4);
    display: flex;
    justify-content: center;
    padding-top: 100px;
    backdrop-filter: blur(4px);
  }
  .command-palette {
    width: 640px;
    height: 64px;
    padding: 0 24px;
    border-radius: var(--radius-m);
    display: flex;
    align-items: center;
    gap: 16px;
    background: var(--md-surface);
    border: 1px solid var(--md-outline);
    box-shadow: 0 24px 48px rgba(0, 0, 0, 0.5);
  }

  .icon {
    color: var(--md-primary);
    display: flex;
    align-items: center;
  }

  input {
    background: transparent;
    border: none;
    color: var(--md-on-surface);
    outline: none;
    flex: 1;
    font-size: 18px;
    font-family: var(--font-main);
  }

  .kbd {
    background: var(--md-surface-variant);
    border: 1px solid var(--md-outline);
    padding: 4px 10px;
    border-radius: var(--radius-s);
    font-size: 11px;
    font-weight: 800;
    color: var(--md-on-surface-variant);
    opacity: 0.8;
  }
</style>
