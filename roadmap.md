# Gorrson Roadmap & Memory Management

## Current Optimizations (Implemented)
- **Scraping Concurrency Limit**: Global semaphore restricts high-memory HTML parsing to a maximum of 5 simultaneous tasks.
- **Sync Concurrency Limit**: Global semaphore restricts concurrent user database synchronization to a maximum of 10 users.
- **Initial Sync Jitter**: Random delay (0-120s) added to background workers at startup to prevent resource spikes (Thundering Herd effect).
- **DOM Sharing & Streaming**: Refactored `ScrapeFullContent` to use `io.Pipe` for streaming to Readability and unified DOM tree processing with `goquery` and `htmlquery` without redundant string conversions.

## Planned Memory Tuning (Punto 5)
Si tras las optimizaciones actuales el consumo de RAM sigue siendo elevado en picos de carga:

1. **Ajuste del Garbage Collector (GOGC)**:
   - Configurar la variable de entorno `GOGC` a un valor menor (ej. `GOGC=50`) para que el recolector sea más agresivo a costa de un ligero aumento de CPU.
   - Utilizar `GOMEMLIMIT` para establecer un límite suave de memoria antes de que el GC sea extremadamente agresivo.

2. **Liberación Manual de Memoria**:
   - Evaluar la llamada a `runtime/debug.FreeOSMemory()` después de ciclos de sincronización masivos o tareas de scraping muy pesadas.
