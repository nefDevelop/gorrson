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

files_to_check = []
for root, dirs, files in os.walk('scripts'):
    for f in files:
        if f.endswith('.go'):
            files_to_check.append(os.path.join(root, f))

for filepath in files_to_check:
    with open(filepath, 'r') as f:
        content = f.read()
    
    for method in methods:
        pattern = r'(store|st)\.' + method + r'\('
        
        matches = list(re.finditer(pattern, content))
        for match in reversed(matches):
            start = match.start()
            end = match.end()
            
            if content[end:end+4] == 'ctx,' or content[end:end+11] == 'r.Context()' or content[end:end+20] == 'context.Background()':
                continue
            
            if content[end] == ')':
                content = content[:end] + 'context.Background()' + content[end:]
            else:
                content = content[:end] + 'context.Background(), ' + content[end:]
                
    with open(filepath, 'w') as f:
        f.write(content)

