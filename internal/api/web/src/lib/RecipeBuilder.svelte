<script>
  import { Shield, Wand2, X, Plus, Trash2, Send, Check } from 'lucide-svelte';
  import { t } from './i18n.js';
  import { success, error as toastError } from './toast.js';
  import { fade, fly } from 'svelte/transition';
  import { onMount } from 'svelte';

  export let show = false;
  export let feed = null;
  export let onClose = () => {};

  let bodyXPaths = [];
  let titleXPath = "";
  let stripXPaths = [];
  let currentSelection = "";
  let selectionMode = 'body'; // 'body', 'strip', 'title'
  let iframe;
  let loading = true;

  onMount(() => {
    const handleMessage = (event) => {
      if (event.data?.type === 'gorrson-select') {
        currentSelection = event.data.xpath;
        if (selectionMode === 'body') {
          if (!bodyXPaths.includes(currentSelection)) {
            bodyXPaths = [...bodyXPaths, currentSelection];
          }
        } else if (selectionMode === 'title') {
          titleXPath = currentSelection;
        } else if (selectionMode === 'strip') {
          if (!stripXPaths.includes(currentSelection)) {
            stripXPaths = [...stripXPaths, currentSelection];
          }
        }
      }
    };
    window.addEventListener('message', handleMessage);
    return () => window.removeEventListener('message', handleMessage);
  });

  $: if (iframe && !loading) {
    iframe.contentWindow.postMessage({
      type: 'gorrson-sync',
      body: bodyXPaths,
      title: titleXPath,
      strip: stripXPaths
    }, '*');
  }

  function clearSelection() {
    bodyXPaths = [];
    titleXPath = "";
    stripXPaths = [];
    currentSelection = "";
  }

  async function handleSaveLocal() {
    if (bodyXPaths.length === 0) {
      toastError("At least one body block is required");
      return;
    }
    
    try {
      const res = await fetch('/recipes/save', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}` 
        },
        body: JSON.stringify({
          feed_id: feed.id,
          title: titleXPath,
          body_list: bodyXPaths,
          strip: stripXPaths
        })
      });

      if (res.ok) {
        success("Recipe saved successfully! Now refresh news to see clean text.");
        onClose();
      } else {
        const err = await res.text();
        toastError("Failed to save: " + err);
      }
    } catch (e) {
      console.error(e);
      toastError("Network error");
    }
  }

  async function handleShare() {
     if (!feed) return;
     const targetUrl = feed.sample_url || feed.site_url || feed.feed_url;
     open("https://github.com/fivefilters/ftr-site-config/issues/new?title=New+Recipe+Request&body=" + encodeURIComponent(
       `Domain: ${new URL(targetUrl).hostname}\n\nRecipe:\ntitle: ${titleXPath}\n${bodyXPaths.map(b => 'body: ' + b).join('\n')}\n${stripXPaths.map(s => 'strip: ' + s).join('\n')}`
     ), "_blank");
  }

  $: targetUrl = feed?.sample_url || feed?.site_url || feed?.feed_url || "";
  $: { console.log("RecipeBuilder target:", targetUrl); }
</script>

{#if show && feed}
<div class="builder-overlay" transition:fade={{ duration: 250 }}>
  <div class="builder-container" transition:fly={{ y: 40, duration: 400 }}>
    <header class="builder-header">
      <div class="header-left">
        <div class="icon-wand"><Wand2 size={24} /></div>
        <div class="title-stack">
          <h2>Recipe Builder [{feed?.id?.substring(0,4) || '...'}]</h2>
          <span class="url">{targetUrl || "MISSING URL!"}</span>
        </div>
      </div>
      <button class="btn-close" on:click={onClose}><X size={24} /></button>
    </header>

    <div class="builder-main">
      <div class="iframe-view">
        <iframe 
          bind:this={iframe}
          title="Recipe Preview"
          src={targetUrl ? `/proxy/page?url=${encodeURIComponent(targetUrl)}` : ''} 
          class="preview-frame"
          on:load={() => loading = false}
        ></iframe>
        {#if loading}
          <div class="iframe-loading" out:fade>
            <div class="spinner"></div>
            <span>Loading Site Preview...</span>
          </div>
        {/if}
        <div class="visual-hint">
          Click elements in the page to select them
        </div>
      </div>

      <aside class="builder-sidebar">
        <div class="config-box">
          <div class="section-label">1. Primary Content (Body)</div>
          <p class="desc-small">Select the containers that hold the article text.</p>
          <button class="btn-add-strip {selectionMode === 'body' ? 'active' : ''}" on:click={() => selectionMode = 'body'}>
             <Plus size={16} />
             <span>Add Body Block</span>
          </button>
          
          <div class="strips-list">
            {#each bodyXPaths as body, i}
              <div class="strip-item body-item">
                <span class="xpath">{body}</span>
                <button class="btn-remove" on:click={() => bodyXPaths = bodyXPaths.filter((_, j) => i !== j)}>
                  <Trash2 size={14} />
                </button>
              </div>
            {/each}
          </div>
        </div>

        <div class="config-box">
          <div class="section-label">2. Article Title (Optional)</div>
          <p class="desc-small">Select the main headline if you want to override the RSS title.</p>
          <button type="button" class="selector-row {selectionMode === 'title' ? 'active' : ''}" on:click={() => selectionMode = 'title'}>
             <span class="xpath">{titleXPath || 'Click in page to select...'}</span>
             {#if titleXPath}<Check size={16} />{/if}
          </button>
        </div>

        <div class="config-box">
          <div class="section-label">3. Elements to Remove (Strip)</div>
          <p class="desc-small">Select ads, social buttons, or related links to hide them.</p>
          <button class="btn-add-strip {selectionMode === 'strip' ? 'active' : ''}" on:click={() => selectionMode = 'strip'}>
             <Plus size={16} />
             <span>Add Strip Rule</span>
          </button>
          
          <div class="strips-list">
            {#each stripXPaths as strip, i}
              <div class="strip-item">
                <span class="xpath">{strip}</span>
                <button class="btn-remove" on:click={() => stripXPaths = stripXPaths.filter((_, j) => i !== j)}>
                  <Trash2 size={14} />
                </button>
              </div>
            {/each}
          </div>
        </div>

        <div class="sidebar-footer">
          <button class="btn-outline" on:click={clearSelection}>Reset</button>
          <button class="btn-primary" on:click={handleSaveLocal}>Save Local</button>
          <button class="btn-secondary" on:click={handleShare}>
            <Send size={18} />
            <span>Share</span>
          </button>
        </div>
      </aside>
    </div>
  </div>
</div>
{/if}

<style>
  .builder-overlay {
    position: fixed;
    inset: 0;
    z-index: 9000;
    background: rgba(0,0,0,0.8);
    display: flex;
    align-items: center;
    justify-content: center;
    backdrop-filter: blur(10px);
    padding: 24px;
  }

  .builder-container {
    width: 100%;
    height: 100%;
    max-width: 1400px;
    background: var(--md-background);
    border-radius: 32px;
    border: 1px solid var(--md-outline);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    box-shadow: 0 40px 100px rgba(0,0,0,0.6);
  }

  .builder-header {
    padding: 20px 32px;
    border-bottom: 1px solid var(--md-outline);
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: var(--md-surface);
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 20px;
  }

  .icon-wand {
    background: var(--md-primary-container);
    color: var(--md-primary);
    width: 48px;
    height: 48px;
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .title-stack h2 {
    margin: 0;
    font: var(--m3-headline-small);
    font-weight: 700;
  }

  .url {
    font: var(--m3-label-medium);
    color: var(--md-on-surface-variant);
    opacity: 0.7;
  }

  .builder-main {
    flex: 1;
    display: flex;
    overflow: hidden;
  }

  .iframe-view {
    flex: 1;
    position: relative;
    background: #fff;
  }

  .preview-frame {
    width: 100%;
    height: 100%;
    border: none;
  }

  .visual-hint {
    position: absolute;
    bottom: 24px;
    left: 50%;
    transform: translateX(-50%);
    background: rgba(0,0,0,0.7);
    color: white;
    padding: 8px 16px;
    border-radius: var(--radius-full);
    font-size: 13px;
    pointer-events: none;
  }

  .iframe-loading {
    position: absolute;
    inset: 0;
    background: var(--md-surface);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 16px;
    z-index: 10;
  }

  .spinner {
    width: 40px;
    height: 40px;
    border: 4px solid var(--md-surface-variant);
    border-top: 4px solid var(--md-primary);
    border-radius: var(--radius-full);
    animation: spin 1s linear infinite;
  }

  @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }

  .builder-sidebar {
    width: 360px;
    background: var(--md-surface);
    border-left: 1px solid var(--md-outline);
    padding: 32px;
    display: flex;
    flex-direction: column;
    gap: 28px;
    overflow-y: auto;
  }

  .section-label {
    font-weight: 700;
    margin-bottom: 12px;
    color: var(--md-primary);
    text-transform: uppercase;
    font-size: 12px;
    letter-spacing: 0.5px;
  }

  .selector-row {
    background: var(--md-background);
    padding: 16px;
    border-radius: 12px;
    border: 2px dashed var(--md-outline);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: space-between;
    transition: all 0.2s;
  }

  .selector-row.active {
    border-color: var(--md-primary);
    background: var(--md-primary-container);
    color: var(--md-primary);
  }

  .xpath {
    font-family: monospace;
    font-size: 13px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }


  .btn-add-strip {
    width: 100%;
    padding: 12px;
    border-radius: 12px;
    border: 1px solid var(--md-outline);
    background: var(--md-surface);
    color: var(--md-on-surface);
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    cursor: pointer;
    font-weight: 600;
  }

  .btn-add-strip.active {
    border-color: var(--md-primary);
    color: var(--md-primary);
    background: var(--md-primary-container);
  }

  .strips-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: 12px;
  }

  .strip-item {
    background: var(--md-background);
    padding: 8px 12px;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 12px;
  }

  .btn-remove {
    background: none;
    border: none;
    color: #ff5252;
    cursor: pointer;
    padding: 4px;
    border-radius: 4px;
  }
  .btn-remove:hover { background: rgba(255,82,82,0.1); }

  .sidebar-footer {
    margin-top: auto;
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }

  .sidebar-footer .btn-secondary {
    grid-column: span 2;
    margin-top: 8px;
  }

  .btn-close {
    background: none;
    border: none;
    color: var(--md-on-surface);
    cursor: pointer;
    opacity: 0.6;
  }
  .btn-close:hover { opacity: 1; }

  .btn-primary {
    background: var(--md-primary);
    color: var(--md-on-primary);
    border: none;
    padding: 12px;
    border-radius: 12px;
    font-weight: 700;
    cursor: pointer;
  }

  .btn-secondary {
    background: var(--md-secondary-container);
    color: var(--md-on-secondary-container);
    border: none;
    padding: 12px;
    border-radius: 12px;
    font-weight: 700;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
  }

  .btn-outline {
    background: none;
    border: 1px solid var(--md-outline);
    color: var(--md-on-surface);
    padding: 12px;
    border-radius: 12px;
    font-weight: 700;
    cursor: pointer;
  }
</style>
