<script>
  import './app.css';
  import { fade, fly } from 'svelte/transition';

  let selectedId = 1;
  let searchQuery = "";

  const news = [
    { 
      id: 1, 
      title: "Quantum Leaps: The Next Era of Computing", 
      source: "The Verge", 
      time: "14 min", 
      category: "Tech", 
      content: "Quantum computing is no longer a distant dream. Recent breakthroughs in error correction and qubit stability are bringing us closer to a future where problems that take traditional computers centuries to solve can be completed in minutes. Experts believe the first commercial applications will appear in pharmaceuticals and cryptography.", 
      img: "https://images.unsplash.com/photo-1635070041078-e363dbe005cb?auto=format&fit=crop&w=800&q=80" 
    },
    { 
      id: 2, 
      title: "Market Trends: Q3 Analysis", 
      source: "Forbes", 
      time: "8 min", 
      category: "Finance", 
      content: "The global markets are showing signs of stabilization despite inflationary pressures and geopolitical uncertainty. Central banks are maintaining a cautious stance, while emerging markets show surprising resilience in the technology and green energy sectors. Investors are now looking towards the year-end projections with cautious optimism." 
    },
    { 
      id: 3, 
      title: "Product Design: Crafting the Future", 
      source: "Design", 
      time: "10 min", 
      category: "Design", 
      content: "Modern design is shifting from purely aesthetic to deeply functional and sustainable. The 'Bento Box' layout trend is just one example of how digital products are organizing complex information into manageable, delightful chunks. We explore how physical hardware design is following this modular philosophy.",
      img: "https://images.unsplash.com/photo-1558655146-d09347e92766?auto=format&fit=crop&w=600&q=80" 
    },
  ];

  $: selectedArticle = news.find(n => n.id === selectedId);
</script>

