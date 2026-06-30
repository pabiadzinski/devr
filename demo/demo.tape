Output demo.gif

Set Shell fish
Set FontSize 14
Set Width 1200
Set Height 700
Set Padding 20
Set Theme "Catppuccin Latte"

# Build, run, and drop straight into the log viewer
Type "devr app run"
Enter
Sleep 9s

# Vim-style navigation
Type "jjjjj"
Sleep 1.5s

# Filter to errors only...
Type "1"
Sleep 2.5s

# ...then warnings
Type "2"
Sleep 2.5s

# Back to everything
Type "0"
Sleep 1.5s

# Filter lines down to a single endpoint
Type "/"
Sleep 0.5s
Type "/api/users"
Sleep 1s
Enter
Sleep 2.5s

# Clear the filter
Type "0"
Sleep 1.5s

# Pretty-print the selected JSON line
Tab
Sleep 3s
Type "jjj"
Sleep 2s
Tab
Sleep 1.5s

# Detach — the app keeps running in the background
Type "q"
Sleep 1.5s
