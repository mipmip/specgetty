# VHS tape file for specgetty preview GIF
# Run with: vhs demo.tape

Output demo.gif

Require spg

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set Padding 10
Set Theme "Catppuccin Mocha"

# Launch specgetty pointing at the demo openspec project
Type "spg ."
Enter
Sleep 3s

# Navigate down through the project list
Down
Sleep 1s
Down
Sleep 1s

# Switch focus to the detail panel
Tab
Sleep 1.5s

# Scroll down inside the detail view
Down
Sleep 0.5s
Down
Sleep 0.5s
Down
Sleep 1s

# Switch to the changes tab
Right
Sleep 1.5s

# Switch to the archive tab
Right
Sleep 1.5s

# Switch to the config tab
Right
Sleep 1.5s

# Switch back to specs tab
Left
Left
Left
Sleep 1s

# Switch focus back to project list
Tab
Sleep 1s

# Quit
Type "q"
Sleep 1s
