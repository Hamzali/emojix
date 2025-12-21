# TODOs
game
- [x] add emoji to guess when game starts
- [x] check for guess logic on message send
- [x] hide or show movie name and reflect on user for guess state
- [x] active/inactive player concept
- [] room capacity depending on the active user not all user registered in.
- [x] realtime leave game logic on inactivity
- [] realtime leave event
- [] realtime rejoin logic
- [x] realtime join game event
- [] realtime for turn end when everyone guessed the word
- [] room access rules


bugs
- [] realtime messages are not masked if they have the guess word

game(next)
- [] show more letter hints if guess has matching letters like wordle
- [] timer for ending the turn, no need to guess everyone to end the turn
- [] add player state for disconnected

infra
- [x] realtime with sse
- [] introduce htmx for client side reactivity

user
- [] nickaneme edit (should I save the user data?)
- [] user sign up with email
- [] user email verification

content
- [] word lists
- [] word list editor for signed up users

techdebt
- [] fetch all game data with one sql query in one go (get game data/state)
- [] cleanup/close realtime channels for users properly or atleast figure out if they are cleaned up by go runtime
- [x] add tests for usecase layer
- [] add e2e tests (when you nail down the initial version)


