<script>
  import { t, locale } from './i18n.js';
  import { Globe, Shield, Info, LogOut, Palette, Moon, Sun, Monitor, Settings as SettingsIcon } from 'lucide-svelte';
  import Filters from './Filters.svelte';
  import { onMount } from 'svelte';
  import { Share2, Plus, Trash2, ExternalLink as ExtIcon } from 'lucide-svelte';

  export let onLogout = () => {};
  export let feeds = [];
  export let username = '';
  export let scrollRead = false;

  let activeTab = 'general';
  
  // Theme state
  let currentTheme = localStorage.getItem('gorrson-theme') || 'lavender';
  let currentMode = localStorage.getItem('gorrson-mode') || 'dark';
  let customColor = localStorage.getItem('gorrson-custom-color') || '#6750A4';


  const modes = [
    { id: 'light', icon: Sun },
    { id: 'dark', icon: Moon },
    { id: 'system', icon: Monitor }
  ];

  function hexToHsl(hex) {
    let r = parseInt(hex.slice(1, 3), 16) / 255;
    let g = parseInt(hex.slice(3, 5), 16) / 255;
    let b = parseInt(hex.slice(5, 7), 16) / 255;
    let max = Math.max(r, g, b), min = Math.min(r, g, b);
    let h, s, l = (max + min) / 2;
    if (max === min) h = s = 0;
    else {
      let d = max - min;
      s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
      switch (max) {
        case r: h = (g - b) / d + (g < b ? 6 : 0); break;
        case g: h = (b - r) / d + 2; break;
        case b: h = (r - g) / d + 4; break;
      }
      h /= 6;
    }
    return [h * 360, s * 100, l * 100];
  }

  function hslToHex(h, s, l) {
    l /= 100;
    const a = s * Math.min(l, 1 - l) / 100;
    const f = n => {
      const k = (n + h / 30) % 12;
      const color = l - a * Math.max(Math.min(k - 3, 9 - k, 1), -1);
      return Math.round(255 * color).toString(16).padStart(2, '0');
    };
    return `#${f(0)}${f(8)}${f(4)}`;
  }

  function generateTonal(hex, mode) {
    const [h, s, l] = hexToHsl(hex);
    const isDark = mode === 'dark';

    // MD3 Expressive: Vibrant palettes with higher chroma
    const primary = isDark ? hslToHex(h, Math.max(s, 50), 80) : hex;
    const secondary = hslToHex((h + 25) % 360, Math.max(s - 15, 30), isDark ? 70 : 40);
    const secondaryContainer = isDark 
      ? hslToHex((h + 25) % 360, Math.max(s - 15, 30), 25)
      : hslToHex((h + 25) % 360, Math.max(s - 15, 30), 90);
    const onSecondaryContainer = isDark
      ? hslToHex((h + 25) % 360, Math.max(s - 15, 30), 90)
      : hslToHex((h + 25) % 360, Math.max(s - 15, 30), 10);
    const tertiary = hslToHex((h + 80) % 360, Math.max(s, 40), isDark ? 75 : 45);

    
    // Adwaita 4 Standard Surfaces (Tinted slightly with Seed Hue)
    const adwaita = {
      dark: { 
        bg: hslToHex(h, 5, 12), 
        surface: hslToHex(h, 6, 16), 
        card: hslToHex(h, 7, 20), 
        border: hslToHex(h, 10, 24) 
      },
      light: { 
        bg: '#FFFFFF', 
        surface: hslToHex(h, 10, 96), 
        card: '#FFFFFF', 
        border: hslToHex(h, 15, 88) 
      }
    };
    const currentA = isDark ? adwaita.dark : adwaita.light;

    if (!isDark) {
      return {
        primary: hex,
        onPrimary: '#FFFFFF',
        container: hslToHex(h, Math.max(s, 70), 92), // Very saturated for "Expressive"
        onContainer: hslToHex(h, s, 10),
        secondary: secondary,
        secondaryContainer: secondaryContainer,
        onSecondaryContainer: onSecondaryContainer,
        tertiary: tertiary,
        surface: currentA.surface,
        onSurface: hslToHex(h, 20, 10),
        onSurfaceVariant: hslToHex(h, 15, 40),
        background: currentA.bg,
        variant: currentA.card,
        outline: currentA.border
      };
    } else {
      return {
        primary: primary,
        onPrimary: hslToHex(h, s, 15),
        container: hslToHex(h, Math.max(s, 60), 25), // Saturated highlight
        onContainer: hslToHex(h, Math.max(s, 40), 90),
        secondary: secondary,
        secondaryContainer: secondaryContainer,
        onSecondaryContainer: onSecondaryContainer,
        tertiary: tertiary,
        surface: currentA.surface,
        onSurface: hslToHex(h, 10, 85),
        onSurfaceVariant: hslToHex(h, 15, 65),
        background: currentA.bg,
        variant: currentA.card,
        outline: currentA.border
      };
    }
  }

  function applyAppearance(themeId, modeId, seed) {
    if (typeof document === 'undefined') return;
    const root = document.documentElement;
    
    let mode = modeId;
    if (mode === 'system') {
      mode = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }

    // Since we only use Seed Color now
    const tonals = generateTonal(seed, mode);
    
    // MD3 Core Tokens
    root.style.setProperty('--md-primary', tonals.primary);
    root.style.setProperty('--md-on-primary', tonals.onPrimary);
    root.style.setProperty('--md-primary-container', tonals.container);
    root.style.setProperty('--md-on-primary-container', tonals.onContainer);
    
    root.style.setProperty('--md-secondary', tonals.secondary);
    root.style.setProperty('--md-secondary-container', tonals.secondaryContainer);
    root.style.setProperty('--md-on-secondary-container', tonals.onSecondaryContainer);
    root.style.setProperty('--md-tertiary', tonals.tertiary);

    // Adwaita + Tonal Alignment
    root.style.setProperty('--md-surface', tonals.surface);
    root.style.setProperty('--md-on-surface', tonals.onSurface);
    root.style.setProperty('--md-on-surface-variant', tonals.onSurfaceVariant);
    
    root.style.setProperty('--md-background', tonals.background);
    root.style.setProperty('--md-surface-variant', tonals.variant);
    root.style.setProperty('--md-outline', tonals.outline);
    
    // UI Effects
    root.style.setProperty('--shadow-color', mode === 'light' ? 'rgba(0,0,0,0.06)' : 'rgba(0,0,0,0.3)');

    localStorage.setItem('gorrson-theme', themeId);
    localStorage.setItem('gorrson-mode', modeId);
    localStorage.setItem('gorrson-custom-color', seed);
  }

  $: if (typeof document !== 'undefined') {
     applyAppearance('custom', currentMode, customColor);
  }

  const toggleLanguage = () => {
    locale.update(l => l === 'en' ? 'es' : 'en');
  };



  $: categoriesList = Array.from(new Set(feeds.map(f => f.category || 'General'))).sort();

  let editingCat = null;
  let newName = "";

  async function renameCategory(oldName) {
    if (!newName || newName === oldName) {
       editingCat = null;
       return;
    }
    const res = await fetch(`/categories/${encodeURIComponent(oldName)}`, {
      method: 'PUT',
      headers: { 
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify({ name: newName })
    });
    if (res.ok) {
      window.location.reload(); // Hard reload to refresh everything
    }
    editingCat = null;
  }

  // Export Services state
  let exportServices = JSON.parse(localStorage.getItem('gorrson-export-services') || '[]');
  let newServiceName = "";
  let newServiceTemplate = "";

  function saveServices() {
    localStorage.setItem('gorrson-export-services', JSON.stringify(exportServices));
    // Trigger update in app
    window.dispatchEvent(new Event('storage'));
  }

  function addService() {
    if (!newServiceName || !newServiceTemplate) return;
    exportServices = [...exportServices, { 
      id: Date.now().toString(), 
      name: newServiceName, 
      template: newServiceTemplate 
    }];
    newServiceName = "";
    newServiceTemplate = "";
    saveServices();
  }

  function removeService(id) {
    exportServices = exportServices.filter(s => s.id !== id);
    saveServices();
  }

  const examples = [
    { name: 'Obsidian', template: 'obsidian://new?title={title}&content={url}' },
    { name: 'Pocket', template: 'https://getpocket.com/edit?url={url}' },
    { name: 'WhatsApp', template: 'https://wa.me/?text={title}%20{url}' },
    { name: 'Telegram', template: 'https://t.me/share/url?url={url}&text={title}' }
  ];
</script>

<div class="view-container">
  <header class="header-bar">
    <div class="header-left">
      <SettingsIcon size={20} />
      <h1 class="title">{$t('settings')}</h1>
    </div>
    <div class="tabs">
      <button class="tab {activeTab === 'general' ? 'active' : ''}" on:click={() => activeTab = 'general'}>General</button>
      <button class="tab {activeTab === 'categories' ? 'active' : ''}" on:click={() => activeTab = 'categories'}>{$t('categories')}</button>
      <button class="tab {activeTab === 'filters' ? 'active' : ''}" on:click={() => activeTab = 'filters'}>{$t('filters')}</button>
      <button class="tab {activeTab === 'sharing' ? 'active' : ''}" on:click={() => activeTab = 'sharing'}>{$t('sharing')}</button>
    </div>
  </header>

  <!-- Sharp edge fix for unibody rounding -->
  <div class="corner-bg"></div>
  <div class="corner-arc"></div>

  <div class="content-scroll scroll-area">
    {#if activeTab === 'general'}
      <div class="settings-section">
        <div class="section-header">
          <Palette size={18} />
          <span>Appearance</span>
        </div>
        
        <div class="setting-item">
          <div class="text">
            <div class="label">Accent Color</div>
            <div class="desc">Tints the entire workspace with your brand/style.</div>
          </div>
          <div class="seed-compact-picker">
            <span class="seed-hex">{customColor.toUpperCase()}</span>
            <div class="color-preview-circle" style="background: {customColor}">
              <input type="color" bind:value={customColor} />
            </div>
          </div>
        </div>

        <div class="setting-item">
          <div class="text">
            <div class="label">Theme Mode</div>
            <div class="desc">Switch between light and dark backgrounds.</div>
          </div>
          <div class="mode-selector">
            {#each modes as mode}
              <button 
                class="mode-btn {currentMode === mode.id ? 'active' : ''}" 
                on:click={() => currentMode = mode.id}
                title={mode.id}
              >
                <svelte:component this={mode.icon} size={18} />
              </button>
            {/each}
          </div>
        </div>

        <div class="setting-item">
          <div class="text">
            <div class="label">Language / Idioma</div>
            <div class="desc">Chose your preferred interface language.</div>
          </div>
          <button class="btn-toggle" on:click={toggleLanguage}>
            {$locale === 'en' ? 'English' : 'Español'}
          </button>
        </div>
        <div class="setting-item">
          <div class="text">
            <div class="label">Auto-mark Read</div>
            <div class="desc">Mark articles as read when you scroll past them.</div>
          </div>
          <button class="btn-toggle" on:click={() => scrollRead = !scrollRead}>
            {scrollRead ? 'ON' : 'OFF'}
          </button>
        </div>
      </div>

      <div class="settings-section">
        <div class="section-header">
          <Shield size={18} />
          <span>Account</span>
        </div>
        <div class="setting-item">
          <div class="text">
            <div class="label">User Profile</div>
            <div class="desc">Logged in as <strong>{username}</strong></div>
          </div>
          <button class="btn-danger" on:click={onLogout}>
            <LogOut size={16} />
            <span>{$t('logout')}</span>
          </button>
        </div>
      </div>

      <div class="settings-section">
        <div class="section-header">
          <Info size={18} />
          <span>System</span>
        </div>
        <div class="info-grid">
          <div class="info-item">
            <span class="key">Version</span>
            <span class="value">v1.2.0-stable</span>
          </div>
          <div class="info-item">
            <span class="key">Engine</span>
            <span class="value">Go 1.22 + SQLite FTS5</span>
          </div>
          <div class="info-item">
            <span class="key">Architecture</span>
            <span class="value">Multi-tenant (Isolated databases)</span>
          </div>
        </div>
      </div>
    {:else if activeTab === 'categories'}
      <div class="settings-section">
        <div class="section-header">
          <Globe size={18} />
          <span>Manage Categories</span>
        </div>
        <div class="category-manager">
          {#each categoriesList as cat}
            <div class="cat-row">
              {#if editingCat === cat}
                <input 
                  class="edit-input"
                  bind:value={newName}
                  on:keydown={e => e.key === 'Enter' && renameCategory(cat)}
                  on:blur={() => editingCat = null}
                />
                <button class="btn-primary-small" on:click={() => renameCategory(cat)}>Save</button>
              {:else}
                <span class="cat-name">{cat}</span>
                <div class="row-actions">
                  <button class="btn-ghost" on:click={() => { editingCat = cat; newName = cat; }}>Rename</button>
                </div>
              {/if}
            </div>
          {/each}
        </div>
      </div>
    {:else if activeTab === 'sharing'}
      <div class="settings-section">
        <div class="section-header">
          <Share2 size={18} />
          <span>Custom Sharing & Export</span>
        </div>
        
        <div class="service-form">
          <input bind:value={newServiceName} placeholder={$t('service_name')} class="modern-input" />
          <input bind:value={newServiceTemplate} placeholder={$t('service_template')} class="modern-input wide" />
          <button class="btn-primary" on:click={addService}>
            <Plus size={18} />
            <span>{$t('add_service')}</span>
          </button>
        </div>

        <div class="examples-row">
          <span class="label">Presets:</span>
          {#each examples as ex}
            <button class="example-chip" on:click={() => { newServiceName = ex.name; newServiceTemplate = ex.template; }}>
              {ex.name}
            </button>
          {/each}
        </div>

        <div class="services-list">
          {#each exportServices as svc}
            <div class="svc-card">
              <div class="svc-info">
                <span class="svc-name">{svc.name}</span>
                <span class="svc-template">{svc.template}</span>
              </div>
              <button class="btn-icon-danger" on:click={() => removeService(svc.id)}>
                <Trash2 size={18} />
              </button>
            </div>
          {:else}
            <div class="empty-list">No custom services added yet. Use presets to get started.</div>
          {/each}
        </div>
      </div>
    {:else}
      <div class="settings-section">
        <div class="section-header">
          <Shield size={18} />
          <span>Filters & Content Cleaning</span>
        </div>
        <div class="filters-view-embedded">
          <Filters {feeds} headless={true} />
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .view-container { 
    display: flex; 
    flex-direction: column; 
    height: 100vh; 
    background: var(--md-surface); 
    flex: 1; 
    width: 100%; 
    position: relative;
    overflow: hidden;
  }
  .header-bar { 
    height: var(--header-height); 
    padding: 0 40px; 
    display: flex; 
    align-items: center; 
    background: var(--md-surface); 
    gap: 48px; 
    box-sizing: border-box;
    position: relative;
    z-index: 20;
  }

  /* Eraser for the straight line crossing the curved gap */
  .header-bar::before {
    content: "";
    position: absolute;
    left: 0px; 
    bottom: -1px; 
    width: 32px; 
    height: 2px;
    background: var(--md-surface);
    z-index: 15;
  }

  /* Solid background to color the gap outside the feed-content radius */
  .corner-bg {
    position: absolute;
    top: var(--header-height);
    left: 0;
    width: 32px;
    height: 32px;
    background: var(--md-surface);
    z-index: 0;
  }

  /* Overlaid pure arc to draw the crisp 1px unibody curve without clipping */
  .corner-arc {
    position: absolute;
    top: var(--header-height);
    left: 0;
    width: 32px;
    height: 32px;
    border-top-left-radius: 32px;
    border-top: 1px solid var(--md-outline);
    border-left: 1px solid var(--md-outline);
    z-index: 20;
    pointer-events: none;
  }

  .header-left { display: flex; align-items: center; gap: 12px; }
  .header-left :global(svg) { color: var(--md-primary); }
  .title { font: var(--m3-headline-small); font-weight: 700; color: var(--md-primary); }

  .tabs { display: flex; gap: 8px; height: 100%; align-items: center; padding-left: 24px; }
  .tab { 
    background: transparent; 
    border: none; 
    color: var(--md-on-surface-variant); 
    font: var(--m3-label-large); 
    padding: 8px 16px; 
    height: 36px;
    display: flex;
    align-items: center;
    cursor: pointer; 
    border-radius: var(--radius-full);
    transition: all 0.2s;
    opacity: 0.8;
  }
  .tab:hover { 
    background: var(--md-surface-variant);
    opacity: 1; 
    color: var(--md-on-surface); 
  }
  .tab.active { 
    background: var(--md-primary-container);
    color: var(--md-primary); 
    font-weight: 800;
    opacity: 1;
  }

  .content-scroll { 
    padding: 48px; 
    flex: 1; 
    width: 100%; 
    box-sizing: border-box; 
    overflow-y: auto; 
    background: var(--md-background);
    border-top-left-radius: 32px;
    border-left: 1px solid var(--md-outline);
    position: relative;
    z-index: 10;
  }
  
  .settings-section { margin-bottom: 48px; max-width: 900px; }
  .section-header { 
    display: flex; 
    align-items: center; 
    gap: 12px; 
    font: var(--m3-title-medium); 
    color: var(--md-primary); 
    margin-bottom: 24px; 
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .setting-item {
    background: var(--md-surface);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-xl);
    padding: 20px 24px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
    box-shadow: 0 1px 3px rgba(0,0,0,0.05);
  }

  .text .label { font: var(--m3-title-medium); color: var(--md-on-surface); margin-bottom: 4px; }
  .text .desc { font: var(--m3-body-small); color: var(--md-on-surface-variant); }

  .btn-toggle {
    background: var(--md-primary-container);
    color: var(--md-on-primary-container);
    border: 1px solid var(--md-primary);
    padding: 8px 16px;
    border-radius: var(--radius-full);
    font: var(--m3-label-large);
    cursor: pointer;
    font-weight: 700;
  }

  .btn-danger {
    background: #ff525222;
    color: #ff5252;
    border: 1px solid #ff525244;
    padding: 8px 20px;
    border-radius: var(--radius-full);
    font: var(--m3-label-large);
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 700;
  }

  .info-grid {
    background: var(--md-surface);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-xl);
    padding: 24px;
    display: grid;
    gap: 12px;
  }

  .info-item { display: flex; justify-content: space-between; font: var(--m3-body-medium); }
  .info-item .key { color: var(--md-on-surface-variant); }
  .info-item .value { color: var(--md-on-surface); font-weight: 600; font-family: var(--font-mono); font-size: 13px; }



  .seed-compact-picker { 
    display: flex; 
    align-items: center; 
    gap: 12px; 
    background: var(--md-surface-variant); 
    padding: 6px 12px; 
    border-radius: var(--radius-full); 
    border: 1px solid var(--md-outline);
  }
  .seed-hex { 
    font-family: var(--font-mono); 
    font-size: 13px; 
    font-weight: 700; 
    color: var(--md-primary); 
    letter-spacing: 0.5px;
  }
  .color-preview-circle { 
    width: 24px; 
    height: 24px; 
    border-radius: 50%; 
    border: 2px solid white; 
    box-shadow: 0 0 0 1px var(--md-outline); 
    position: relative; 
    cursor: pointer;
    transition: transform 0.2s;
  }
  .color-preview-circle:hover { transform: scale(1.15); }
  .color-preview-circle input { 
    position: absolute; 
    inset: 0; 
    width: 100%; 
    height: 100%; 
    opacity: 0; 
    cursor: pointer; 
  }

  .mode-selector { display: flex; background: var(--md-surface); padding: 4px; border-radius: var(--radius-full); border: 1px solid var(--md-outline); }
  .mode-btn { 
    background: transparent; 
    border: none; 
    color: var(--md-on-surface-variant); 
    width: 36px; 
    height: 36px; 
    display: flex; 
    align-items: center; 
    justify-content: center; 
    border-radius: var(--radius-full);
    cursor: pointer;
    transition: all 0.2s;
  }
  .mode-btn.active { background: var(--md-primary); color: var(--md-on-primary); }
  .category-manager {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: 16px;
  }
  .cat-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 24px;
    background: var(--md-surface);
    box-sizing: border-box;
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-xl);
    transition: all 0.2s;
  }
  .cat-row:hover {
    border-color: var(--md-primary);
    background: var(--md-surface-variant);
  }
  .btn-ghost {
    background: transparent;
    border: 1px solid var(--md-outline);
    color: var(--md-primary);
    padding: 6px 16px;
    border-radius: var(--radius-full);
    font: var(--m3-label-large);
    font-weight: 700;
    cursor: pointer;
    transition: all 0.2s;
  }
  .btn-ghost:hover {
    background: var(--md-primary-container);
    border-color: var(--md-primary);
  }
  .cat-row:hover {
    border-color: var(--md-primary);
  }
  .btn-ghost {
    background: transparent;
    border: 1px solid var(--md-outline);
    color: var(--md-primary);
    padding: 6px 16px;
    border-radius: var(--radius-full);
    font: var(--m3-label-large);
    font-weight: 700;
    cursor: pointer;
    transition: all 0.2s;
  }
  .btn-ghost:hover {
    background: var(--md-primary-container);
    border-color: var(--md-primary);
  }
  .cat-name {
    font: var(--m3-body-large);
    font-weight: 700;
    color: var(--md-on-surface);
  }
  .edit-input {
    flex: 1;
    margin-right: 12px;
    padding: 8px 16px;
    border: 1px solid var(--md-primary);
    border-radius: var(--radius-full);
    background: var(--md-surface-variant);
    color: var(--md-on-surface);
    font: var(--m3-body-medium);
    outline: none;
  }
  .btn-primary-small {
    background: var(--md-primary);
    color: var(--md-on-primary);
    border: none;
    padding: 8px 16px;
    border-radius: var(--radius-full);
    font: var(--m3-label-medium);
    font-weight: 700;
    cursor: pointer;
  }
  .service-form { display: flex; gap: 12px; margin-bottom: 12px; }
  .modern-input {
    background: var(--md-surface);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-full);
    padding: 12px 20px;
    color: var(--md-on-surface);
    font: var(--m3-body-medium);
    outline: none;
    transition: all 0.2s;
  }
  .modern-input:focus { border-color: var(--md-primary); box-shadow: 0 0 0 2px var(--md-primary-container); }
  .modern-input.wide { flex: 1; }
  
  .btn-primary {
    background: var(--md-primary);
    color: var(--md-on-primary);
    border: none;
    padding: 12px 24px;
    border-radius: var(--radius-full);
    font: var(--m3-label-large);
    font-weight: 700;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .examples-row { display: flex; align-items: center; gap: 8px; margin-bottom: 32px; }
  .examples-row .label { font: var(--m3-label-medium); color: var(--md-on-surface-variant); }
  .example-chip {
    background: var(--md-surface-variant);
    border: 1px solid var(--md-outline);
    padding: 4px 12px;
    border-radius: var(--radius-full);
    font: var(--m3-label-small);
    color: var(--md-primary);
    cursor: pointer;
    transition: all 0.2s;
  }
  .example-chip:hover { border-color: var(--md-primary); background: var(--md-primary-container); }

  .services-list { display: flex; flex-direction: column; gap: 12px; }
  .svc-card {
    background: var(--md-surface);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-xl);
    padding: 16px 24px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    transition: all 0.2s;
  }
  .svc-card:hover { border-color: var(--md-primary); transform: translateX(4px); }
  .svc-info { display: flex; flex-direction: column; gap: 4px; overflow: hidden; }
  .svc-name { font: var(--m3-title-small); color: var(--md-on-surface); font-weight: 700; }
  .svc-template { font: var(--m3-body-small); color: var(--md-on-surface-variant); white-space: nowrap; text-overflow: ellipsis; overflow: hidden; font-family: var(--font-mono); }
  
  .btn-icon-danger {
    background: transparent;
    border: none;
    color: var(--md-on-surface-variant);
    cursor: pointer;
    padding: 8px;
    border-radius: 50%;
    transition: all 0.2s;
  }
  .btn-icon-danger:hover { background: #ff525222; color: #ff5252; }

  .empty-list { text-align: center; padding: 40px; color: var(--md-on-surface-variant); font: var(--m3-body-large); opacity: 0.6; }
</style>
