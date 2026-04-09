<script>
  import { fade, fly } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { Link, Star, ExternalLink, ArrowLeft, Share2, Send, Copy, MoreHorizontal, Maximize2, Minimize2 } from 'lucide-svelte';
  import { t } from './i18n.js';
  import { toggleFavorite, fetchFullContent } from './api.js';
  import { success as toastSuccess } from './toast.js';
  import { onMount } from 'svelte';
  
  /** @type {any} */
  export let article;
  export let onBack = () => {};
  export let isFullscreen = false;
  export let onToggleFullscreen = () => {};

  let exportServices = [];
  function loadServices() {
    exportServices = JSON.parse(localStorage.getItem('gorrson-export-services') || '[]');
  }
  onMount(() => {
    loadServices();
    const handleOutsideClick = (e) => {
      if (showExportMenu && !e.target.closest('.export-popover-container')) {
        showExportMenu = false;
      }
    };
    window.addEventListener('click', handleOutsideClick);
    return () => window.removeEventListener('click', handleOutsideClick);
  });

  // Re-load services if localStorage changes (e.g. from Settings tab)
  if (typeof window !== 'undefined') {
    window.addEventListener('storage', loadServices);
  }

  let showExportMenu = false;

  let progress = 0;
  let container;

  function handleScroll() {
    if (!container) return;
    const { scrollTop, scrollHeight, clientHeight } = container;
    const total = scrollHeight - clientHeight;
    progress = total > 0 ? (scrollTop / total) * 100 : 0;
  }

  let lastHandledId = null;

  // Reset and auto-load full content when article changes
  $: if (article && article.id !== lastHandledId) {
    lastHandledId = article.id;
    progress = 0;
    if (container) container.scrollTop = 0;
    if (!article.is_full_content) {
      loadFullContent();
    }
  }

  async function shareArticle() {
    if (navigator.share) {
      try {
        await navigator.share({
          title: article.title,
          url: article.url
        });
      } catch (e) {
        // Fallback if shared cancelled or fails
        await copyToClipboard();
      }
    } else {
      await copyToClipboard();
    }
  }

  async function copyToClipboard() {
    await navigator.clipboard.writeText(article.url);
    toastSuccess($t('link_copied'));
  }

  function sendToService(svc) {
    const url = svc.template
      .replace('{title}', encodeURIComponent(article.title))
      .replace('{url}', encodeURIComponent(article.url));
    window.open(url, '_blank');
    showExportMenu = false;
  }

  async function handleFavorite() {
    if (!article) return;
    await toggleFavorite(article.id, !article.is_favorite);
    article = { ...article, is_favorite: !article.is_favorite };
  }

  let loadingFull = false;
  async function loadFullContent() {
    if (!article || loadingFull) return;
    loadingFull = true;
    try {
      const text = await fetchFullContent(article.id);
      article = { ...article, content: text, is_full_content: true };
    } catch (e) {
      console.error(e);
      if (e.message === 'Unauthorized') {
        // We might want to notify the parent app or show a clear error
        article = { ...article, content: `<div class="error-inline">${$t('unauthorized_content') || 'Unauthorized to load full content. Please check your session.'}</div>`, is_full_content: false };
      }
    } finally {
      loadingFull = false;
    }
  }

  function handleImageError(e) {
    const parent = e.currentTarget.parentElement;
    if (parent) parent.style.display = 'none';
  }

  $: cleanedContent = (() => {
    if (!article) return '';
    let html = (article.content || article.summary || '');
    
    // 1. Hide lazy-loading placeholders (BBC, etc.)
    html = html.replace(/<img[^>]+src="[^"]*(placeholder|grey|static\.files\.bbci)[^"]*"[^>]*>/gi, '');
    
    // 2. Hide hero image in body if it's a version of the same image
    if (article.image_url) {
      // Extract unique ID from the URL (e.g., from PC Gamer's futurecdn URLs or similar)
      const parts = article.image_url.split('/');
      const lastPart = parts[parts.length - 1];
      const mainId = lastPart.split('-')[0].split('?')[0].split('.')[0];
      
      if (mainId && mainId.length > 8) {
        const fuzzyHeroRegex = new RegExp(`<img[^>]+src=["'][^"']*${mainId}[^"']*["'][^>]*>`, 'gi');
        html = html.replace(fuzzyHeroRegex, '');
      }
    }
    
    return html;
  })();
</script>

