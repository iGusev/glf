# VHS documentation
#
# Output:
#   Output <path>.gif               Create a GIF output at the given <path>
#   Output <path>.mp4               Create an MP4 output at the given <path>
#   Output <path>.webm              Create a WebM output at the given <path>
#
# Settings:
#   Set FontSize <number>           Set the font size of the terminal
#   Set FontFamily <string>         Set the font family of the terminal
#   Set Height <number>             Set the height of the terminal
#   Set Width <number>              Set the width of the terminal
#   Set LetterSpacing <float>       Set the letter spacing of the terminal
#   Set LineHeight <float>          Set the line height of the terminal
#   Set Theme <string>              Set the theme of the terminal (JSON)
#   Set Padding <number>            Set the padding of the terminal
#   Set Framerate <number>          Set the framerate of the GIF
#   Set PlaybackSpeed <float>       Set the playback speed of the GIF
#
# Sleep:
#   Sleep <time>                    Sleep for a set amount of <time> in seconds
#
# Type:
#   Type[@<time>] "<characters>"    Type <characters> into the terminal with a
#                                   <time> delay between each character
#
# Keys:
#   Backspace[@<time>] [number]     Press the Backspace key
#   Down[@<time>] [number]          Press the Down key
#   Enter[@<time>] [number]         Press the Enter key
#   Space[@<time>] [number]         Press the Space key
#   Tab[@<time>] [number]           Press the Tab key
#   Left[@<time>] [number]          Press the Left Arrow key
#   Right[@<time>] [number]         Press the Right Arrow key
#   Up[@<time>] [number]            Press the Up Arrow key
#   Down[@<time>] [number]          Press the Down Arrow key
#   Ctrl+<key>                      Press the Control key + <key> (e.g. Ctrl+C)
#
# Display:
#   Hide                            Hide the subsequent commands from the output
#   Show                            Show the subsequent commands in the output

Output demo.webm

Set FontSize 16
Set Width 900
Set Height 600
Set Padding 20
Set Theme "Dracula"
Set Shell "bash"

# Setup alias for demo and fake project directory
Hide
Type "export PROJECT_ROOT=$(pwd)"
Enter
Type "alias glf='$PROJECT_ROOT/demo/demo-glf.sh'"
Enter
Type "clear"
Enter
Show

# Demo interactive search
Type@100ms "# GLF - Fast GitLab project search"
Enter
Sleep 1.5s

Type@200ms "glf"
Enter
Sleep 3s

# Type search query
Type@300ms "api"
Sleep 2s

# Navigate through results
Down 3
Sleep 800ms
Up 2
Sleep 800ms
Down 1
Sleep 1s

# Clear search and try another
Ctrl+U
Sleep 500ms
Type@300ms "mobile"
Sleep 2s

# Navigate again
Down 2
Sleep 800ms
Up 1
Sleep 1s

# Press Enter to open selected project in browser
Enter
Sleep 1s

# Show that URL was opened in browser
Type@30ms "# Opened in browser: https://gitlab.company.com/frontend/mobile/ios-app"
Enter
Sleep 2s

Type@100ms "clear"
Enter
Sleep 1s

# Demo -g flag (auto-select first result)
Type@100ms "# Use 'glf <query> -g' to auto-open first result"
Enter
Sleep 1.5s

Type@200ms "glf terraform -g"
Enter
Sleep 2s

# Show it opened
Type@30ms "# Opened in browser: https://gitlab.company.com/infrastructure/terraform/aws"
Enter
Sleep 2s

Type@100ms "clear"
Enter
Sleep 1s

# Demo opening current project with dot
Type@100ms "# Use 'glf .' to open current git project"
Enter
Sleep 1.5s

Type@150ms "cd ~/projects/backend/api/user-service"
Enter
Sleep 1s

Type@200ms "glf ."
Enter
Sleep 2s

# Show it opened
Type@30ms "# Opened in browser: https://gitlab.company.com/backend/api/user-service"
Enter
Sleep 2s

Type@100ms "clear"
Enter
Sleep 1s

Type@100ms "# GitHub: github.com/igusev/glf"
Enter
Sleep 4s

Type@100ms "clear"
Enter
Sleep 1.5s
