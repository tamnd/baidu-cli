# Demo tape for baidu. Rendered with ascii-gif (github.com/tamnd/ascii-gif),
# which supplies the window chrome and theme; this file is just the action.
#
#   ascii-gif render docs/demo/baidu.tape -o docs/static/demo.gif
#
# baidu must be on PATH inside the recording shell.

Hide
Type "export PS1='$ ' PATH=/tmp:$PATH"
Enter
Type "clear"
Enter
Show

Sleep 500ms
Type "baidu hot --fields rank,word,tag -n 8"
Sleep 800ms
Enter
Sleep 3.5s

Type "baidu suggest 百度 -o jsonl | jq -r '.word'"
Sleep 800ms
Enter
Sleep 4s
