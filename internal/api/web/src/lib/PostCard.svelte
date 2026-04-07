<script>
  import { stripHtml } from './utils.js';
  import { Star, Share2 } from 'lucide-svelte';
  /** @type {{id: string, title: string, source: string, time: string, category: string, content?: string, summary?: string, selected?: boolean, is_favorite?: boolean}} */
  export let article;
  
  /** @type {(id: string | number) => void} */
  export let onClick;
  
  /** @type {(id: string) => void} */
  export let onToggleFavorite;
  
  /** @type {() => void} */
  export let onShare;

  $: summary = stripHtml(article.content || article.summary || "");
</script>

<div 
  class="post-card {article.selected ? 'selected' : ''}" 
  on:click={() => onClick(article.id)}
  role="button"
  tabindex="0"
  on:keydown={(e) => e.key === 'Enter' && onClick(article.id)}
>
  <div class="title-row">
    <h3 class="title" class:active-title={article.selected}>{article.title}</h3>
    <span class="time">{article.time}</span>
  </div>

  <p class="summary">
    {summary || "No preview available..."}
  </p>

  <div class="footer-row">
    <div class="actions">
      <button 
        class="action-btn" 
        on:click|stopPropagation={onShare} 
        title="Share"
      >
        <Share2 size={16} strokeWidth={2} />
      </button>
      <button 
        class="action-btn" 
        class:starred={article.is_favorite} 
        on:click|stopPropagation={() => onToggleFavorite(article.id)}
      >
        <Star size={16} strokeWidth={2} />
      </button>
    </div>
  </div>
</div>

<style>
  .post-card {
    background: var(--md-surface-variant);
    border: 1px solid var(--md-outline);
    border-radius: 24px;
    padding: 16px 20px 12px 20px; /* Reduced bottom padding */
    margin-bottom: 8px;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0, 0, 0.2, 1);
    width: 100%;
    outline: none;
    display: block;
    text-align: left;
  }

  .post-card:hover {
    background: var(--md-secondary-container);
    border-color: var(--md-primary);
    transform: translateX(2px);
  }

  .post-card.selected {
    background: var(--md-primary-container);
    border-color: var(--md-primary);
    box-shadow: none; /* Adwaita 4 prefers flat surfaces */
  }

  .post-card.selected .title,
  .post-card.selected .summary,
  .post-card.selected .time {
    color: var(--md-on-primary-container);
    opacity: 1;
  }
  .title-row {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 12px;
    margin-bottom: 12px; /* Slightly reduced */
  }

  .title {
    font-family: var(--font-display);
    font-size: 18px;
    font-weight: 800;
    margin: 0;
    flex: 1;
    line-height: 1.35;
    color: var(--md-on-surface);
    display: -webkit-box;
    -webkit-line-clamp: 3;
    line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

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

  .summary {
    font-family: var(--font-main);
    font-size: 13px;
    color: var(--md-on-surface-variant);
    margin: 0;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
    line-height: 1.5;
    opacity: 0.8;
  }

  .footer-row {
    margin-top: 8px; /* Reduced from 16px */
    display: flex;
    justify-content: flex-end;
    align-items: center;
  }

  .actions {
    display: flex;
    gap: 8px;
  }

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

  .action-btn:hover {
    opacity: 1;
    background: var(--md-surface-variant);
    color: var(--md-primary);
  }

  .action-btn.starred {
    color: #FFB300; /* Subtle Gold */
    opacity: 1;
    fill: #FFB300;
  }

  .post-card.selected .action-btn {
    color: var(--md-on-primary-container);
    opacity: 0.6;
  }

  .post-card.selected .action-btn:hover {
    opacity: 1;
    background: rgba(255, 255, 255, 0.1);
  }

  .post-card.selected .action-btn.starred {
     color: #FFD54F;
     opacity: 1;
  }
</style>
