import re
import os

methods = [
    "AddFeed", "AddEntry", "AddEntryTx", "ListFeeds", "ListEntries", "GetEntryByID", 
    "UpdateEntryContent", "MarkAsRead", "MarkAsReadByFeverID", "ToggleFavoriteByFeverID", 
    "MarkAllRead", "ToggleFavorite", "CleanupOldEntries", "SearchEntries", "GetFeedByURL", 
    "UpsertUser", "GetUserHash", "GetUserFeverKey", "DeleteFeed", "UpdateFeed", 
    "RenameCategory", "GetFeedByID", "AddFilter", "ListFilters", "GetFiltersForFeed", 
    "DeleteFilter", "CreateSession", "GetSession", "DeleteSession", "DeleteExpiredSessions", 
    "AddAPIKey", "ListAPIKeys", "DeleteAPIKey", "ValidateAPIKey", "GetStats"
]

for filename in os.listdir('internal/api'):
    if not filename.endswith('.go'):
        continue
    filepath = os.path.join('internal/api', filename)
    with open(filepath, 'r') as f:
        content = f.read()
    
    for method in methods:
        # We need to replace store.Method( with store.Method(r.Context(),
        # or store.Method() with store.Method(r.Context())
        # but what if it's in a goroutine? In handleImportOPML goroutine, it uses context.Background()
        
        # Regex to find store.Method(
        pattern = r'store\.' + method + r'\('
        
        # Find all occurrences
        matches = list(re.finditer(pattern, content))
        # Process from back to front to preserve indices
        for match in reversed(matches):
            start = match.start()
            end = match.end()
            
            # Check if it already has context
            if content[end:end+4] == 'ctx,' or content[end:end+11] == 'r.Context()':
                continue
            
            # If inside a goroutine we might need context.Background()
            # A simple heuristic: if 'r *http.Request' is in scope, r.Context() usually works.
            # We'll use 'r.Context()' for all, and fix manually if it doesn't compile.
            # Except in handler_feeds.go line 144 (goroutine in handleImportOPML), it has 'context.Background()' already but we didn't add it to store yet.
            # Actually, let's just insert r.Context() if it's not empty, else r.Context()
            
            if content[end] == ')':
                content = content[:end] + 'r.Context()' + content[end:]
            else:
                content = content[:end] + 'r.Context(), ' + content[end:]
                
    with open(filepath, 'w') as f:
        f.write(content)

