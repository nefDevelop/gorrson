<script>
  import { stripHtml, extractFirstImage } from './utils.js';
  import { Star, Share2, Image as ImageIcon } from 'lucide-svelte';
  /** @type {{id: string, title: string, source: string, time: string, category: string, is_read?: boolean, content?: string, summary?: string, selected?: boolean, is_favorite?: boolean, matched_action?: string}} */
  export let article;
  
  /** @type {(id: string | number) => void} */
  export let onClick;
  
  /** @type {(id: string) => void} */
  export let onToggleFavorite;
  
  /** @type {() => void} */
  export let onShare;

  /** @type {string} */
  export let viewMode = 'default'; // 'default', 'compact', 'minimal', 'image'
 
  $: summary = stripHtml(article.content || article.summary || "");
  $: imageUrl = (viewMode === 'image') ? extractFirstImage(article.content) : null;
</script>

<div 
  data-id={article.id}
  class="post-card {article.selected ? 'selected' : ''} {article.is_read ? 'read' : ''} {article.matched_action?.toLowerCase()} mode-{viewMode}" 
  on:click={() => onClick(article.id)}
  role="button"
  tabindex="0"
  on:keydown={(e) => e.key === 'Enter' && onClick(article.id)}
>
  <div class="card-layout">
    <div class="content-side">
      <div class="title-row">
        <h3 class="title" class:active-title={article.selected}>
          {article.title}
        </h3>
        {#if viewMode !== 'compact' && viewMode !== 'minimal'}
          <span class="time">{article.time}</span>
        {/if}
      </div>

      {#if viewMode !== 'minimal'}
        <p class="summary">
          {summary || "No preview available..."}
        </p>
      {/if}

      <div class="footer-row">
        <div class="meta">
          <span class="source-tag">{article.source}</span>
          {#if viewMode === 'compact' || viewMode === 'minimal'}
            <span class="dot">•</span>
            <span class="time-mini">{article.time}</span>
          {/if}
        </div>
        
        <div class="actions">
          <button 
            class="action-btn" 
            on:click|stopPropagation={onShare} 
            title="Share"
          >
            <Share2 size={14} strokeWidth={2} />
          </button>
          <button 
            class="action-btn" 
            class:starred={article.is_favorite} 
            on:click|stopPropagation={() => onToggleFavorite(article.id)}
          >
            <Star size={14} strokeWidth={2} />
          </button>
        </div>
      </div>
    </div>

    {#if viewMode === 'image' && imageUrl}
      <div class="image-side">
        <img src={imageUrl} alt="Article thumbnail" loading="lazy" />
      </div>
    {/if}
  </div>
</div>

<style>
  .post-card {
    background: var(--md-surface-variant);
    border: none;
    border-radius: 24px;
    padding: 16px 20px 12px 20px;
    margin-bottom: 8px;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0, 0, 0.2, 1);
    width: 100%;
    outline: none;
    display: block;
    text-align: left;
    position: relative;
    opacity: 0.8;
  }

  .post-card:not(.read) { opacity: 1; }

  .post-card:hover {
    background: var(--md-secondary-container);
    border-color: var(--md-primary);
    transform: translateX(2px);
    opacity: 1;
  }

  /* Layout Base */
  .card-layout { display: flex; gap: 16px; width: 100%; }
  .content-side { flex: 1; min-width: 0; display: flex; flex-direction: column; }
  
  .image-side { 
    width: 84px; 
    height: 84px; 
    flex-shrink: 0; 
    border-radius: 16px; 
    overflow: hidden; 
    background: var(--md-surface); 
    border: none;
    margin-top: 4px;
  }
  .image-side img { width: 100%; height: 100%; object-fit: cover; transition: transform 0.3s; }
  .post-card:hover .image-side img { transform: scale(1.05); }

  /* Modes */
  .mode-compact { padding: 12px 16px 8px 16px; border-radius: 16px; margin-bottom: 4px; }
  .mode-compact .title { font-size: 15px; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; display: -webkit-box; }
  .mode-compact .summary { font-size: 12px; -webkit-line-clamp: 1; -webkit-box-orient: vertical; overflow: hidden; display: -webkit-box; margin-top: 2px; }
  
  .mode-minimal { padding: 10px 14px; border-radius: 12px; margin-bottom: 4px; }
  .mode-minimal .title { font-size: 14px; font-weight: 500; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; display: -webkit-box; margin-bottom: 2px; }
  .mode-minimal .footer-row { margin-top: 2px; }
  .mode-minimal .action-btn { padding: 2px; }

  /* Shared Components */
  .title-row {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 12px;
    margin-bottom: 8px;
  }

  .title {
    font-family: var(--font-display);
    font-size: 18px;
    font-weight: 500;
    margin: 0;
    flex: 1;
    line-height: 1.3;
    color: var(--md-on-surface-variant);
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
    transition: all 0.2s;
  }

  .post-card:not(.read) .title { font-weight: 800; color: var(--md-on-surface); }
  .post-card.selected .title { color: var(--md-primary); }

  .time {
    font-family: var(--font-main);
    font-size: 11px;
    color: var(--md-on-surface-variant);
    opacity: 0.6;
    flex-shrink: 0;
    white-space: nowrap;
    margin-top: 4px;
    font-weight: 500;
  }

  .post-card:not(.read) .time { color: var(--md-primary); opacity: 1; font-weight: 700; }

  .summary {
    font-family: var(--font-main);
    font-size: 13px;
    color: var(--md-on-surface-variant);
    margin: 0;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    line-height: 1.5;
    opacity: 0.6;
  }

  .post-card:not(.read) .summary { opacity: 0.85; }

  .footer-row {
    margin-top: 10px;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .meta { display: flex; align-items: center; gap: 8px; min-width: 0; }
  .dot { opacity: 0.3; font-size: 12px; }
  .time-mini { font-size: 11px; opacity: 0.6; white-space: nowrap; }

  .source-tag {
    font-size: 10px;
    font-weight: 600;
    color: var(--md-on-surface-variant);
    opacity: 0.5;
    text-transform: uppercase;
    letter-spacing: 0.6px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 120px;
  }

  .actions { display: flex; gap: 4px; }

  .action-btn {
    background: none;
    border: none;
    padding: 6px;
    border-radius: 50%;
    color: var(--md-on-surface-variant);
    opacity: 0.4;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s ease;
  }

  .action-btn:hover { opacity: 1; background: var(--md-surface-variant); color: var(--md-primary); }
  .action-btn.starred { color: #FFB300; opacity: 1; fill: #FFB300; }

  /* Specialty States */
  .post-card.highlight { background: var(--md-secondary-container); border-color: var(--md-secondary); border-width: 2px; }
  .post-card.silent { opacity: 0.25 !important; filter: grayscale(1); transform: scale(0.98); }
</style>