<main class="app-container">
  <!-- 1. SIDEBAR (Navegación) -->
  <aside style="width: var(--sidebar-width); border-right: 1px solid var(--border-glass); padding: 32px 24px; display: flex; flex-direction: column; gap: 32px; background: var(--bg-sidebar);">
    <div style="font-weight: 900; font-size: 22px; color: var(--accent-amber); letter-spacing: -1.5px;">GORRSON</div>
    
    <nav style="display: flex; flex-direction: column; gap: 8px;">
      <div class="nav-item active">
        <span>Inbox</span>
        <span class="badge">12</span>
      </div>
      <div class="nav-item">Favorites</div>
      <div class="nav-item">Saved</div>
    </nav>

    <div style="margin-top: auto; padding: 20px; border-radius: 16px; background: rgba(255,255,255,0.03); border: 1px solid var(--border-glass);">
      <div style="font-size: 11px; color: var(--text-dim); text-transform: uppercase; letter-spacing: 0.05em; margin-bottom: 4px;">Subscription</div>
      <div style="font-size: 14px; font-weight: 600; color: var(--text-primary);">Gold Member</div>
    </div>
  </aside>

  <!-- 2. FEED (Lista de noticias) -->
  <section style="width: var(--feed-width); border-right: 1px solid var(--border-glass); display: flex; flex-direction: column;" class="scroll-area">
    <header style="padding: 32px 24px 20px 24px; position: sticky; top: 0; background: var(--bg-obsidian); z-index: 10;">
      <h1 style="font-size: 26px; margin: 0; font-weight: 700;">Latest</h1>
      <p style="color: var(--text-dim); font-size: 13px; margin: 4px 0 0 0;">Monday, April 6</p>
    </header>

    <div style="padding: 0 16px 120px 16px; display: flex; flex-direction: column; gap: 8px;">
      {#each news as item}
        <button 
          class="feed-card {selectedId === item.id ? 'selected' : ''}" 
          on:click={() => selectedId = item.id}
        >
          <div style="display: flex; justify-content: space-between; margin-bottom: 8px;">
            <span style="font-size: 11px; font-weight: 600; color: var(--accent-amber); opacity: 0.8;">{item.source}</span>
            <span style="font-size: 11px; color: var(--text-dim);">{item.time}</span>
          </div>
          <h3 style="font-size: 15px; margin: 0 0 8px 0; line-height: 1.4; text-align: left; font-weight: 600; color: {selectedId === item.id ? 'white' : 'var(--text-secondary)'};">{item.title}</h3>
          <p style="font-size: 13px; color: var(--text-dim); margin: 0; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; text-align: left;">
            {item.content || "No preview available..."}
          </p>
        </button>
      {/each}
    </div>
  </section>

  <!-- 3. DETAIL (Visor del artículo) -->
  <article style="flex: 1; display: flex; flex-direction: column;" class="scroll-area">
    {#if selectedArticle}
      <div in:fade={{ duration: 250 }} style="padding: 80px 10% 120px 10%; max-width: 900px; margin: 0 auto;">
        {#if selectedArticle.img}
          <div style="width: 100%; height: 440px; border-radius: 24px; overflow: hidden; margin-bottom: 48px; box-shadow: 0 20px 60px rgba(0,0,0,0.4);">
            <img src={selectedArticle.img} alt="" style="width: 100%; height: 100%; object-fit: cover;" />
          </div>
        {/if}
        
        <div style="display: flex; gap: 12px; align-items: center; margin-bottom: 24px;">
          <span style="padding: 4px 12px; border-radius: 100px; background: var(--accent-amber-glow); color: var(--accent-amber); font-size: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em;">{selectedArticle.category}</span>
          <span style="color: var(--text-dim); font-size: 14px;">{selectedArticle.source} • {selectedArticle.time} read</span>
        </div>
        
        <h1 style="font-size: 52px; line-height: 1.05; font-weight: 800; margin-bottom: 32px; letter-spacing: -0.04em;">{selectedArticle.title}</h1>
        
        <p style="font-size: 20px; line-height: 1.7; color: var(--text-primary); opacity: 0.9; font-family: var(--font-serif); white-space: pre-wrap;">
          {selectedArticle.content}
        </p>
      </div>
    {/if}
  </article>

  <!-- Command Palette (Floating) -->
  <div class="command-palette glass">
    <span style="color: var(--accent-amber); font-size: 14px;">⚡</span>
    <input type="text" placeholder="Search feeds or articles... (⌘K)" bind:value={searchQuery} />
    <span class="kbd">⌘K</span>
  </div>
</main>

<style>
  .nav-item {
    padding: 10px 16px;
    border-radius: 12px;
    color: var(--text-secondary);
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    display: flex;
    justify-content: space-between;
    align-items: center;
    transition: all 0.2s;
  }
  .nav-item:hover { background: rgba(255,255,255,0.05); color: white; }
  .nav-item.active { background: var(--accent-amber-glow); color: var(--accent-amber); }
  
  .badge { font-size: 10px; background: rgba(255,255,255,0.08); padding: 2px 8px; border-radius: 100px; color: var(--text-secondary); }

  .feed-card {
    background: transparent;
    border: 1px solid transparent;
    border-radius: 16px;
    padding: 20px;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    width: 100%;
    outline: none;
  }
  .feed-card:hover { background: rgba(255,255,255,0.03); }
  .feed-card.selected { 
    background: var(--bg-card); 
    border-color: var(--border-glass);
    box-shadow: 0 10px 30px rgba(0,0,0,0.3);
  }

  .command-palette {
    position: fixed;
    bottom: 32px;
    left: 50%;
    transform: translateX(-50%);
    width: 520px;
    padding: 14px 24px;
    border-radius: 100px;
    display: flex;
    align-items: center;
    gap: 16px;
    box-shadow: 0 30px 60px rgba(0,0,0,0.6);
    z-index: 1000;
  }
  
  .command-palette input {
    background: transparent; border: none; color: white; outline: none; flex: 1; font-size: 14px;
    font-family: var(--font-main);
  }
</style>