<article 
  class="post-viewer scroll-area" 
  bind:this={container} 
  on:scroll={handleScroll}
  transition:fly={{ x: 100, duration: 400, easing: cubicOut }}
>
  <!-- Header removed per user request -->


  {#if article}
    {#key article.id}
      <div 
        in:fly={{ x: 100, duration: 600, delay: 100, easing: cubicOut }} 
        out:fade={{ duration: 200 }} 
        class="content-wrapper"
      >
        {#if article.image_url}
          <div class="image-container">
            <img src={article.image_url} alt={article.title} on:error={handleImageError} />
          </div>
        {/if}
        
        <div class="meta">
          <div class="meta-top">
            <div class="info-left">
              <button class="action-btn mobile-only back-btn" on:click={onBack} title="Back to list">
                <ArrowLeft size={20} />
              </button>
              <span class="category">{article.category || 'General'}</span>
              <span class="info-sep">•</span>
              <span class="info">{article.source}</span>
            </div>
            
            <div class="meta-actions">
              <button class="action-btn desktop-only" on:click={onToggleFullscreen} title={isFullscreen ? 'Exit reader mode' : 'Expand to reader mode'}>
                {#if isFullscreen}
                  <Minimize2 size={18} />
                {:else}
                  <Maximize2 size={18} />
                {/if}
              </button>
  
              <button class="action-btn" on:click={shareArticle} title={$t('share')}>
                <Share2 size={18} />
              </button>
  
              {#if exportServices.length > 0}
                <div class="export-popover-container">
                  <button class="action-btn" on:click={() => showExportMenu = !showExportMenu} title={$t('send_to')}>
                    <Send size={18} />
                  </button>
                  {#if showExportMenu}
                    <div class="export-menu" in:fly={{ y: 10, duration: 200 }} out:fade={{ duration: 150 }}>
                      {#each exportServices as svc}
                        <button class="menu-item" on:click={() => sendToService(svc)}>
                          <span class="svc-name">{svc.name}</span>
                        </button>
                      {/each}
                    </div>
                  {/if}
                </div>
              {/if}
  
              <button class="action-btn {article.is_favorite ? 'active' : ''}" on:click={handleFavorite} title={$t('favorites')}>
                <Star size={18} fill={article.is_favorite ? 'currentColor' : 'none'} />
              </button>
              <a class="action-btn" href={article.url} target="_blank" rel="noopener" title="Open original">
                <ExternalLink size={18} />
              </a>
            </div>
          </div>
          
          <div class="meta-bottom">
            <span class="info time">{article.time}</span>
          </div>
        </div>

        
        <h1 class="title">{article.title}</h1>
        
        {#if loadingFull}
          <div class="loader-centered">
            <div class="mini-spinner"></div>
          </div>
        {/if}

        <div class="body-text prose">
          {@html cleanedContent || '<p>No content available.</p>'}
        </div>
      </div>
    {/key}
  {:else}
    <div class="empty-state">
      <p>{$t('select_article')}</p>
    </div>
  {/if}
</article>

<style>
  .post-viewer {
    flex: 1;
    display: flex;
    flex-direction: column;
    height: 100vh;
    background: var(--md-background);
    position: relative;
    overflow-y: auto;
    overflow-x: hidden;
  }


  
  .loader-centered {
    display: flex;
    justify-content: center;
    align-items: center;
    padding: 24px;
    width: 100%;
    margin-bottom: 32px;
  }

  .mini-spinner {
    width: 28px;
    height: 28px;
    border: 3px solid var(--md-primary);
    border-right-color: transparent;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  .mobile-only {
    display: none !important;
  }

  @media (min-width: 1001px) {
    .desktop-only {
      display: flex !important;
    }
  }
  @media (max-width: 1000px) {
    .desktop-only {
      display: none !important;
    }
    .mobile-only {
      display: flex !important;
    }
    .content-wrapper {
      padding: 24px 20px;
    }
    .title {
      font-size: 24px;
    }
  }

  .action-btn {
    background: transparent;
    border: none;
    color: var(--md-on-surface-variant);
    cursor: pointer;
    width: 40px;
    height: 40px;
    border-radius: var(--radius-full);
    transition: all 0.2s;
    display: flex;
    align-items: center;
    justify-content: center;
    text-decoration: none;
    flex-shrink: 0;
  }
  .action-btn:hover { background: var(--md-surface-variant); color: var(--md-on-surface); }
  .action-btn.active { color: var(--md-primary); }

  .export-popover-container {
    position: relative;
    display: flex;
  }

  .export-menu {
    position: absolute;
    top: 100%;
    right: 0;
    margin-top: 12px;
    background: var(--md-surface);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-xl);
    box-shadow: 0 4px 20px rgba(0,0,0,0.15), 0 8px 40px rgba(0,0,0,0.1);
    z-index: 1000;
    min-width: 200px;
    padding: 8px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    pointer-events: all;
    overflow: hidden;
  }

  .menu-item {
    background: transparent;
    border: none;
    padding: 10px 16px;
    border-radius: var(--radius-full);
    color: var(--md-on-surface);
    font: var(--m3-label-large);
    text-align: left;
    cursor: pointer;
    transition: background 0.2s;
    width: 100%;
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .menu-item:hover {
    background: var(--md-surface-variant);
  }

  .svc-name {
    font-weight: 600;
  }



  
  @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }


  .content-wrapper {
    padding: 40px 48px;
    max-width: 760px;
    margin: 0 auto;
    width: 100%;
  }

  .image-container {
    width: 100%;
    height: 360px;
    border-radius: var(--radius-xl);
    overflow: hidden;
    margin-bottom: 32px;
    border: 1px solid var(--md-outline);
    background: var(--md-surface-variant);
  }
  img { width: 100%; height: 100%; object-fit: cover; }

  .meta {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 24px;
    width: 100%;
  }

  .meta-top {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 16px;
    flex-wrap: wrap;
  }

  .info-left {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .info-sep {
    opacity: 0.3;
    font-size: 10px;
  }

  .meta-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .meta-actions .action-btn {
    opacity: 0.6;
    transition: opacity 0.2s;
  }

  .meta-actions .action-btn:hover {
    opacity: 1;
  }

  .meta-bottom {
    display: flex;
    justify-content: flex-end;
  }
  .category {
    font: var(--m3-label-large);
    color: var(--md-primary);
    background: var(--md-primary-container);
    padding: 4px 14px;
    border-radius: var(--radius-full);
    text-transform: uppercase;
    letter-spacing: 0.8px;
    font-weight: 600;
  }
  .info {
    font: var(--m3-label-medium);
    color: var(--md-on-surface-variant);
    opacity: 0.8;
  }

  .title {
    font-family: var(--font-display);
    font-size: 36px;
    font-weight: 900;
    line-height: 1.2;
    color: var(--md-on-surface);
    margin-bottom: 24px;
    letter-spacing: -0.5px;
  }

  /* Prose: style the raw HTML from RSS feeds */
  .body-text {
    font-family: var(--font-main);
    font-size: 17px;
    line-height: 1.8;
    color: var(--md-on-surface);
  }
  .body-text :global(p) {
    margin-bottom: 24px;
    opacity: 0.95;
  }
  .body-text :global(h1),
  .body-text :global(h2),
  .body-text :global(h3) {
    font-family: var(--font-display);
    font-weight: 800;
    color: var(--md-on-surface);
    margin: 40px 0 16px 0;
    line-height: 1.3;
  }
  .body-text :global(h2) { font-size: 22px; }
  .body-text :global(h3) { font-size: 18px; }
  .body-text :global(a) {
    color: var(--md-primary);
    text-decoration: underline;
    text-underline-offset: 3px;
  }
  .body-text :global(img) {
    max-width: 100%;
    border-radius: var(--radius-xl);
    margin: 24px 0;
    height: auto;
    border: 1px solid var(--md-outline);
  }
  .body-text :global(blockquote) {
    border-left: 3px solid var(--md-primary);
    padding-left: 20px;
    margin: 20px 0;
    color: var(--md-on-surface-variant);
    font-style: italic;
  }
  .body-text :global(pre),
  .body-text :global(code) {
    font-family: var(--font-mono);
    background: var(--md-surface-variant);
    border-radius: var(--radius-s);
    font-size: 13px;
  }
  .body-text :global(pre) { padding: 16px; overflow-x: auto; }
  .body-text :global(code) { padding: 2px 6px; }
  .body-text :global(ul),
  .body-text :global(ol) {
    padding-left: 24px;
    margin-bottom: 16px;
    line-height: 1.9;
    color: var(--md-on-surface);
    opacity: 0.9;
  }

  .body-text :global(hr) {
    border: none;
    border-top: 1px solid var(--md-outline);
    margin: 32px 0;
  }

  .empty-state {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--md-on-surface-variant);
    font: var(--m3-body-large);
    opacity: 0.5;
  }
</style>
