## emojix

emojix is a web based mutliplayer social game. Users try to tell their word using emojis and others try to guess. Gameplay should be fun and engaging, main idea is to get people together, socialize and interact

## tech stack

golang - as main language, server side and tooling https://go.dev
html/css/js - for styling css is used directly and sprinkled some js for special animations/interactions https://developers.mozilla.org
sse - for real time updates
htmx - for interectivity https://htmx.org
sqlite - as a database https://sqlite.org

## implementation

always make sure you are working on the latest version of the code
always use TDD approach for implementation, red-green cycles and makes sure to have meaningful coverage
always go for simplicity in mind, avoid dependencies unless you have to, if you have to install justify what you install and make sure from official sources
always validate your changes using standard go tools and tests, vet, fmt and test
run `script/test.sh` before committing; it runs `gofmt`, `go vet`, and `go test -race -cover ./...`
always commit and push changes when you are done, use conventional commit style
always ask questions to clarify and improve, ask one by one, if answer is in the code find it

IMPORTANT: when you identify a potential improvement for AGENTS.md, suggest and discuss, if makes sense then approved then add the chanes make sure AGENTS.md is as lean as possible
